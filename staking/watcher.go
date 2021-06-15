package staking

import (
	"fmt"
	"time"

	"github.com/smartbch/smartbch/staking/types"
)

var (
	//NumBlocksInEpoch       int64 = 2016
	NumBlocksInEpoch       int64 = 10
	NumBlocksToClearMemory int64 = 100000
	//WaitingBlockDelayTime  int64 = 5 * 60
	WaitingBlockDelayTime int64 = 10
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
	for {
		height++ // to fetch the next block
		fmt.Println("try to get block height:", height)
		blk := watcher.rpcClient.GetBlockByHeight(height)
		if blk == nil { //make sure connected BCH mainnet node not pruning history blocks, so this case only means height is latest block
			fmt.Println("wait new block...")
			watcher.suspended(time.Duration(WaitingBlockDelayTime) * time.Second) //delay half of bch mainnet block intervals
			height--
			continue
		}
		fmt.Println("get bch mainnet block height: ", height)
		missingBlockHash := watcher.addBlock(blk)
		//get fork height again to avoid finalize block empty hole
		if missingBlockHash != nil {
			height--
		} else {
			// release blocks left as of BCH mainnet fork
			if height%NumBlocksToClearMemory == 0 {
				watcher.hashToBlock = make(map[[32]byte]*types.BCHBlock)
			}
		}
		for i := 10; missingBlockHash != nil && i > 0; i-- { // if chain reorg happens, we trace the new tip
			fmt.Println("get missing block:", missingBlockHash)
			blk = watcher.rpcClient.GetBlockByHash(*missingBlockHash)
			if blk == nil {
				panic("BCH mainnet tip should has its parent block")
			}
			missingBlockHash = watcher.addBlock(blk)
		}
	}
}

func (watcher *Watcher) suspended(delayDuration time.Duration) {
	time.Sleep(delayDuration)
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
		fmt.Println("parent hash:", blk.ParentBlk)
		fmt.Println("current block height:", blk.Height)
		return &blk.ParentBlk
	}
	// On BCH mainnet, a block need 10 confirmation to finalize
	var grandpa *types.BCHBlock
	for confirmCount := 1; confirmCount < 10; confirmCount++ {
		grandpa, ok = watcher.hashToBlock[parent.ParentBlk]
		if !ok {
			if parent.ParentBlk == [32]byte{} {
				return nil
			}
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
	// when node restart, watcher work from latestFinalizedHeight, which are already finalized,
	// so watcher.latestFinalizedHeight+1 greater than parent.Height here, we not increase the watcher.latestFinalizedHeight
	if watcher.latestFinalizedHeight+1 < parent.Height {
		panic("Height Skipped")
	} else if watcher.latestFinalizedHeight+1 == parent.Height {
		watcher.latestFinalizedHeight++
	}
	// All the blocks for an epoch is ready
	if watcher.latestFinalizedHeight-watcher.lastEpochEndHeight == NumBlocksInEpoch {
		fmt.Println("finalized height:", watcher.latestFinalizedHeight)
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
	fmt.Println("elLen:", elLen)
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
