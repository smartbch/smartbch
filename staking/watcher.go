package staking

import (
	"time"

	"github.com/moeing-chain/moeing-chain/staking/types"
)

const (
	NumBlocksInEpoch int64 = 2016
)

type Watcher struct {
	lastEpochEndHeight     int64
	latestFinalizedHeight  int64
	hashToBlock            map[[32]byte]*types.BCHBlock
	heightToFinalizedBlock map[int64]*types.BCHBlock
	epochList              []*types.Epoch
	rpcClient              types.RpcClient
	EpochChan              chan *types.Epoch
}

func NewWatcher(lastHeight int64, rpcClient types.RpcClient) *Watcher {
	return &Watcher{
		lastEpochEndHeight:     lastHeight,
		latestFinalizedHeight:  lastHeight,
		hashToBlock:            make(map[[32]byte]*types.BCHBlock),
		heightToFinalizedBlock: make(map[int64]*types.BCHBlock),
		epochList:              make([]*types.Epoch, 0, 10),
		rpcClient:              rpcClient,
		EpochChan:              make(chan *types.Epoch, 2),
	}
}

func (watcher *Watcher) Run(blk *types.BCHBlock) {
	height := watcher.lastEpochEndHeight + 1
	watcher.rpcClient.Dial()
	latestHeight := watcher.rpcClient.GetLatestHeight()
	for {
		if height > latestHeight {
			watcher.rpcClient.Close()
			time.Sleep(30 * time.Second)
			watcher.rpcClient.Dial()
		}
		blk := watcher.rpcClient.GetBlockByHeight(height)
		missingBlockHash := watcher.addBlock(blk)
		for missingBlockHash != nil {
			blk = watcher.rpcClient.GetBlockByHash(*missingBlockHash)
			missingBlockHash = watcher.addBlock(blk)
		}
		height++
	}
}

func (watcher *Watcher) addBlock(blk *types.BCHBlock) (missingBlockHash *[32]byte) {
	parent, ok := watcher.hashToBlock[blk.ParentBlk]
	if !ok {
		return &blk.ParentBlk
	}
	for confirmCount := 1; confirmCount < 10; confirmCount++ {
		parent, ok = watcher.hashToBlock[parent.ParentBlk]
		if !ok {
			panic("Blocken Chain")
		}
	}
	finalizedBlk, ok := watcher.heightToFinalizedBlock[parent.Height]
	if ok {
		if finalizedBlk == parent {
			return nil //nothing to do
		} else {
			panic("Deep Reorganization")
		}
	}
	watcher.heightToFinalizedBlock[parent.Height] = parent
	if watcher.latestFinalizedHeight+1 != parent.Height {
		panic("Height Skipped")
	}
	watcher.latestFinalizedHeight = parent.Height
	if watcher.latestFinalizedHeight-watcher.lastEpochEndHeight == NumBlocksInEpoch {
		watcher.analyzeNewEpoch()
	}
	return nil
}

func (watcher *Watcher) analyzeNewEpoch() {
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
}
