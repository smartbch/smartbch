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
)

// TODO: move to params.go
const (
	OperatorsGovSeq         = 0 // TODO
	OperatorsSlot           = 0
	OperatorWords           = 8
	OperatorsCount          = 10
	OperatorsMaxChangeCount = 3
	OperatorMinStakedAmt    = 10000e8

	MonitorsGovSeq               = 0 // TODO
	MonitorsLastElectionTimeSlot = 0
	MonitorsSlot                 = 1
	MonitorWords                 = 6
	MonitorsCount                = 3
	MonitorsMaxChangeCount       = 1
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

	// only used by election logic
	electedFlag bool
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
	addr := gethcmn.BytesToAddress(ctx.GetStorageAt(seq, string(loc.PaddedBytes(32))))
	pubkeyPrefix := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	pubkeyX := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	rpcUrl := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	intro := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	selfStakedAmt := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))))
	totalStakedAmt := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))))
	key := loc.AddUint64(loc, 1).PaddedBytes(32)
	electedTime := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(key)))
	loc.AddUint64(loc, 1)
	return OperatorInfo{
		Addr:           addr,
		Pubkey:         append(pubkeyPrefix[31:], pubkeyX...),
		RpcUrl:         rpcUrl[:],
		Intro:          intro[:],
		TotalStakedAmt: totalStakedAmt,
		SelfStakedAmt:  selfStakedAmt,
		ElectedTime:    electedTime,
	}
}

func WriteOperatorElectedTime(ctx *mevmtypes.Context, seq uint64, operatorIdx uint64, val uint64) {
	arrSlot := uint256.NewInt(OperatorsSlot).PaddedBytes(32)
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))
	itemLoc := uint256.NewInt(0).AddUint64(arrLoc, operatorIdx*OperatorWords)
	fieldLoc := uint256.NewInt(0).AddUint64(itemLoc, OperatorWords-1)
	ctx.SetStorageAt(seq, string(fieldLoc.PaddedBytes(32)),
		uint256.NewInt(val).PaddedBytes(32))
}

func ElectOperators(ctx *mevmtypes.Context, blockTime int64, logger log.Logger) {
	logger.Info("elect operators")
	operatorInfos := ReadOperatorInfos(ctx, OperatorsGovSeq)
	eligibleOperatorInfos := getEligibleOperatorCandidates(operatorInfos)
	if len(eligibleOperatorInfos) < OperatorsCount {
		logger.Info("not enough eligible operator candidates!")
		return
	}

	sortOperatorInfos(eligibleOperatorInfos)
	if len(eligibleOperatorInfos) > OperatorsCount {
		eligibleOperatorInfos = eligibleOperatorInfos[:OperatorsCount]
	}

	electedOperatorPubkeyMap := map[string]bool{}
	for _, operatorInfo := range eligibleOperatorInfos {
		electedOperatorPubkeyMap[string(operatorInfo.Pubkey)] = true
	}

	lastTimeElectedCount := 0
	thisTimeElectedCount := 0
	newElectedCount := 0
	for _, operatorInfo := range operatorInfos {
		if electedOperatorPubkeyMap[string(operatorInfo.Pubkey)] {
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
		return
	}
	if newElectedCount > OperatorsMaxChangeCount {
		if newElectedCount == OperatorsCount && lastTimeElectedCount == 0 {
			logger.Info("first operators election")
		} else {
			logger.Info("too many new operators!",
				"newElectedCount", newElectedCount)
			return
		}
	}

	// everything is ok
	for idx, operatorInfo := range operatorInfos {
		if !operatorInfo.electedFlag {
			WriteOperatorElectedTime(ctx, OperatorsGovSeq, uint64(idx), 0)
		} else {
			if operatorInfo.ElectedTime.Uint64() == 0 {
				WriteOperatorElectedTime(ctx, OperatorsGovSeq, uint64(idx), uint64(blockTime))
			}
		}
	}
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
func sortOperatorInfos(operatorInfos []OperatorInfo) {
	sort.Slice(operatorInfos, func(i, j int) bool {
		return operatorInfoLessFn(operatorInfos[i], operatorInfos[j])
	})
}
func isEligibleOperator(operatorInfo OperatorInfo) bool {
	// TODO: check more fields
	return operatorInfo.SelfStakedAmt.Uint64() >= OperatorMinStakedAmt
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

func GetOperatorPubkeySet(ctx *mevmtypes.Context, seq uint64) (pubkeys [][]byte) {
	operatorInfos := ReadOperatorInfos(ctx, seq)
	for _, operatorInfo := range operatorInfos {
		if operatorInfo.ElectedTime.Uint64() > 0 {
			pubkeys = append(pubkeys, operatorInfo.Pubkey[:])
		}
	}
	return
}

/*
   struct MonitorInfo {
       address addr;         // address
       uint    pubkeyPrefix; // 0x02 or 0x03
       bytes32 pubkeyX;      // x
       bytes32 intro;        // introduction
       uint    stakedAmt;    // in BCH
       uint    electedTime;  // 0 means not elected, set by Golang
   }
*/

type MonitorInfo struct {
	Addr        gethcmn.Address
	Pubkey      []byte // 33 bytes
	Intro       []byte // 32 bytes
	StakedAmt   *uint256.Int
	ElectedTime *uint256.Int

	// only used by election logic
	nominatedCount int64
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
	addr := gethcmn.BytesToAddress(ctx.GetStorageAt(seq, string(loc.PaddedBytes(32))))
	pubkeyPrefix := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	pubkeyX := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	intro := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	stakedAmt := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))))
	key := loc.AddUint64(loc, 1).PaddedBytes(32)
	electedTime := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(key)))
	loc.AddUint64(loc, 1)
	return MonitorInfo{
		Addr:        addr,
		Pubkey:      append(pubkeyPrefix[31:], pubkeyX...),
		Intro:       intro[:],
		StakedAmt:   stakedAmt,
		ElectedTime: electedTime,
	}
}

func WriteMonitorElectedTime(ctx *mevmtypes.Context, seq uint64, monitorIdx uint64, val uint64) {
	arrSlot := uint256.NewInt(MonitorsSlot).PaddedBytes(32)
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))
	itemLoc := uint256.NewInt(0).AddUint64(arrLoc, monitorIdx*MonitorWords)
	fieldLoc := uint256.NewInt(0).AddUint64(itemLoc, MonitorWords-1)
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

func ElectMonitors(ctx *mevmtypes.Context, nominations []*cctypes.Nomination, blockTime int64, logger log.Logger) {
	nominationsJson, _ := json.Marshal(nominations)
	logger.Info("elect monitors", "nominationsJson", nominationsJson)

	if len(nominations) != MonitorsCount {
		logger.Info("invalid nominations count!")
		return
	}

	monitorInfos := ReadMonitorInfos(ctx, MonitorsGovSeq)
	monitorInfosJson, _ := json.Marshal(monitorInfos)
	logger.Info("monitorInfos", "json", monitorInfosJson)

	lastTimeElectedCount := 0
	thisTimeElectedCount := 0
	newElectedCount := 0
	for _, monitorInfo := range monitorInfos {
		// TODO: check MIN_STAKE ?
		for _, nomination := range nominations {
			if bytes.Equal(nomination.Pubkey[:], monitorInfo.Pubkey) {
				monitorInfo.nominatedCount = nomination.NominatedCount
				break
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

	if thisTimeElectedCount != MonitorsCount {
		logger.Info("invalid nominations",
			"thisTimeElectedCount", thisTimeElectedCount)
		return
	}
	if newElectedCount == 0 {
		logger.Info("monitors not changed")
		return
	}
	if newElectedCount > MonitorsMaxChangeCount {
		if newElectedCount == MonitorsCount && lastTimeElectedCount == 0 {
			logger.Info("first monitors election")
		} else {
			logger.Info("too many new monitors!",
				"newElectedCount", newElectedCount)
			return
		}
	}

	// everything is ok
	for idx, monitorInfo := range monitorInfos {
		if monitorInfo.nominatedCount == 0 {
			WriteMonitorElectedTime(ctx, MonitorsGovSeq, uint64(idx), 0)
		} else {
			if monitorInfo.ElectedTime.Uint64() == 0 {
				WriteMonitorElectedTime(ctx, MonitorsGovSeq, uint64(idx), uint64(blockTime))
			}
		}
	}
}

func GetMonitorPubkeySet(ctx *mevmtypes.Context, seq uint64) (pubkeys [][]byte) {
	monitorInfos := ReadMonitorInfos(ctx, seq)
	for _, monitorInfo := range monitorInfos {
		if monitorInfo.ElectedTime.Uint64() > 0 {
			pubkeys = append(pubkeys, monitorInfo.Pubkey[:])
		}
	}
	return
}

func GetCCCovenantP2SHAddr(ctx *mevmtypes.Context) [20]byte {
	operatorPubkeys := GetOperatorPubkeySet(ctx, OperatorsGovSeq)
	monitorsPubkeys := GetMonitorPubkeySet(ctx, MonitorsGovSeq)
	ccc, err := covenant.NewCcCovenantMainnet(operatorPubkeys, monitorsPubkeys)
	if err != nil {
		panic(err)
	}
	addr, err := ccc.GetP2SHAddress20()
	if err != nil {
		panic(err)
	}
	return addr
}
