package staking

import (
	"github.com/smartbch/smartbch/staking/types"
	"time"
)

var (
	NumBlocksInEpoch int64 = 2016
)

// A watcher watches the new blocks generated on bitcoin cash's mainnet, and
// outputs epoch information through a channel
type Watcher struct {
	lastEpochEndHeight     int64
	latestFinalizedHeight  int64
	hashToBlock            map[[32]byte]*types.BCHBlock
	heightToFinalizedBlock map[int64]*types.BCHBlock
	epochList              []*types.Epoch
	rpcClient              types.RpcClient
	EpochChan              chan *types.Epoch
}

// A new watch will start watching from lastHeight+1, using rpcClient
func NewWatcher(lastHeight int64, rpcClient types.RpcClient) *Watcher {
	return &Watcher{
		lastEpochEndHeight:     lastHeight,
		latestFinalizedHeight:  lastHeight,
		hashToBlock:            make(map[[32]byte]*types.BCHBlock),
		heightToFinalizedBlock: make(map[int64]*types.BCHBlock),
		epochList:              make([]*types.Epoch, 0, 10),
		rpcClient:              rpcClient,
		EpochChan:              make(chan *types.Epoch, 1),
	}
}

// The main function to do a watcher's job. It must be run as a goroutine
func (watcher *Watcher) Run() {
	height := watcher.latestFinalizedHeight
	//todo: for debug temp
	if watcher.rpcClient == nil {
		return
	}
	watcher.rpcClient.Dial()
	for {
		height++ // to fetch the next block
		blk := watcher.rpcClient.GetBlockByHeight(height)
		if blk == nil { //make sure connected BCH mainnet node not pruning history blocks, so this case only means height is latest block
			watcher.suspended(5 * time.Minute) //delay half of bch mainnet block intervals
		}
		missingBlockHash := watcher.addBlock(blk)
		for missingBlockHash != nil { // if chain reorg happens, we trace the new tip
			blk = watcher.rpcClient.GetBlockByHash(*missingBlockHash)
			if blk == nil {
				panic("BCH mainnet tip should has its parent block")
			}
			missingBlockHash = watcher.addBlock(blk)
		}
	}
}

func (watcher *Watcher) suspended(delayDuration time.Duration) {
	watcher.rpcClient.Close()
	time.Sleep(delayDuration)
	watcher.rpcClient.Dial()
}

// Record new block and if the blocks for a new epoch is all ready, output the new epoch
func (watcher *Watcher) addBlock(blk *types.BCHBlock) (missingBlockHash *[32]byte) {
	watcher.hashToBlock[blk.HashId] = blk
	parent, ok := watcher.hashToBlock[blk.ParentBlk]
	if !ok {
		// If parent is init block, return directly
		if blk.ParentBlk == [32]byte{} {
			return nil
		}
		return &blk.ParentBlk
	}
	// On BCH mainnet, a block need 10 confirmation to finalize
	var grandpa *types.BCHBlock
	for confirmCount := 1; confirmCount < 10; confirmCount++ {
		grandpa, ok = watcher.hashToBlock[parent.ParentBlk]
		if !ok {
			return &parent.ParentBlk // actually impossible to reach here
		}
		parent = grandpa
	}
	finalizedBlk, ok := watcher.heightToFinalizedBlock[parent.Height]
	if ok {
		if finalizedBlk.Equal(parent) {
			return nil //nothing to do
		} else {
			panic("Deep Reorganization")
		}
	}
	// A new block is finalized
	watcher.heightToFinalizedBlock[parent.Height] = grandpa
	if watcher.latestFinalizedHeight+1 != parent.Height {
		panic("Height Skipped")
	}
	watcher.latestFinalizedHeight++
	// All the blocks for an epoch is ready
	if watcher.latestFinalizedHeight-watcher.lastEpochEndHeight == NumBlocksInEpoch {
		watcher.generateNewEpoch()
	}
	return nil
}

// Generate a new block's information
func (watcher *Watcher) generateNewEpoch() {
	epoch := &types.Epoch{
		StartHeight:    watcher.lastEpochEndHeight + 1,
		ValMapByPubkey: make(map[[32]byte]*types.Nomination),
	}
	startTime := int64(1 << 62)
	for i := epoch.StartHeight; i <= watcher.latestFinalizedHeight; i++ {
		blk, ok := watcher.heightToFinalizedBlock[i]
		if !ok {
			panic("Missing Block")
		}
		if epoch.EndTime < blk.Timestamp {
			epoch.EndTime = blk.Timestamp
		}
		if startTime > blk.Timestamp {
			startTime = blk.Timestamp
		}
		for _, nomination := range blk.Nominations {
			if _, ok := epoch.ValMapByPubkey[nomination.Pubkey]; !ok {
				epoch.ValMapByPubkey[nomination.Pubkey] = &nomination
			}
			epoch.ValMapByPubkey[nomination.Pubkey].NominatedCount++
		}
	}
	epoch.Duration = epoch.EndTime - startTime
	if len(watcher.epochList) != 0 {
		lastEpoch := watcher.epochList[len(watcher.epochList)-1]
		epoch.Duration = epoch.EndTime - lastEpoch.EndTime
	}
	watcher.epochList = append(watcher.epochList, epoch)
	watcher.EpochChan <- epoch
	watcher.lastEpochEndHeight = watcher.latestFinalizedHeight
	watcher.ClearOldData()
}

func (watcher *Watcher) ClearOldData() {
	elLen := len(watcher.epochList)
	if elLen == 0 {
		return
	}
	height := watcher.epochList[elLen-1].StartHeight
	height -= 5 * NumBlocksInEpoch
	for {
		blk, ok := watcher.heightToFinalizedBlock[height]
		if !ok {
			break
		}
		delete(watcher.heightToFinalizedBlock, height)
		delete(watcher.hashToBlock, blk.HashId)
		height--
	}
	if elLen > 5 /*param it*/ {
		watcher.epochList = watcher.epochList[elLen-5:]
	}
}
