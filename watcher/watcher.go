package watcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync/atomic"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingads/datatree"
	cctypes "github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/param"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
	"github.com/smartbch/smartbch/watcher/types"
)

const (
	NumBlocksToClearMemory = 1000
	WaitingBlockDelayTime  = 2
	blockFinalizeNumber    = 9
)

// A watcher watches the new blocks generated on bitcoin cash's mainnet, and
// outputs epoch information through a channel
type Watcher struct {
	logger log.Logger

	rpcClient         types.RpcClient
	smartBchRpcClient types.RpcClient

	latestFinalizedHeight int64

	heightToFinalizedBlock map[int64]*types.BCHBlock

	catchupChan chan bool

	EpochChan          chan *stakingtypes.Epoch
	epochList          []*stakingtypes.Epoch
	numBlocksInEpoch   int64
	lastEpochEndHeight int64
	lastKnownEpochNum  int64

	CCEpochChan          chan *cctypes.CCEpoch
	lastCCEpochEndHeight int64
	numBlocksInCCEpoch   int64
	ccEpochList          []*cctypes.CCEpoch
	lastKnownCCEpochNum  int64

	numBlocksToClearMemory int
	waitingBlockDelayTime  int
	parallelNum            int

	chainConfig *param.ChainConfig

	currentMainnetBlockTimestamp int64
}

func NewWatcher(logger log.Logger, lastHeight, lastCCEpochEndHeight int64, lastKnownEpochNum int64, chainConfig *param.ChainConfig) *Watcher {
	w := &Watcher{
		logger:            logger,
		smartBchRpcClient: NewRpcClient(chainConfig.AppConfig.SmartBchRPCUrl, "", "", "application/json", logger),

		lastEpochEndHeight:    lastHeight,
		latestFinalizedHeight: lastHeight,
		lastKnownEpochNum:     lastKnownEpochNum,

		catchupChan: make(chan bool, 1),

		heightToFinalizedBlock: make(map[int64]*types.BCHBlock),
		epochList:              make([]*stakingtypes.Epoch, 0, 10),

		EpochChan: make(chan *stakingtypes.Epoch, 10000),

		numBlocksInEpoch:       param.StakingNumBlocksInEpoch,
		numBlocksToClearMemory: NumBlocksToClearMemory,
		waitingBlockDelayTime:  WaitingBlockDelayTime,

		CCEpochChan:          make(chan *cctypes.CCEpoch, 96*10000),
		ccEpochList:          make([]*cctypes.CCEpoch, 0, 40),
		lastCCEpochEndHeight: lastCCEpochEndHeight,
		numBlocksInCCEpoch:   param.BlocksInCCEpoch,

		parallelNum: 10,
		chainConfig: chainConfig,
		// set big enough for single node startup when no BCH node connected. it will be updated when mainnet block finalize.
		currentMainnetBlockTimestamp: math.MaxInt64 - 14*24*3600,
	}
	if !chainConfig.AppConfig.DisableBchClient {
		w.rpcClient = NewRpcClient(chainConfig.AppConfig.MainnetRPCUrl, chainConfig.AppConfig.MainnetRPCUsername, chainConfig.AppConfig.MainnetRPCPassword, "text/plain;", logger)
	}
	return w
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

func (watcher *Watcher) WaitCatchup() {
	<-watcher.catchupChan
}

// The main function to do a watcher's job. It must be run as a goroutine
func (watcher *Watcher) Run() {
	if watcher.rpcClient == (*RpcClient)(nil) {
		//for ut
		if !watcher.chainConfig.AppConfig.DisableBchClient {
			watcher.catchupChan <- true
			return
		}
	}
	if watcher.chainConfig.AppConfig.DisableBchClient {
		// must connect a smartbchd client which tip height higher than staking upgrade,
		// which has all epochs in pow + pos mode.
		watcher.speedupInBchClientDisableMode()
		return
	}
	watcher.speedup()
	watcher.fetchBlocks()
}

func (watcher *Watcher) fetchBlocks() {
	catchedUp := false
	latestMainnetHeight := watcher.rpcClient.GetLatestHeight(true)
	heightWanted := watcher.latestFinalizedHeight + 1
	// parallel fetch blocks when startup
	if heightWanted+blockFinalizeNumber+int64(watcher.parallelNum) <= latestMainnetHeight {
		watcher.logger.Debug("block parallel fetch info", "latestFinalizedHeight", watcher.latestFinalizedHeight, "latestMainnetHeight", latestMainnetHeight)
		watcher.parallelFetchBlocks(heightWanted, latestMainnetHeight-blockFinalizeNumber)
		heightWanted = watcher.latestFinalizedHeight + 1
	}
	// normal catchup
	for {
		latestMainnetHeight = watcher.rpcClient.GetLatestHeight(true)
		for heightWanted+blockFinalizeNumber <= latestMainnetHeight {
			watcher.addFinalizedBlock(watcher.rpcClient.GetBlockByHeight(heightWanted, true))
			heightWanted++
			latestMainnetHeight = watcher.rpcClient.GetLatestHeight(true)
		}
		if catchedUp {
			watcher.logger.Debug("waiting BCH mainnet", "height now is", latestMainnetHeight)
			watcher.suspended(time.Duration(watcher.waitingBlockDelayTime) * time.Second) //delay half of bch mainnet block intervals
		} else {
			watcher.logger.Debug("AlreadyCaughtUp")
			catchedUp = true
			close(watcher.catchupChan)
		}
	}
}

func (watcher *Watcher) parallelFetchBlocks(heightStart, heightEnd int64) {
	var blockSet = make([]*types.BCHBlock, heightEnd-heightStart+1)
	sharedIdx := int64(-1)
	datatree.ParallelRun(watcher.parallelNum, func(_ int) {
		for {
			index := atomic.AddInt64(&sharedIdx, 1)
			if heightStart+index > heightEnd {
				break
			}
			blockSet[index] = watcher.rpcClient.GetBlockByHeight(heightStart+index, true)
		}
	})
	for _, blk := range blockSet {
		watcher.addFinalizedBlock(blk)
	}
	watcher.logger.Debug("Get bch mainnet blocks parallel", "latestFinalizedHeight", watcher.latestFinalizedHeight)
}

func (watcher *Watcher) speedupInBchClientDisableMode() {
	// no need get epochs if lastKnownCCEpochNum >= 50
	if watcher.lastKnownEpochNum >= 50 {
		close(watcher.catchupChan)
		return
	}
	if watcher.smartBchRpcClient == (*RpcClient)(nil) {
		panic("must provide valid smartbchd node info for epoch fetch")
	}
	start := uint64(watcher.lastKnownEpochNum) + 1
	for {
		epochs := watcher.smartBchRpcClient.GetEpochs(start, start+100)
		watcher.epochList = append(watcher.epochList, epochs...)
		watcher.lastKnownEpochNum += int64(len(epochs))
		for _, e := range epochs {
			if e.EndTime != 0 {
				watcher.EpochChan <- e
			}
			out, _ := json.Marshal(e)
			fmt.Println(string(out))
		}
		if len(epochs) < 100 {
			break
		}
		start = start + 100
	}
	watcher.logger.Debug("After speedup in bchClientDisabled mode", "lastKnownEpochNum", watcher.lastKnownEpochNum)
	if watcher.lastKnownEpochNum < 50 { // todo: param 50
		panic("must get epoch 50 when run in bchClientDisabled mode, please try to connect another smartbchd node")
	}
	close(watcher.catchupChan)
}

func (watcher *Watcher) speedup() {
	if watcher.chainConfig.AppConfig.Speedup {
		latestFinalizedHeight := watcher.latestFinalizedHeight
		latestMainnetHeight := watcher.rpcClient.GetLatestHeight(true)
		start := uint64(watcher.lastKnownEpochNum) + 1
		for {
			if latestMainnetHeight < latestFinalizedHeight+watcher.numBlocksInEpoch {
				watcher.ccEpochSpeedup()
				break
			}
			epochs := watcher.smartBchRpcClient.GetEpochs(start, start+100)
			if len(epochs) == 0 {
				watcher.ccEpochSpeedup()
				break
			}
			for _, e := range epochs {
				out, _ := json.Marshal(e)
				fmt.Println(string(out))
			}
			watcher.epochList = append(watcher.epochList, epochs...)
			for _, e := range epochs {
				if e.EndTime != 0 {
					watcher.EpochChan <- e
				}
			}
			latestFinalizedHeight += int64(len(epochs)) * watcher.numBlocksInEpoch
			start = start + uint64(len(epochs))
		}
		watcher.latestFinalizedHeight = latestFinalizedHeight
		watcher.lastEpochEndHeight = latestFinalizedHeight
		watcher.logger.Debug("After speedup", "latestFinalizedHeight", watcher.latestFinalizedHeight)
	}
}

func (watcher *Watcher) ccEpochSpeedup() {
	if !param.ShaGateSwitch {
		return
	}
	start := uint64(watcher.lastKnownCCEpochNum) + 1
	for {
		epochs := watcher.smartBchRpcClient.GetCCEpochs(start, start+100)
		if len(epochs) == 0 {
			break
		}
		for _, e := range epochs {
			out, _ := json.Marshal(e)
			fmt.Println(string(out))
		}
		watcher.ccEpochList = append(watcher.ccEpochList, epochs...)
		for _, e := range epochs {
			watcher.CCEpochChan <- e
		}
		start = start + uint64(len(epochs))
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
	//if watcher.latestFinalizedHeight-watcher.lastCCEpochEndHeight == watcher.numBlocksInCCEpoch {
	//	watcher.generateNewCCEpoch()
	//}
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

func (watcher *Watcher) CheckSanity(disableBchClient, skipCheck bool) {
	if disableBchClient || skipCheck {
		return
	}
	latestHeight := watcher.rpcClient.GetLatestHeight(false)
	if latestHeight <= 0 {
		panic("Watcher GetLatestHeight failed in Sanity Check")
	}
	blk := watcher.rpcClient.GetBlockByHeight(latestHeight, false)
	if blk == nil {
		panic("Watcher GetBlockByHeight failed in Sanity Check")
	}
}

// sort by pubkey (small to big) first; then sort by nominationCount;
// so nominations sort by NominationCount, if count is equal, smaller pubkey stand front
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
	ccEpochLen := len(watcher.ccEpochList)
	if ccEpochLen > 5*int(param.StakingNumBlocksInEpoch/param.BlocksInCCEpoch) {
		watcher.epochList = watcher.epochList[ccEpochLen-5:]
	}
}
