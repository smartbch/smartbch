package crosschain

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/smartbch/moeingevm/types"
)

const (
	OperatorsGovSeq = 0 // TODO
	OperatorsSlot   = 0
	OperatorWords   = 8

	MonitorsGovSeq              = 0 // TODO
	MonitorLastElectionTimeSlot = 0
	MonitorsSlot                = 1
	MonitorWords                = 6
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

func ReadOperatorArr(ctx *types.Context, seq uint64) (result []OperatorInfo) {
	arrSlot := uint256.NewInt(OperatorsSlot).PaddedBytes(32)
	arrLen := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(arrSlot)))
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))

	for i := uint64(0); i < arrLen.ToBig().Uint64(); i++ {
		result = append(result, readOperatorInfo(ctx, seq, arrLoc))
	}
	return
}
func readOperatorInfo(ctx *types.Context, seq uint64, loc *uint256.Int) OperatorInfo {
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

func WriteOperatorElectedTime(ctx *types.Context, seq uint64, operatorIdx uint64, val uint64) {
	arrSlot := uint256.NewInt(OperatorsSlot).PaddedBytes(32)
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))
	itemLoc := uint256.NewInt(0).AddUint64(arrLoc, operatorIdx*OperatorWords)
	fieldLoc := uint256.NewInt(0).AddUint64(itemLoc, OperatorWords-1)
	ctx.SetStorageAt(seq, string(fieldLoc.PaddedBytes(32)),
		uint256.NewInt(val).PaddedBytes(32))
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
}

func ReadMonitorArr(ctx *types.Context, seq uint64) (result []MonitorInfo) {
	arrSlot := uint256.NewInt(MonitorsSlot).PaddedBytes(32)
	arrLen := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(arrSlot)))
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))

	for i := uint64(0); i < arrLen.ToBig().Uint64(); i++ {
		result = append(result, readMonitorInfo(ctx, seq, arrLoc))
	}
	return
}
func readMonitorInfo(ctx *types.Context, seq uint64, loc *uint256.Int) MonitorInfo {
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

func WriteMonitorElectedTime(ctx *types.Context, seq uint64, monitorIdx uint64, val uint64) {
	arrSlot := uint256.NewInt(MonitorsSlot).PaddedBytes(32)
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))
	itemLoc := uint256.NewInt(0).AddUint64(arrLoc, monitorIdx*MonitorWords)
	fieldLoc := uint256.NewInt(0).AddUint64(itemLoc, MonitorWords-1)
	ctx.SetStorageAt(seq, string(fieldLoc.PaddedBytes(32)),
		uint256.NewInt(val).PaddedBytes(32))
}

func ReadMonitorsLastElectionTime(ctx *types.Context, seq uint64) *uint256.Int {
	slot := uint256.NewInt(MonitorLastElectionTimeSlot).PaddedBytes(32)
	val := ctx.GetStorageAt(seq, string(slot))
	return uint256.NewInt(0).SetBytes(val)
}
func WriteMonitorsLastElectionTime(ctx *types.Context, seq uint64, val uint64) {
	slot := uint256.NewInt(MonitorLastElectionTimeSlot).PaddedBytes(32)
	ctx.SetStorageAt(seq, string(slot), uint256.NewInt(val).PaddedBytes(32))
}
