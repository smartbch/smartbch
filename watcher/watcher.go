package watcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/crosschain"
	cctypes "github.com/smartbch/smartbch/crosschain/types"
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

	latestFinalizedHeight int64

	heightToFinalizedBlock map[int64]*types.BCHBlock

	EpochChan chan *stakingtypes.Epoch
	epochList []*stakingtypes.Epoch

	MonitorVoteChan     chan *cctypes.MonitorVoteInfo
	monitorVoteInfoList []*cctypes.MonitorVoteInfo

	voteInfoList []*types.VoteInfo

	numBlocksInEpoch   int64
	lastEpochEndHeight int64
	lastKnownEpochNum  int64

	numBlocksToClearMemory int
	waitingBlockDelayTime  int
	parallelNum            int

	chainConfig *param.ChainConfig

	currentMainnetBlockTimestamp int64

	//executors
	ccContractExecutor *crosschain.CcContractExecutor
}

func NewWatcher(logger log.Logger, historyDB modbtypes.DB, lastHeight, lastKnownEpochNum int64, chainConfig *param.ChainConfig) *Watcher {
	return &Watcher{
		logger: logger,

		rpcClient:         NewRpcClient(chainConfig.AppConfig.MainnetRPCUrl, chainConfig.AppConfig.MainnetRPCUsername, chainConfig.AppConfig.MainnetRPCPassword, "text/plain;", historyDB, logger),
		smartBchRpcClient: NewRpcClient(chainConfig.AppConfig.SmartBchRPCUrl, "", "", "application/json", nil, logger),

		lastEpochEndHeight:    lastHeight,
		latestFinalizedHeight: lastHeight,
		lastKnownEpochNum:     lastKnownEpochNum,

		heightToFinalizedBlock: make(map[int64]*types.BCHBlock),

		EpochChan:           make(chan *stakingtypes.Epoch, 10000),
		MonitorVoteChan:     make(chan *cctypes.MonitorVoteInfo, 5000),
		monitorVoteInfoList: make([]*cctypes.MonitorVoteInfo, 0, 10),

		voteInfoList: make([]*types.VoteInfo, 0, 10),

		numBlocksInEpoch:       param.StakingNumBlocksInEpoch,
		numBlocksToClearMemory: NumBlocksToClearMemory,
		waitingBlockDelayTime:  WaitingBlockDelayTime,

		parallelNum: 10,
		chainConfig: chainConfig,
		// set big enough for single node startup when no BCH node connected. it will be updated when mainnet block finalize.
		currentMainnetBlockTimestamp: math.MaxInt64 - 14*24*3600,
	}
}

func (watcher *Watcher) SetCCExecutor(exe *crosschain.CcContractExecutor) {
	watcher.ccContractExecutor = exe
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
	if watcher.rpcClient == (*RpcClient)(nil) {
		//for ut
		catchupChan <- true
		return
	}
	latestFinalizedHeight := watcher.latestFinalizedHeight
	latestMainnetHeight := watcher.rpcClient.GetLatestHeight(true)
	latestFinalizedHeight = watcher.speedup(latestFinalizedHeight, latestMainnetHeight)
	go watcher.CollectCCTransferInfos()
	watcher.fetchBlocks(catchupChan, latestFinalizedHeight, latestMainnetHeight)
}

func (watcher *Watcher) fetchBlocks(catchupChan chan bool, latestFinalizedHeight, latestMainnetHeight int64) {
	catchup := false
	for {
		if !catchup && latestMainnetHeight <= latestFinalizedHeight+9 {
			latestMainnetHeight = watcher.rpcClient.GetLatestHeight(true)
			if latestMainnetHeight <= latestFinalizedHeight+9 {
				watcher.logger.Debug("Catchup")
				catchup = true
				catchupChan <- true
				close(catchupChan)
			}
		}
		latestFinalizedHeight++
		latestMainnetHeight = watcher.rpcClient.GetLatestHeight(true)
		//10 confirms
		if latestMainnetHeight < latestFinalizedHeight+9 {
			watcher.logger.Debug("waiting BCH mainnet", "height now is", latestMainnetHeight)
			watcher.suspended(time.Duration(watcher.waitingBlockDelayTime) * time.Second) //delay half of bch mainnet block intervals
			latestFinalizedHeight--
			continue
		}
		for latestFinalizedHeight+9 <= latestMainnetHeight {
			fmt.Printf("latestFinalizedHeight:%d,latestMainnetHeight:%d\n", latestFinalizedHeight, latestMainnetHeight)
			if latestFinalizedHeight+9+int64(watcher.parallelNum) <= latestMainnetHeight {
				watcher.parallelFetchBlocks(latestFinalizedHeight)
				latestFinalizedHeight += int64(watcher.parallelNum)
			} else {
				blk := watcher.rpcClient.GetBlockByHeight(latestFinalizedHeight, true)
				if blk == nil {
					//todo: panic it
					fmt.Printf("get block:%d failed\n", latestFinalizedHeight)
					latestFinalizedHeight--
					continue
				}
				watcher.addFinalizedBlock(blk)
				latestFinalizedHeight++
			}
		}
		latestFinalizedHeight--
	}
}

func (watcher *Watcher) parallelFetchBlocks(latestFinalizedHeight int64) {
	fmt.Printf("begin paralell fetch blocks\n")
	var blockSet = make([]*types.BCHBlock, watcher.parallelNum)
	var w sync.WaitGroup
	w.Add(watcher.parallelNum)
	for i := 0; i < watcher.parallelNum; i++ {
		go func(index int) {
			blockSet[index] = watcher.rpcClient.GetBlockByHeight(latestFinalizedHeight+int64(index), true)
			w.Done()
		}(i)
	}
	w.Wait()
	fmt.Printf("after paralell fetch blocks\n")
	for _, blk := range blockSet {
		watcher.addFinalizedBlock(blk)
	}
	watcher.logger.Debug("Get bch mainnet block", "latestFinalizedHeight", latestFinalizedHeight)
}

func (watcher *Watcher) speedup(latestFinalizedHeight, latestMainnetHeight int64) int64 {
	if watcher.chainConfig.AppConfig.Speedup {
		start := uint64(watcher.lastKnownEpochNum) + 1
		for {
			infos := watcher.smartBchRpcClient.GetVoteInfoByEpochNumber(start, start+100)
			if len(infos) == 0 {
				break
			}
			debugVoteInfo(infos)
			watcher.voteInfoList = append(watcher.voteInfoList, infos...)
			for _, in := range infos {
				if in.Epoch.EndTime != 0 {
					watcher.EpochChan <- &in.Epoch
				}
				if !param.IsAmber && in.MonitorVote.EndTime != 0 {
					watcher.MonitorVoteChan <- &in.MonitorVote
				}
			}
			latestFinalizedHeight += int64(len(infos)) * watcher.numBlocksInEpoch
			start = start + uint64(len(infos))
		}
		watcher.latestFinalizedHeight = latestFinalizedHeight
		watcher.lastEpochEndHeight = latestFinalizedHeight
		watcher.logger.Debug("After speedup", "latestFinalizedHeight", watcher.latestFinalizedHeight)
	}
	return latestFinalizedHeight
}

func debugVoteInfo(infos []*types.VoteInfo) {
	for _, e := range infos {
		out, _ := json.Marshal(e.Epoch)
		fmt.Println(string(out))
		out, _ = json.Marshal(e.MonitorVote)
		fmt.Println(string(out))
	}
}

func (watcher *Watcher) suspended(delayDuration time.Duration) {
	time.Sleep(delayDuration)
}

// Record new block and if the blocks for a new epoch is all ready, output the new epoch
func (watcher *Watcher) addFinalizedBlock(blk *types.BCHBlock) {
	watcher.heightToFinalizedBlock[blk.Height] = blk
	watcher.latestFinalizedHeight++
	watcher.currentMainnetBlockTimestamp = blk.Timestamp

	if watcher.latestFinalizedHeight-watcher.lastEpochEndHeight == watcher.numBlocksInEpoch {
		watcher.generateNewEpoch()
	}
}

// Generate a new block's information
func (watcher *Watcher) generateNewEpoch() {
	epoch := watcher.buildNewEpoch()
	watcher.logger.Debug("Generate new epoch", "epochNumber", epoch.Number, "startHeight", epoch.StartHeight)
	watcher.EpochChan <- epoch
	info := watcher.buildMonitorVoteInfo()
	if info != nil {
		watcher.MonitorVoteChan <- info
	}
	var voteInfo types.VoteInfo
	voteInfo.Epoch = *epoch
	if info != nil {
		voteInfo.MonitorVote = *info
	}
	watcher.voteInfoList = append(watcher.voteInfoList, &voteInfo)
	watcher.lastEpochEndHeight = watcher.latestFinalizedHeight
	watcher.ClearOldData()
}

func (watcher *Watcher) buildMonitorVoteInfo() *cctypes.MonitorVoteInfo {
	startHeight := watcher.lastEpochEndHeight + 1
	if startHeight < param.EpochStartHeightForCC {
		return nil
	}
	var info cctypes.MonitorVoteInfo
	var monitorMapByPubkey = make(map[[33]byte]*cctypes.Nomination)

	for i := startHeight; i <= watcher.latestFinalizedHeight; i++ {
		blk, ok := watcher.heightToFinalizedBlock[i]
		if !ok {
			panic("Missing Block")
		}
		for _, ccNomination := range blk.CCNominations {
			if _, ok := monitorMapByPubkey[ccNomination.Pubkey]; !ok {
				monitorMapByPubkey[ccNomination.Pubkey] = &ccNomination
			}
			monitorMapByPubkey[ccNomination.Pubkey].NominatedCount += ccNomination.NominatedCount
		}
	}
	for _, v := range monitorMapByPubkey {
		info.Nominations = append(info.Nominations, v)
	}
	crosschain.SortMonitorVoteNominations(info.Nominations)
	return &info
}

func (watcher *Watcher) buildNewEpoch() *stakingtypes.Epoch {
	epoch := &stakingtypes.Epoch{
		StartHeight: watcher.lastEpochEndHeight + 1,
		Nominations: make([]*stakingtypes.Nomination, 0, 10),
	}
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

func (watcher *Watcher) GetCurrEpoch() *stakingtypes.Epoch {
	return watcher.buildNewEpoch()
}
func (watcher *Watcher) GetEpochList() []*stakingtypes.Epoch {
	list := stakingtypes.CopyEpochs(watcher.epochList)
	currEpoch := watcher.buildNewEpoch()
	return append(list, currEpoch)
}

func (watcher *Watcher) GetCurrMainnetBlockTimestamp() int64 {
	return watcher.currentMainnetBlockTimestamp
}

func (watcher *Watcher) CheckSanity(skipCheck bool) {
	if !skipCheck {
		latestHeight := watcher.rpcClient.GetLatestHeight(false)
		if latestHeight <= 0 {
			panic("Watcher GetLatestHeight failed in Sanity Check")
		}
		blk := watcher.rpcClient.GetBlockByHeight(latestHeight, false)
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
	vLen := len(watcher.voteInfoList)
	if vLen == 0 {
		return
	}
	height := watcher.voteInfoList[vLen-1].Epoch.StartHeight
	height -= 5 * watcher.numBlocksInEpoch
	for {
		_, ok := watcher.heightToFinalizedBlock[height]
		if !ok {
			break
		}
		delete(watcher.heightToFinalizedBlock, height)
		height--
	}
	if vLen > 5 /*param it*/ {
		watcher.voteInfoList = watcher.voteInfoList[vLen-5:]
	}
}

func (watcher *Watcher) CollectCCTransferInfos() {
	var beginBlockHeight, endBlockHeight int64
	for {
		if watcher.latestFinalizedHeight < param.EpochStartHeightForCC {
			time.Sleep(5 * time.Second)
			continue
		}
		heightInfo := <-watcher.ccContractExecutor.StartUTXOCollect
		beginBlockHeight = heightInfo.BeginHeight
		endBlockHeight = heightInfo.EndHeight
		for {
			//todo: add lock
			endBlock := watcher.heightToFinalizedBlock[endBlockHeight]
			if endBlock == nil {
				watcher.suspended(15 * time.Second)
				continue
			}
			watcher.ccContractExecutor.Infos = nil
			for h := beginBlockHeight; h <= endBlockHeight; h++ {
				blk := watcher.heightToFinalizedBlock[h]
				watcher.ccContractExecutor.Infos = append(watcher.ccContractExecutor.Infos, blk.CCTransferInfos...)
			}
			watcher.ccContractExecutor.UTXOCollectDone <- true
			break
		}
	}
}
