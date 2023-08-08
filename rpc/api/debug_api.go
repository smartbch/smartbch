package api

import (
	"encoding/json"
	"runtime"
	"sync/atomic"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/param"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

const (
	StatusUpdateInterval = 60 // seconds
)

type Stats struct {
	NumGoroutine     int    `json:"numGoroutine"`
	NumGC            uint32 `json:"numGC"`
	MemAllocMB       uint64 `json:"memAllocMB"`
	MemSysMB         uint64 `json:"memSysMB"`
	OsMemTotalMB     uint64 `json:"osMemTotalMB"`
	OsMemUsedMB      uint64 `json:"osMemUsedMB"`
	OsMemCachedMB    uint64 `json:"osMemCachedMB"`
	OsMemFreeMB      uint64 `json:"osMemFreeMB"`
	OsMemActiveMB    uint64 `json:"osMemActiveMB"`
	OsMemInactiveMB  uint64 `json:"osMemInactiveMB"`
	OsMemSwapTotalMB uint64 `json:"osMemSwapTotalMB"`
	OsMemSwapUsedMB  uint64 `json:"osMemSwapUsedMB"`
	OsMemSwapFreeMB  uint64 `json:"osMemSwapFreeMB"`
	NumEthCall       uint64 `json:"numEthCall"`
}

type DebugAPI interface {
	GetStats() Stats
	GetSeq(addr gethcmn.Address) hexutil.Uint64
	NodeInfo() json.RawMessage
	ValidatorOnlineInfos() json.RawMessage
	ValidatorWatchInfos() json.RawMessage
}

type debugAPI struct {
	logger         log.Logger
	ethAPI         *ethAPI
	lastUpdateTime int64
	stats          Stats
}

func newDebugAPI(ethAPI *ethAPI, logger log.Logger) DebugAPI {
	return &debugAPI{
		logger: logger,
		ethAPI: ethAPI,
	}
}

func (api *debugAPI) GetSeq(addr gethcmn.Address) hexutil.Uint64 {
	api.logger.Debug("debug_getSeq")
	return hexutil.Uint64(api.ethAPI.backend.GetSeq(addr))
}

func (api *debugAPI) NodeInfo() json.RawMessage {
	api.logger.Debug("debug_nodeInfo")
	nodeInfo := api.ethAPI.backend.NodeInfo()
	bytes, _ := json.Marshal(nodeInfo)
	return bytes
}

func (api *debugAPI) GetStats() Stats {
	api.logger.Debug("debug_getStats")

	now := time.Now().Unix()
	lastUpdateTime := atomic.LoadInt64(&api.lastUpdateTime)
	if now > lastUpdateTime+StatusUpdateInterval {
		if atomic.CompareAndSwapInt64(&api.lastUpdateTime, lastUpdateTime, now) {
			api.updateStats()
		}
	}

	return api.stats
}

func (api *debugAPI) updateStats() {
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)

	api.stats.NumGoroutine = runtime.NumGoroutine()
	api.stats.NumGC = memStats.NumGC
	api.stats.MemAllocMB = toMB(memStats.Alloc)
	api.stats.MemSysMB = toMB(memStats.Sys)

	osMemStats, err := memory.Get()
	if err == nil {
		api.stats.OsMemTotalMB = toMB(osMemStats.Total)
		api.stats.OsMemUsedMB = toMB(osMemStats.Used)
		api.stats.OsMemCachedMB = toMB(osMemStats.Cached)
		api.stats.OsMemFreeMB = toMB(osMemStats.Free)
		api.stats.OsMemActiveMB = toMB(osMemStats.Active)
		api.stats.OsMemInactiveMB = toMB(osMemStats.Inactive)
		api.stats.OsMemSwapTotalMB = toMB(osMemStats.SwapTotal)
		api.stats.OsMemSwapUsedMB = toMB(osMemStats.SwapUsed)
		api.stats.OsMemSwapFreeMB = toMB(osMemStats.SwapFree)
	}

	api.stats.NumEthCall = atomic.LoadUint64(&api.ethAPI.numCall)
}

func toMB(n uint64) uint64 {
	return n / 1024 / 1024
}

/* Validator Online Info */

type ValidatorOnlineInfosToMarshal struct {
	StartHeight            int64                  `json:"start_height"`
	EndHeight              int64                  `json:"end_height"`
	OnlineInfos            []*OnlineInfoToMarshal `json:"online_infos"`
	ValidatorsMaybeSlashed []gethcmn.Address      `json:"validators_maybe_slashed"`
}

type OnlineInfoToMarshal struct {
	ValidatorConsensusAddress gethcmn.Address `json:"validator_consensus_address"`
	SignatureCount            int32           `json:"signature_count"`
	HeightOfLastSignature     int64           `json:"height_of_last_signature"`
}

func castValidatorOnlineInfos(latestHeight int64, infos stakingtypes.ValidatorOnlineInfos) ValidatorOnlineInfosToMarshal {
	infosToMarshal := ValidatorOnlineInfosToMarshal{
		StartHeight: infos.StartHeight,
		EndHeight:   infos.StartHeight + param.OnlineWindowSize,
		OnlineInfos: make([]*OnlineInfoToMarshal, len(infos.OnlineInfos)),
	}
	for i, onlineInfo := range infos.OnlineInfos {
		infosToMarshal.OnlineInfos[i] = &OnlineInfoToMarshal{
			ValidatorConsensusAddress: onlineInfo.ValidatorConsensusAddress,
			SignatureCount:            onlineInfo.SignatureCount,
			HeightOfLastSignature:     onlineInfo.HeightOfLastSignature,
		}
		blockNumsSinceStartHeight := latestHeight - infos.StartHeight
		if blockNumsSinceStartHeight > param.OnlineWindowSize/5 {
			if int64(onlineInfo.SignatureCount) < blockNumsSinceStartHeight*int64(param.MinOnlineSignatures)/param.OnlineWindowSize {
				infosToMarshal.ValidatorsMaybeSlashed = append(infosToMarshal.ValidatorsMaybeSlashed, onlineInfo.ValidatorConsensusAddress)
			}
		}
	}
	return infosToMarshal
}

func (api *debugAPI) ValidatorOnlineInfos() json.RawMessage {
	api.logger.Debug("debug_validatorOnlineInfos")
	latestHeight, onlineInfos := api.ethAPI.backend.ValidatorOnlineInfos()
	onlineInfosToMarshal := castValidatorOnlineInfos(latestHeight, onlineInfos)
	bytes, _ := json.Marshal(onlineInfosToMarshal)
	return bytes
}

type ValidatorWatchInfosToMarshal struct {
	StartHeight int64                 `json:"start_height"`
	EndHeight   int64                 `json:"end_height"`
	WatchInfos  []*WatchInfoToMarshal `json:"online_infos"`
}

type WatchInfoToMarshal struct {
	ValidatorConsensusAddress gethcmn.Address `json:"validator_consensus_address"`
	SignatureCount            int32           `json:"signature_count"`
	HeightOfLastSignature     int64           `json:"height_of_last_signature"`
	VotingPowerBeDecreased    bool            `json:"voting_power_be_decreased"`
}

func castValidatorWatchInfos(infos stakingtypes.ValidatorWatchInfos) ValidatorWatchInfosToMarshal {
	infosToMarshal := ValidatorWatchInfosToMarshal{
		StartHeight: infos.StartHeight,
		EndHeight:   infos.StartHeight + param.ValidatorWatchWindowSize,
		WatchInfos:  make([]*WatchInfoToMarshal, len(infos.WatchInfos)),
	}
	for i, watchInfo := range infos.WatchInfos {
		infosToMarshal.WatchInfos[i] = &WatchInfoToMarshal{
			ValidatorConsensusAddress: watchInfo.ValidatorConsensusAddress,
			SignatureCount:            watchInfo.SignatureCount,
			HeightOfLastSignature:     watchInfo.HeightOfLastSignature,
			VotingPowerBeDecreased:    watchInfo.Handled,
		}
	}
	return infosToMarshal
}

func (api *debugAPI) ValidatorWatchInfos() json.RawMessage {
	api.logger.Debug("debug_validatorWatchInfos")
	watchInfos := api.ethAPI.backend.ValidatorWatchInfos()
	onlineInfosToMarshal := castValidatorWatchInfos(watchInfos)
	bytes, _ := json.Marshal(onlineInfosToMarshal)
	return bytes
}
