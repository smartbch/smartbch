package crosschain

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/smartbch/moeingevm/types"
)

const (
	VotingContractSeq = 0 // TODO
	OperatorsSlot     = 0
	MonitorsSlot      = 1
)

/*
	struct OperatorOrMonitorInfo {
		address addr;              // address
		uint    pubkeyPrefix;      // 0x02 or 0x03
		bytes32 pubkeyX;           // x
		bytes32 rpcUrl;            // ip:port (not used by monitors)
		bytes32 intro;             // introduction
		uint    totalStakedAmt;    // in BCH
		uint    selfStakedAmt;     // in BCH
		uint    inOfficeStartTime; // 0 means not in office, this field is set from Golang
	}
*/
type OperatorOrMonitorInfo struct {
	Addr              gethcmn.Address
	Pubkey            []byte // 33 bytes
	RpcUrl            []byte // 32 bytes
	Intro             []byte // 32 bytes
	TotalStakedAmt    *uint256.Int
	SelfStakedAmt     *uint256.Int
	InOfficeStartTime *uint256.Int
}

func ReadOperators(ctx *types.Context) []OperatorOrMonitorInfo {
	return ReadOperatorOrMonitorArr(ctx, VotingContractSeq, OperatorsSlot)
}
func ReadMonitors(ctx *types.Context) []OperatorOrMonitorInfo {
	return ReadOperatorOrMonitorArr(ctx, VotingContractSeq, MonitorsSlot)
}

func ReadOperatorOrMonitorArr(ctx *types.Context, seq uint64, slot uint64) (result []OperatorOrMonitorInfo) {
	arrSlot := uint256.NewInt(slot).PaddedBytes(32)
	arrLen := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(arrSlot)))
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))

	for i := uint64(0); i < arrLen.ToBig().Uint64(); i++ {
		result = append(result, readOperatorOrMonitorInfo(ctx, seq, arrLoc))
	}
	return
}

func readOperatorOrMonitorInfo(ctx *types.Context, seq uint64, loc *uint256.Int) OperatorOrMonitorInfo {
	addr := gethcmn.BytesToAddress(ctx.GetStorageAt(seq, string(loc.PaddedBytes(32))))
	pubkeyPrefix := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	pubkeyX := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	rpcUrl := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	intro := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	selfStakedAmt := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))))
	totalStakedAmt := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))))
	key := loc.AddUint64(loc, 1).PaddedBytes(32)
	//println("inOfficeStartTimeKey:", hex.EncodeToString(key))
	inOfficeStartTime := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(key)))
	//println("inOfficeStartTime:", inOfficeStartTime.Uint64())
	loc.AddUint64(loc, 1)
	return OperatorOrMonitorInfo{
		Addr:              addr,
		Pubkey:            append(pubkeyPrefix[31:], pubkeyX...),
		RpcUrl:            rpcUrl[:],
		Intro:             intro[:],
		TotalStakedAmt:    totalStakedAmt,
		SelfStakedAmt:     selfStakedAmt,
		InOfficeStartTime: inOfficeStartTime,
	}
}

func WriteOperatorInOfficeStartTime(ctx *types.Context, idx uint64, val uint64) {
	WriteOperatorOrMonitorInOfficeStartTime(ctx, VotingContractSeq, OperatorsSlot, idx, val)
}
func WriteMonitorInOfficeStartTime(ctx *types.Context, idx uint64, val uint64) {
	WriteOperatorOrMonitorInOfficeStartTime(ctx, VotingContractSeq, MonitorsSlot, idx, val)
}

func WriteOperatorOrMonitorInOfficeStartTime(ctx *types.Context, seq uint64, slot uint64, idx uint64, val uint64) {
	arrSlot := uint256.NewInt(slot).PaddedBytes(32)
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))
	itemLoc := uint256.NewInt(0).AddUint64(arrLoc, idx*8)
	fieldLoc := uint256.NewInt(0).AddUint64(itemLoc, 7)
	//println("kkk:", hex.EncodeToString(fieldLoc.PaddedBytes(32)), val)
	ctx.SetStorageAt(seq, string(fieldLoc.PaddedBytes(32)),
		uint256.NewInt(val).PaddedBytes(32))
}
