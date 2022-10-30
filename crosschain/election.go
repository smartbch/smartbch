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
	MonitorWords                 = 8
)

const (
	OperatorElectionOK = iota
	OperatorElectionNoNewCandidates
	OperatorElectionNotEnoughCandidates
	OperatorElectionNotChanged
)

const (
	MonitorElectionOK = iota
	MonitorElectionNoNewCandidates
	MonitorElectionNotEnoughCandidates
	MonitorElectionNotChanged
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
       uint    totalStakedAmt; // total staked BCH
       uint    selfStakedAmt;  // self staked BCH
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

func GetOperatorInfos(ctx *mevmtypes.Context) (result []*OperatorInfo) {
	return ReadOperatorInfos(ctx, param.OperatorsGovSequence)
}
func ReadOperatorInfos(ctx *mevmtypes.Context, seq uint64) (result []*OperatorInfo) {
	arrSlot := uint256.NewInt(OperatorsSlot).PaddedBytes(32)
	arrLen := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(arrSlot)))
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))

	for i := uint64(0); i < arrLen.Uint64(); i++ {
		loc := uint256.NewInt(0).AddUint64(arrLoc, i*OperatorWords)
		result = append(result, readOperatorInfo(ctx, seq, loc))
	}
	return
}
func readOperatorInfo(ctx *mevmtypes.Context, seq uint64, loc *uint256.Int) *OperatorInfo {
	addr := ctx.GetStorageAt(seq, string(loc.PaddedBytes(32)))                             // slot#0
	pubkeyPrefix := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))   // slot#1
	pubkeyX := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))        // slot#2
	rpcUrl := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))         // slot#3
	intro := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))          // slot#4
	totalStakedAmt := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))) // slot#5
	selfStakedAmt := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))  // slot#6
	electedTime := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))    // slot#7
	oldElectedTime := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))) // slot#8
	return &OperatorInfo{
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
	return electOperators(ctx, param.OperatorsGovSequence, blockTime, logger)
}
func ElectOperatorsForUT(ctx *mevmtypes.Context, seq uint64, blockTime int64, logger log.Logger) int {
	return electOperators(ctx, seq, blockTime, logger)
}
func electOperators(ctx *mevmtypes.Context, seq uint64, blockTime int64, logger log.Logger) int {
	logger.Info("elect operators ...")
	operatorInfos := ReadOperatorInfos(ctx, seq)
	logger.Info("allOperatorInfos", "json", toJSON(operatorInfos))

	// get and sort current operators
	currOperators := getCurrOperators(operatorInfos)
	sortOperatorInfosDesc(currOperators)
	logger.Info("currOperators", "json", toJSON(currOperators))

	// get and sort eligible new candidates
	newOperatorCandidates := getNewOperatorCandidates(operatorInfos)
	sortOperatorInfosDesc(newOperatorCandidates)
	logger.Info("newOperatorCandidates", "json", toJSON(newOperatorCandidates))

	// first election ?
	if len(currOperators) == 0 {
		logger.Info("first operators election")
		if len(newOperatorCandidates) < param.OperatorsCount {
			logger.Info("not enough candidates for the first election!")
			return OperatorElectionNotEnoughCandidates
		}
		newOperators := newOperatorCandidates[:param.OperatorsCount]
		markOperatorElectedFlags(newOperators, true)
		updateOperatorElectedTimes(ctx, seq, blockTime, operatorInfos)
		return OperatorElectionOK
	}

	if len(newOperatorCandidates) == 0 {
		logger.Info("no new eligible operator candidates!")
		return OperatorElectionNoNewCandidates
	}

	// kick out needless candidates
	if len(newOperatorCandidates) > param.OperatorsMaxChangeCount {
		newOperatorCandidates = newOperatorCandidates[:param.OperatorsMaxChangeCount]
	}

	if operatorInfoLessFn(newOperatorCandidates[0], currOperators[len(currOperators)-1]) {
		logger.Info("operator set not changed")
		return OperatorElectionNotChanged
	}

	allCandidates := append(currOperators, newOperatorCandidates...)
	sortOperatorInfosDesc(allCandidates)
	markOperatorElectedFlags(allCandidates[:param.OperatorsCount], true)
	markOperatorElectedFlags(allCandidates[param.OperatorsCount:], false)
	updateOperatorElectedTimes(ctx, seq, blockTime, operatorInfos)
	logger.Info("new operator set", "json", allCandidates)
	return OperatorElectionOK
}
func getCurrOperators(allOperatorInfos []*OperatorInfo) []*OperatorInfo {
	operators := make([]*OperatorInfo, 0, param.OperatorsCount)
	for _, operatorInfo := range allOperatorInfos {
		if operatorInfo.ElectedTime.GtUint64(0) {
			operators = append(operators, operatorInfo)
		}
	}
	return operators
}
func getNewOperatorCandidates(allOperatorInfos []*OperatorInfo) []*OperatorInfo {
	candidates := make([]*OperatorInfo, 0, len(allOperatorInfos))
	for _, operatorInfo := range allOperatorInfos {
		if operatorInfo.ElectedTime.GtUint64(0) {
			// skip current operators
			continue
		}
		if isEligibleOperatorCandidate(operatorInfo) {
			candidates = append(candidates, operatorInfo)
		}
	}
	return candidates
}
func isEligibleOperatorCandidate(operatorInfo *OperatorInfo) bool {
	return !operatorInfo.SelfStakedAmt.Lt(operatorMinStakedAmt)
	// TODO: check more fields ?
}
func sortOperatorInfosDesc(operatorInfos []*OperatorInfo) {
	sort.Slice(operatorInfos, func(i, j int) bool {
		return !operatorInfoLessFn(operatorInfos[i], operatorInfos[j])
	})
}
func operatorInfoLessFn(a, b *OperatorInfo) bool {
	if x := a.TotalStakedAmt.Cmp(b.TotalStakedAmt); x != 0 {
		return x < 0
	}
	if x := a.SelfStakedAmt.Cmp(b.SelfStakedAmt); x != 0 {
		return x < 0
	}
	return bytes.Compare(a.Addr[:], b.Addr[:]) < 0
}
func markOperatorElectedFlags(operatorInfos []*OperatorInfo, flag bool) {
	for _, operatorInfo := range operatorInfos {
		operatorInfo.electedFlag = flag
	}
}
func updateOperatorElectedTimes(ctx *mevmtypes.Context, seq uint64,
	blockTime int64, operatorInfos []*OperatorInfo) {

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
func GetOldOperatorPubkeySet(ctx *mevmtypes.Context) (pubkeys [][]byte) {
	operatorInfos := ReadOperatorInfos(ctx, param.OperatorsGovSequence)
	for _, operatorInfo := range operatorInfos {
		if operatorInfo.OldElectedTime.Uint64() > 0 {
			pubkeys = append(pubkeys, operatorInfo.Pubkey[:])
		}
	}
	return
}

/*
   struct MonitorInfo {
       address   addr;           // address
       uint      pubkeyPrefix;   // 0x02 or 0x03
       bytes32   pubkeyX;        // x
       bytes32   intro;          // introduction
       uint      stakedAmt;      // staked BCH
       uint      electedTime;    // 0 means not elected, set by Golang
       uint      oldElectedTime; // used to get old monitors, set by Golang
       address[] nominatedBy;    // length of nominatedBy is read by Golang
   }
*/

type MonitorInfo struct {
	Addr           gethcmn.Address
	Pubkey         []byte // 33 bytes
	Intro          []byte // 32 bytes
	StakedAmt      *uint256.Int
	ElectedTime    *uint256.Int
	OldElectedTime *uint256.Int
	NominatedByOps *uint256.Int

	// only used by election logic
	powNominatedCount int64
	electedFlag       bool
}

func GetMonitorInfos(ctx *mevmtypes.Context) []*MonitorInfo {
	return ReadMonitorInfos(ctx, param.MonitorsGovSequence)
}
func ReadMonitorInfos(ctx *mevmtypes.Context, seq uint64) (result []*MonitorInfo) {
	arrSlot := uint256.NewInt(MonitorsSlot).PaddedBytes(32)
	arrLen := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(arrSlot)))
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))

	for i := uint64(0); i < arrLen.Uint64(); i++ {
		loc := uint256.NewInt(0).AddUint64(arrLoc, i*MonitorWords)
		result = append(result, readMonitorInfo(ctx, seq, loc))
	}
	return
}

func readMonitorInfo(ctx *mevmtypes.Context, seq uint64, loc *uint256.Int) *MonitorInfo {
	addr := ctx.GetStorageAt(seq, string(loc.PaddedBytes(32)))                             // slot#0
	pubkeyPrefix := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))   // slot#1
	pubkeyX := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))        // slot#2
	intro := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))          // slot#3
	stakedAmt := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))      // slot#4
	electedTime := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))    // slot#5
	oldElectedTime := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))) // slot#6
	nominatedBy := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))    // slot#7
	return &MonitorInfo{
		Addr:           gethcmn.BytesToAddress(addr),
		Pubkey:         append(pubkeyPrefix[31:], pubkeyX...),
		Intro:          intro[:],
		StakedAmt:      uint256.NewInt(0).SetBytes(stakedAmt),
		ElectedTime:    uint256.NewInt(0).SetBytes(electedTime),
		OldElectedTime: uint256.NewInt(0).SetBytes(oldElectedTime),
		NominatedByOps: uint256.NewInt(0).SetBytes(nominatedBy),
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

func ElectMonitors(ctx *mevmtypes.Context, powNominations map[[33]byte]int64, blockTime int64, logger log.Logger) int {
	return electMonitors(ctx, param.MonitorsGovSequence, powNominations, blockTime, logger)
}
func ElectMonitorsForUT(ctx *mevmtypes.Context, seq uint64,
	powNominations map[[33]byte]int64, blockTime int64, logger log.Logger,
) int {
	return electMonitors(ctx, seq, powNominations, blockTime, logger)
}
func electMonitors(ctx *mevmtypes.Context, seq uint64,
	powNominations map[[33]byte]int64, blockTime int64, logger log.Logger,
) int {
	logger.Info("elect monitors ...")
	monitorInfos := ReadMonitorInfos(ctx, seq)
	logger.Info("allMonitorInfos", "json", toJSON(monitorInfos))
	logger.Info("allPowNominations", "json", toJSON(powNominations))

	// get and sort current monitors
	currMonitors := getCurrMonitors(monitorInfos, powNominations)
	sortMonitorInfosDesc(currMonitors)
	logger.Info("currMonitors", "json", toJSON(currMonitors))

	// get and sort eligible new candidates
	newMonitorCandidates := getNewMonitorCandidates(monitorInfos, powNominations)
	sortMonitorInfosDesc(newMonitorCandidates)
	logger.Info("newMonitorCandidates", "json", toJSON(newMonitorCandidates))

	// first election ?
	if len(currMonitors) == 0 {
		logger.Info("first monitors election")
		if len(newMonitorCandidates) < param.MonitorsCount {
			logger.Info("not enough candidates for the first election!")
			return MonitorElectionNotEnoughCandidates
		}
		newMonitors := newMonitorCandidates[:param.MonitorsCount]
		markMonitorElectedFlags(newMonitors, true)
		updateMonitorElectedTimes(ctx, seq, blockTime, monitorInfos)
		WriteMonitorsLastElectionTime(ctx, seq, uint64(blockTime))
		return OperatorElectionOK
	}

	if len(newMonitorCandidates) == 0 {
		logger.Info("no new eligible monitor candidates!")
		return MonitorElectionNoNewCandidates
	}

	// kick out needless candidates
	if len(newMonitorCandidates) > param.MonitorsMaxChangeCount {
		newMonitorCandidates = newMonitorCandidates[:param.MonitorsMaxChangeCount]
	}

	if monitorInfoLessFn(newMonitorCandidates[0], currMonitors[len(currMonitors)-1]) {
		logger.Info("monitor set not changed")
		return MonitorElectionNotChanged
	}

	allCandidates := append(currMonitors, newMonitorCandidates...)
	sortMonitorInfosDesc(allCandidates)
	markMonitorElectedFlags(allCandidates[:param.MonitorsCount], true)
	markMonitorElectedFlags(allCandidates[param.MonitorsCount:], false)
	updateMonitorElectedTimes(ctx, seq, blockTime, monitorInfos)
	WriteMonitorsLastElectionTime(ctx, seq, uint64(blockTime))
	logger.Info("new monitor set", "json", allCandidates)
	return MonitorElectionOK
}
func getCurrMonitors(allMonitorInfos []*MonitorInfo, powNominations map[[33]byte]int64) []*MonitorInfo {
	monitors := make([]*MonitorInfo, 0, param.MonitorsCount)
	for _, monitorInfo := range allMonitorInfos {
		if monitorInfo.ElectedTime.GtUint64(0) {
			monitorInfo.powNominatedCount = getPowNomination(powNominations, monitorInfo)
			monitors = append(monitors, monitorInfo)
		}
	}
	return monitors
}
func getNewMonitorCandidates(allMonitorInfos []*MonitorInfo, powNominations map[[33]byte]int64) []*MonitorInfo {
	candidates := make([]*MonitorInfo, 0, len(allMonitorInfos))
	for _, monitorInfo := range allMonitorInfos {
		if monitorInfo.ElectedTime.GtUint64(0) {
			// skip current monitors
			continue
		}
		powNominatedCount := getPowNomination(powNominations, monitorInfo)
		if powNominatedCount > 0 && isEligibleMonitorCandidate(monitorInfo) {
			monitorInfo.powNominatedCount = powNominatedCount
			candidates = append(candidates, monitorInfo)
		}
	}
	return candidates
}
func isEligibleMonitorCandidate(monitorInfo *MonitorInfo) bool {
	return !monitorInfo.StakedAmt.Lt(monitorMinStakedAmt) &&
		!monitorInfo.NominatedByOps.LtUint64(param.MonitorMinOpsNomination)
	// TODO: check more fields ?
}
func getPowNomination(powNominations map[[33]byte]int64, monitorInfo *MonitorInfo) int64 {
	var pubKey [33]byte
	copy(pubKey[:], monitorInfo.Pubkey)
	return powNominations[pubKey]
}
func sortMonitorInfosDesc(monitorInfos []*MonitorInfo) {
	sort.Slice(monitorInfos, func(i, j int) bool {
		return !monitorInfoLessFn(monitorInfos[i], monitorInfos[j])
	})
}
func monitorInfoLessFn(a, b *MonitorInfo) bool {
	if a.powNominatedCount != b.powNominatedCount {
		return a.powNominatedCount < b.powNominatedCount
	}
	if x := a.StakedAmt.Cmp(b.StakedAmt); x != 0 {
		return x < 0
	}
	if x := a.NominatedByOps.Cmp(b.NominatedByOps); x != 0 {
		return x < 0
	}
	return bytes.Compare(a.Addr[:], b.Addr[:]) < 0
}
func markMonitorElectedFlags(monitorInfos []*MonitorInfo, flag bool) {
	for _, monitorInfo := range monitorInfos {
		monitorInfo.electedFlag = flag
	}
}
func updateMonitorElectedTimes(ctx *mevmtypes.Context, seq uint64,
	blockTime int64, monitorInfos []*MonitorInfo) {

	for idx, monitorInfo := range monitorInfos {
		WriteMonitorOldElectedTime(ctx, seq, uint64(idx), monitorInfo.ElectedTime.Uint64())
		if !monitorInfo.electedFlag {
			WriteMonitorElectedTime(ctx, seq, uint64(idx), 0)
		} else {
			WriteMonitorElectedTime(ctx, seq, uint64(idx), uint64(blockTime))
		}
	}
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
func GetOldMonitorPubkeySet(ctx *mevmtypes.Context) (pubkeys [][]byte) {
	monitorInfos := ReadMonitorInfos(ctx, param.MonitorsGovSequence)
	for _, monitorInfo := range monitorInfos {
		if monitorInfo.OldElectedTime.Uint64() > 0 {
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

func toJSON(v any) string {
	bz, _ := json.Marshal(v)
	return string(bz)
}
