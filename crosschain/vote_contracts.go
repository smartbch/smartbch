package crosschain

import (
	"bytes"
	"encoding/json"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/libs/log"

	mevmtypes "github.com/smartbch/moeingevm/types"
	cctypes "github.com/smartbch/smartbch/crosschain/types"
)

// TODO: move to params.go
const (
	OperatorsGovSeq         = 0 // TODO
	OperatorsSlot           = 0
	OperatorWords           = 8
	OperatorsCount          = 10
	OperatorsMaxChangeCount = 3

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

func ElectOperators(ctx *mevmtypes.Context) {
	// TODO
}

func GetOperatorPubkeySet(ctx *mevmtypes.Context, seq uint64) (pubkeys [][33]byte) {
	operatorInfos := ReadOperatorInfos(ctx, seq)
	for _, operatorInfo := range operatorInfos {
		if operatorInfo.ElectedTime.Uint64() > 0 {
			var pubkey [33]byte
			copy(pubkey[:], operatorInfo.Pubkey)
			pubkeys = append(pubkeys, pubkey)
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
	if newElectedCount == MonitorsCount {
		logger.Info("first monitors election?")
		if lastTimeElectedCount != 0 {
			logger.Info("lastTimeElectedCount != 0")
		}
	}
	if newElectedCount > MonitorsMaxChangeCount {
		logger.Info("too many new monitors!",
			"newElectedCount", newElectedCount)
		return
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

func GetMonitorPubkeySet(ctx *mevmtypes.Context) (pubkeys [][33]byte) {
	monitorInfos := ReadMonitorInfos(ctx, MonitorsSlot)
	for _, monitorInfo := range monitorInfos {
		if monitorInfo.ElectedTime.Uint64() > 0 {
			var pubkey [33]byte
			copy(pubkey[:], monitorInfo.Pubkey)
			pubkeys = append(pubkeys, pubkey)
		}
	}
	return
}
