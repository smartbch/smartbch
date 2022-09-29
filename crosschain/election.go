package crosschain

import (
	"bytes"
	"encoding/json"
	"sort"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/libs/log"

	mevmtypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/crosschain/covenant"
	cctypes "github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/param"
)

const (
	OperatorsSlot               = 0
	OperatorElectedTimeField    = 7
	OperatorOldElectedTimeField = 8
	OperatorWords               = 9

	MonitorsLastElectionTimeSlot = 0
	MonitorsSlot                 = 1
	MonitorElectedTimeField      = 5
	MonitorOldElectedTimeField   = 6
	MonitorWords                 = 7
)

const (
	OperatorElectionOK                  = 0
	OperatorElectionNotEnoughCandidates = 1
	OperatorElectionNotChanged          = 2
	OperatorElectionChangedTooMany      = 3

	MonitorElectionOK                     = 0
	MonitorElectionInvalidNominationCount = 1
	MonitorElectionInvalidNominations     = 2
	MonitorElectionNotChanged             = 2
	MonitorElectionChangedTooMany         = 3
)

var (
	operatorMinStakedAmt = uint256.NewInt(0).Mul(uint256.NewInt(param.OperatorMinStakedBCH), uint256.NewInt(1e18))
	monitorMinStakedAmt  = uint256.NewInt(0).Mul(uint256.NewInt(param.MonitorMinStakedBCH), uint256.NewInt(1e18))
)

/*
   struct OperatorInfo {
       address addr;           // address
       uint    pubkeyPrefix;   // 0x02 or 0x03
       bytes32 pubkeyX;        // x
       bytes32 rpcUrl;         // ip:port
       bytes32 intro;          // introduction
       uint    totalStakedAmt; // in BCH
       uint    selfStakedAmt;  // in BCH
       uint    electedTime;    // 0 means not elected, set by Golang
       uint    oldElectedTime; // used to get old operators, set by Golang
   }
*/

type OperatorInfo struct {
	Addr           gethcmn.Address
	Pubkey         []byte // 33 bytes
	RpcUrl         []byte // 32 bytes
	Intro          []byte // 32 bytes
	TotalStakedAmt *uint256.Int
	SelfStakedAmt  *uint256.Int
	ElectedTime    *uint256.Int
	OldElectedTime *uint256.Int

	// only used by election logic
	electedFlag bool
}

func GetOperatorInfos(ctx *mevmtypes.Context) (result []OperatorInfo) {
	return ReadOperatorInfos(ctx, param.OperatorsGovSequence)
}
func ReadOperatorInfos(ctx *mevmtypes.Context, seq uint64) (result []OperatorInfo) {
	arrSlot := uint256.NewInt(OperatorsSlot).PaddedBytes(32)
	arrLen := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(arrSlot)))
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))

	for i := uint64(0); i < arrLen.ToBig().Uint64(); i++ {
		result = append(result, readOperatorInfo(ctx, seq, arrLoc))
	}
	return
}
func readOperatorInfo(ctx *mevmtypes.Context, seq uint64, loc *uint256.Int) OperatorInfo {
	addr := ctx.GetStorageAt(seq, string(loc.PaddedBytes(32)))                             // slot#0
	pubkeyPrefix := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))   // slot#1
	pubkeyX := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))        // slot#2
	rpcUrl := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))         // slot#3
	intro := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))          // slot#4
	totalStakedAmt := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))) // slot#5
	selfStakedAmt := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))  // slot#6
	electedTime := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))    // slot#7
	oldElectedTime := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))) // slot#8
	return OperatorInfo{
		Addr:           gethcmn.BytesToAddress(addr),
		Pubkey:         append(pubkeyPrefix[31:], pubkeyX...),
		RpcUrl:         rpcUrl[:],
		Intro:          intro[:],
		TotalStakedAmt: uint256.NewInt(0).SetBytes(totalStakedAmt),
		SelfStakedAmt:  uint256.NewInt(0).SetBytes(selfStakedAmt),
		ElectedTime:    uint256.NewInt(0).SetBytes(electedTime),
		OldElectedTime: uint256.NewInt(0).SetBytes(oldElectedTime),
	}
}

func WriteOperatorElectedTime(ctx *mevmtypes.Context, seq uint64, operatorIdx uint64, val uint64) {
	arrSlot := uint256.NewInt(OperatorsSlot).PaddedBytes(32)
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))
	itemLoc := uint256.NewInt(0).AddUint64(arrLoc, operatorIdx*OperatorWords)
	fieldLoc := uint256.NewInt(0).AddUint64(itemLoc, OperatorElectedTimeField)
	ctx.SetStorageAt(seq, string(fieldLoc.PaddedBytes(32)),
		uint256.NewInt(val).PaddedBytes(32))
}
func WriteOperatorOldElectedTime(ctx *mevmtypes.Context, seq uint64, operatorIdx uint64, val uint64) {
	arrSlot := uint256.NewInt(OperatorsSlot).PaddedBytes(32)
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))
	itemLoc := uint256.NewInt(0).AddUint64(arrLoc, operatorIdx*OperatorWords)
	fieldLoc := uint256.NewInt(0).AddUint64(itemLoc, OperatorOldElectedTimeField)
	ctx.SetStorageAt(seq, string(fieldLoc.PaddedBytes(32)),
		uint256.NewInt(val).PaddedBytes(32))
}

func ElectOperators(ctx *mevmtypes.Context, blockTime int64, logger log.Logger) int {
	return ElectOperators_(ctx, param.OperatorsGovSequence, blockTime, logger)
}
func ElectOperators_(ctx *mevmtypes.Context, seq uint64, blockTime int64, logger log.Logger) int {
	logger.Info("elect operators")
	operatorInfos := ReadOperatorInfos(ctx, seq)
	eligibleOperatorInfos := getEligibleOperatorCandidates(operatorInfos)
	if len(eligibleOperatorInfos) < param.OperatorsCount {
		logger.Info("not enough eligible operator candidates!")
		return OperatorElectionNotEnoughCandidates
	}

	sortOperatorInfos(eligibleOperatorInfos)
	if len(eligibleOperatorInfos) > param.OperatorsCount {
		eligibleOperatorInfos = eligibleOperatorInfos[:param.OperatorsCount]
	}

	electedOperatorPubkeyMap := map[string]bool{}
	for _, operatorInfo := range eligibleOperatorInfos {
		electedOperatorPubkeyMap[string(operatorInfo.Pubkey)] = true
	}

	lastTimeElectedCount := 0
	thisTimeElectedCount := 0
	newElectedCount := 0
	for i, operatorInfo := range operatorInfos {
		if electedOperatorPubkeyMap[string(operatorInfo.Pubkey)] {
			operatorInfos[i].electedFlag = true
			operatorInfo.electedFlag = true
		}

		lastElectedTime := operatorInfo.ElectedTime.Uint64()
		if lastElectedTime > 0 {
			lastTimeElectedCount++
		}
		if operatorInfo.electedFlag {
			thisTimeElectedCount++
			if lastElectedTime == 0 {
				newElectedCount++
			}
		}
	}

	if newElectedCount == 0 {
		logger.Info("operators not changed")
		return OperatorElectionNotChanged
	}
	if newElectedCount > param.OperatorsMaxChangeCount {
		if newElectedCount == param.OperatorsCount && lastTimeElectedCount == 0 {
			logger.Info("first operators election")
		} else {
			logger.Info("too many new operators!",
				"newElectedCount", newElectedCount)
			return OperatorElectionChangedTooMany
		}
	}

	// everything is ok
	for idx, operatorInfo := range operatorInfos {
		if !operatorInfo.electedFlag {
			WriteOperatorElectedTime(ctx, seq, uint64(idx), 0)
		} else {
			if operatorInfo.ElectedTime.Uint64() == 0 {
				WriteOperatorElectedTime(ctx, seq, uint64(idx), uint64(blockTime))
			}
		}
	}
	return OperatorElectionOK
}
func getEligibleOperatorCandidates(allOperatorInfos []OperatorInfo) []OperatorInfo {
	eligibleOperatorInfos := make([]OperatorInfo, 0, len(allOperatorInfos))
	for _, operatorInfo := range allOperatorInfos {
		if isEligibleOperator(operatorInfo) {
			eligibleOperatorInfos = append(eligibleOperatorInfos, operatorInfo)
		}
	}
	return eligibleOperatorInfos
}
func isEligibleOperator(operatorInfo OperatorInfo) bool {
	// TODO: check more fields
	return !operatorInfo.SelfStakedAmt.Lt(operatorMinStakedAmt)
}
func sortOperatorInfos(operatorInfos []OperatorInfo) {
	sort.Slice(operatorInfos, func(i, j int) bool {
		return !operatorInfoLessFn(operatorInfos[i], operatorInfos[j])
	})
}
func operatorInfoLessFn(a, b OperatorInfo) bool {
	if a.TotalStakedAmt.Lt(b.TotalStakedAmt) {
		return true
	}
	if a.SelfStakedAmt.Lt(b.SelfStakedAmt) {
		return true
	}
	// TODO: compare more fields
	return false
}

func updateOperatorElectedTimes(ctx *mevmtypes.Context, seq uint64,
	blockTime int64, operatorInfos []OperatorInfo) {

	for idx, operatorInfo := range operatorInfos {
		WriteOperatorOldElectedTime(ctx, seq, uint64(idx), operatorInfo.ElectedTime.Uint64())

		if !operatorInfo.electedFlag {
			WriteOperatorElectedTime(ctx, seq, uint64(idx), 0)
		} else {
			WriteOperatorElectedTime(ctx, seq, uint64(idx), uint64(blockTime))
		}
	}
}

func GetOperatorPubkeySet(ctx *mevmtypes.Context) (pubkeys [][]byte) {
	operatorInfos := ReadOperatorInfos(ctx, param.OperatorsGovSequence)
	for _, operatorInfo := range operatorInfos {
		if operatorInfo.ElectedTime.Uint64() > 0 {
			pubkeys = append(pubkeys, operatorInfo.Pubkey[:])
		}
	}
	return
}

/*
   struct MonitorInfo {
       address addr;           // address
       uint    pubkeyPrefix;   // 0x02 or 0x03
       bytes32 pubkeyX;        // x
       bytes32 intro;          // introduction
       uint    stakedAmt;      // staked BCH
       uint    electedTime;    // 0 means not elected, set by Golang
       uint    oldElectedTime; // used to get old monitors, set by Golang
   }
*/

type MonitorInfo struct {
	Addr           gethcmn.Address
	Pubkey         []byte // 33 bytes
	Intro          []byte // 32 bytes
	StakedAmt      *uint256.Int
	ElectedTime    *uint256.Int
	OldElectedTime *uint256.Int

	// only used by election logic
	nominatedCount int64
}

func GetMonitorInfos(ctx *mevmtypes.Context) []MonitorInfo {
	return ReadMonitorInfos(ctx, param.MonitorsGovSequence)
}
func ReadMonitorInfos(ctx *mevmtypes.Context, seq uint64) (result []MonitorInfo) {
	arrSlot := uint256.NewInt(MonitorsSlot).PaddedBytes(32)
	arrLen := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(arrSlot)))
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))

	for i := uint64(0); i < arrLen.ToBig().Uint64(); i++ {
		result = append(result, readMonitorInfo(ctx, seq, arrLoc))
	}
	return
}

func readMonitorInfo(ctx *mevmtypes.Context, seq uint64, loc *uint256.Int) MonitorInfo {
	addr := ctx.GetStorageAt(seq, string(loc.PaddedBytes(32)))                             // slot#0
	pubkeyPrefix := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))   // slot#1
	pubkeyX := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))        // slot#2
	intro := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))          // slot#3
	stakedAmt := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))      // slot#4
	electedTime := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))    // slot#5
	oldElectedTime := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))) // slot#6
	return MonitorInfo{
		Addr:           gethcmn.BytesToAddress(addr),
		Pubkey:         append(pubkeyPrefix[31:], pubkeyX...),
		Intro:          intro[:],
		StakedAmt:      uint256.NewInt(0).SetBytes(stakedAmt),
		ElectedTime:    uint256.NewInt(0).SetBytes(electedTime),
		OldElectedTime: uint256.NewInt(0).SetBytes(oldElectedTime),
	}
}

func WriteMonitorElectedTime(ctx *mevmtypes.Context, seq uint64, monitorIdx uint64, val uint64) {
	arrSlot := uint256.NewInt(MonitorsSlot).PaddedBytes(32)
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))
	itemLoc := uint256.NewInt(0).AddUint64(arrLoc, monitorIdx*MonitorWords)
	fieldLoc := uint256.NewInt(0).AddUint64(itemLoc, MonitorElectedTimeField)
	ctx.SetStorageAt(seq, string(fieldLoc.PaddedBytes(32)),
		uint256.NewInt(val).PaddedBytes(32))
}
func WriteMonitorOldElectedTime(ctx *mevmtypes.Context, seq uint64, monitorIdx uint64, val uint64) {
	arrSlot := uint256.NewInt(MonitorsSlot).PaddedBytes(32)
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))
	itemLoc := uint256.NewInt(0).AddUint64(arrLoc, monitorIdx*MonitorWords)
	fieldLoc := uint256.NewInt(0).AddUint64(itemLoc, MonitorOldElectedTimeField)
	ctx.SetStorageAt(seq, string(fieldLoc.PaddedBytes(32)),
		uint256.NewInt(val).PaddedBytes(32))
}

func ReadMonitorsLastElectionTime(ctx *mevmtypes.Context, seq uint64) *uint256.Int {
	slot := uint256.NewInt(MonitorsLastElectionTimeSlot).PaddedBytes(32)
	val := ctx.GetStorageAt(seq, string(slot))
	return uint256.NewInt(0).SetBytes(val)
}
func WriteMonitorsLastElectionTime(ctx *mevmtypes.Context, seq uint64, val uint64) {
	slot := uint256.NewInt(MonitorsLastElectionTimeSlot).PaddedBytes(32)
	ctx.SetStorageAt(seq, string(slot), uint256.NewInt(val).PaddedBytes(32))
}

func ElectMonitors(ctx *mevmtypes.Context, nominations []*cctypes.Nomination, blockTime int64, logger log.Logger) int {
	return ElectMonitors_(ctx, param.MonitorsGovSequence, nominations, blockTime, logger)
}
func ElectMonitors_(ctx *mevmtypes.Context, seq uint64,
	nominations []*cctypes.Nomination, blockTime int64, logger log.Logger,
) int {
	nominationsJson, _ := json.Marshal(nominations)
	logger.Info("elect monitors", "nominationsJson", nominationsJson)

	if len(nominations) != param.MonitorsCount {
		logger.Info("invalid nomination count!")
		return MonitorElectionInvalidNominationCount
	}

	monitorInfos := ReadMonitorInfos(ctx, seq)
	monitorInfosJson, _ := json.Marshal(monitorInfos)
	logger.Info("monitorInfos", "json", monitorInfosJson)

	lastTimeElectedCount := 0
	thisTimeElectedCount := 0
	newElectedCount := 0
	for i, monitorInfo := range monitorInfos {
		if !monitorInfo.StakedAmt.Lt(monitorMinStakedAmt) {
			for _, nomination := range nominations {
				if bytes.Equal(nomination.Pubkey[:], monitorInfo.Pubkey) {
					monitorInfos[i].nominatedCount = nomination.NominatedCount
					monitorInfo.nominatedCount = nomination.NominatedCount
					break
				}
			}
		}

		lastElectedTime := monitorInfo.ElectedTime.Uint64()
		if lastElectedTime > 0 {
			lastTimeElectedCount++
		}
		if monitorInfo.nominatedCount > 0 {
			thisTimeElectedCount++
			if lastElectedTime == 0 {
				newElectedCount++
			}
		}
	}

	if thisTimeElectedCount != param.MonitorsCount {
		logger.Info("invalid nominations",
			"thisTimeElectedCount", thisTimeElectedCount)
		return MonitorElectionInvalidNominations
	}
	if newElectedCount == 0 {
		logger.Info("monitors not changed")
		return MonitorElectionNotChanged
	}
	if newElectedCount > param.MonitorsMaxChangeCount {
		if newElectedCount == param.MonitorsCount && lastTimeElectedCount == 0 {
			logger.Info("first monitors election")
		} else {
			logger.Info("too many new monitors!",
				"newElectedCount", newElectedCount)
			return MonitorElectionChangedTooMany
		}
	}

	// everything is ok
	for idx, monitorInfo := range monitorInfos {
		WriteMonitorOldElectedTime(ctx, seq, uint64(idx), monitorInfo.ElectedTime.Uint64())
		if monitorInfo.nominatedCount == 0 {
			WriteMonitorElectedTime(ctx, seq, uint64(idx), 0)
		} else {
			WriteMonitorElectedTime(ctx, seq, uint64(idx), uint64(blockTime))
		}
	}
	WriteMonitorsLastElectionTime(ctx, seq, uint64(blockTime))
	return MonitorElectionOK
}

func GetMonitorPubkeySet(ctx *mevmtypes.Context) (pubkeys [][]byte) {
	monitorInfos := ReadMonitorInfos(ctx, param.MonitorsGovSequence)
	for _, monitorInfo := range monitorInfos {
		if monitorInfo.ElectedTime.Uint64() > 0 {
			pubkeys = append(pubkeys, monitorInfo.Pubkey[:])
		}
	}
	return
}

func GetCCCovenantP2SHAddr(ctx *mevmtypes.Context) ([20]byte, error) {
	operatorPubkeys := GetOperatorPubkeySet(ctx)
	monitorsPubkeys := GetMonitorPubkeySet(ctx)
	ccc, err := covenant.NewDefaultCcCovenant(operatorPubkeys, monitorsPubkeys)
	if err != nil {
		return [20]byte{}, err
	}
	addr, err := ccc.GetP2SHAddress20()
	if err != nil {
		return [20]byte{}, err
	}
	return addr, nil
}
