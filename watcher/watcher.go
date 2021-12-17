package watcher

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/param"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
	"github.com/smartbch/smartbch/watcher/types"
)

const (
	NumBlocksToClearMemory = 1000
	WaitingBlockDelayTime  = 2
)

// A watcher watches the new blocks generated on bitcoin cash's mainnet, and
// outputs epoch information through a channel
type Watcher struct {
	logger log.Logger

	rpcClient         types.RpcClient
	smartBchRpcClient types.RpcClient

	EpochChan chan *stakingtypes.Epoch

	lastEpochEndHeight    int64
	latestFinalizedHeight int64
	lastKnownEpochNum     int64

	hashToBlock            map[[32]byte]*types.BCHBlock
	heightToFinalizedBlock map[int64]*types.BCHBlock
	epochList              []*stakingtypes.Epoch

	speedup bool

	numBlocksInEpoch       int64
	numBlocksToClearMemory int
	waitingBlockDelayTime  int
}

// A new watch will start watching from lastHeight+1, using rpcClient
func NewWatcher(logger log.Logger, lastHeight int64, rpcClient types.RpcClient, smartBchUrl string, lastKnownEpochNum int64, speedup bool) *Watcher {
	return &Watcher{
		logger: logger,

		rpcClient:         rpcClient,
		smartBchRpcClient: NewRpcClient(smartBchUrl, "", "", "application/json", logger),

		lastEpochEndHeight:    lastHeight,
		latestFinalizedHeight: lastHeight,
		lastKnownEpochNum:     lastKnownEpochNum,

		hashToBlock:            make(map[[32]byte]*types.BCHBlock),
		heightToFinalizedBlock: make(map[int64]*types.BCHBlock),
		epochList:              make([]*stakingtypes.Epoch, 0, 10),

		EpochChan: make(chan *stakingtypes.Epoch, 10000),
		speedup:   speedup,

		numBlocksInEpoch:       param.StakingNumBlocksInEpoch,
		numBlocksToClearMemory: NumBlocksToClearMemory,
		waitingBlockDelayTime:  WaitingBlockDelayTime,
	}
}

func (watcher *Watcher) SetNumBlocksInEpoch(n int64) {
	watcher.numBlocksInEpoch = n
}

func (watcher *Watcher) SetNumBlocksToClearMemory(n int) {
	watcher.numBlocksToClearMemory = n
}

func (watcher *Watcher) SetWaitingBlockDelayTime(n int) {
	watcher.waitingBlockDelayTime = n
}

// The main function to do a watcher's job. It must be run as a goroutine
func (watcher *Watcher) Run(catchupChan chan bool) {
	height := watcher.latestFinalizedHeight
	if watcher.rpcClient == nil {
		return
	}
	latestHeight := watcher.rpcClient.GetLatestHeight()
	catchup := false
	if watcher.speedup {
		start := uint64(watcher.lastKnownEpochNum) + 1
		for {
			if latestHeight < height+watcher.numBlocksInEpoch {
				break
			}
			epochs := watcher.smartBchRpcClient.GetEpochs(start, start+100)
			if len(epochs) == 0 {
				break
			}
			for _, e := range epochs {
				out, _ := json.Marshal(e)
				fmt.Println(string(out))
			}
			watcher.epochList = append(watcher.epochList, epochs...)
			for _, e := range epochs {
				watcher.EpochChan <- e
			}
			height += int64(len(epochs)) * watcher.numBlocksInEpoch
			start = start + uint64(len(epochs))
		}
		watcher.latestFinalizedHeight = height
		watcher.lastEpochEndHeight = height
		watcher.logger.Debug("After speedup", "latestFinalizedHeight", watcher.latestFinalizedHeight)
	}
	for {
		if !catchup && latestHeight <= height {
			latestHeight = watcher.rpcClient.GetLatestHeight()
			if latestHeight <= height {
				watcher.logger.Debug("Catchup")
				catchup = true
				catchupChan <- true
				close(catchupChan)
			}
		}
		height++
		blk := watcher.rpcClient.GetBlockByHeight(height)
		if blk == nil { //make sure connected BCH mainnet node not pruning history blocks, so this case only means height is latest block
			watcher.suspended(time.Duration(watcher.waitingBlockDelayTime) * time.Second) //delay half of bch mainnet block intervals
			height--
			continue
		}
		watcher.logger.Debug("Get bch mainnet block", "height", height)
		missingBlockHash := watcher.addBlock(blk)
		if missingBlockHash == nil {
			// release blocks left to prevent BCH mainnet forks take too much memory
			if height%int64(watcher.numBlocksToClearMemory) == 0 {
				watcher.hashToBlock = make(map[[32]byte]*types.BCHBlock)
			}
			continue
		}
		// follow the forked tip to avoid finalize block empty hole
		for i := 10; missingBlockHash != nil && i > 0; i-- { // if chain reorg happens, we trace the new tip
			watcher.logger.Debug("Get missing block", "hash", hex.EncodeToString(missingBlockHash[:]))
			prevBlk := watcher.rpcClient.GetBlockByHash(*missingBlockHash)
			if prevBlk == nil {
				panic("BCH mainnet tip should has its parent block")
			}
			missingBlockHash = watcher.addBlock(prevBlk)
		}
		// when we get the forked full branch, we try to add the tip again
		missingBlockHash = watcher.addBlock(blk)
		if missingBlockHash != nil {
			panic(fmt.Sprintf("The parent %s must not be missing", hex.EncodeToString(missingBlockHash[:])))
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
		// If parent is the genesis block, return directly
		if blk.ParentBlk == [32]byte{} {
			return nil
		}
		watcher.logger.Debug("Missing parent block", "parent hash", hex.EncodeToString(blk.ParentBlk[:]), "parent height", blk.Height-1)
		return &blk.ParentBlk
	}
	// On BCH mainnet, a block need 10 confirmation to finalize
	var grandpa *types.BCHBlock
	for confirmCount := 1; confirmCount < 10; confirmCount++ {
		grandpa, ok = watcher.hashToBlock[parent.ParentBlk]
		if !ok {
			if parent.ParentBlk == [32]byte{} {
				return nil //met genesis in less than 10, nothing to do
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
	if watcher.latestFinalizedHeight-watcher.lastEpochEndHeight == watcher.numBlocksInEpoch {
		watcher.generateNewEpoch()
	}
	return nil
}

// Generate a new block's information
func (watcher *Watcher) generateNewEpoch() {
	epoch := watcher.buildNewEpoch()
	watcher.epochList = append(watcher.epochList, epoch)
	watcher.logger.Debug("Generate new epoch", "epochNumber", epoch.Number, "startHeight", epoch.StartHeight)
	watcher.EpochChan <- epoch
	watcher.lastEpochEndHeight = watcher.latestFinalizedHeight
	watcher.ClearOldData()
}

func (watcher *Watcher) buildNewEpoch() *stakingtypes.Epoch {
	epoch := &stakingtypes.Epoch{
		StartHeight: watcher.lastEpochEndHeight + 1,
		Nominations: make([]*stakingtypes.Nomination, 0, 10),
	}
	startTime := int64(1 << 62)
	var valMapByPubkey = make(map[[32]byte]*stakingtypes.Nomination)
	for i := epoch.StartHeight; i <= watcher.latestFinalizedHeight; i++ {
		blk, ok := watcher.heightToFinalizedBlock[i]
		if !ok {
			panic("Missing Block")
		}
		//Please note that BCH's timestamp is not always linearly increasing
		if epoch.EndTime < blk.Timestamp {
			epoch.EndTime = blk.Timestamp
		}
		if startTime > blk.Timestamp {
			startTime = blk.Timestamp
		}
		for _, nomination := range blk.Nominations {
			if _, ok := valMapByPubkey[nomination.Pubkey]; !ok {
				valMapByPubkey[nomination.Pubkey] = &nomination
			}
			valMapByPubkey[nomination.Pubkey].NominatedCount += nomination.NominatedCount
		}
	}
	for _, v := range valMapByPubkey {
		epoch.Nominations = append(epoch.Nominations, v)
	}
	sortEpochNominations(epoch)
	return epoch
}

func (watcher *Watcher) CheckSanity(forTest bool) {
	if !forTest {
		latestHeight := watcher.rpcClient.GetLatestHeight()
		if latestHeight <= 0 {
			panic("Watcher GetLatestHeight failed in Sanity Check")
		}
		blk := watcher.rpcClient.GetBlockByHeight(latestHeight)
		if blk == nil {
			panic("Watcher GetBlockByHeight failed in Sanity Check")
		}
	}
}

//sort by pubkey (small to big) first; then sort by nominationCount;
//so nominations sort by NominationCount, if count is equal, smaller pubkey stand front
func sortEpochNominations(epoch *stakingtypes.Epoch) {
	sort.Slice(epoch.Nominations, func(i, j int) bool {
		return bytes.Compare(epoch.Nominations[i].Pubkey[:], epoch.Nominations[j].Pubkey[:]) < 0
	})
	sort.SliceStable(epoch.Nominations, func(i, j int) bool {
		return epoch.Nominations[i].NominatedCount > epoch.Nominations[j].NominatedCount
	})
}

func (watcher *Watcher) ClearOldData() {
	elLen := len(watcher.epochList)
	if elLen == 0 {
		return
	}
	height := watcher.epochList[elLen-1].StartHeight
	for hash, blk := range watcher.hashToBlock {
		if blk.Height < height {
			delete(watcher.hashToBlock, hash)
		}
	}
	height -= 5 * watcher.numBlocksInEpoch
	for {
		_, ok := watcher.heightToFinalizedBlock[height]
		if !ok {
			break
		}
		delete(watcher.heightToFinalizedBlock, height)
		height--
	}
	if elLen > 5 /*param it*/ {
		watcher.epochList = watcher.epochList[elLen-5:]
	}
}
