package crosschain

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"

	mevmtypes "github.com/smartbch/moeingevm/types"

	"github.com/smartbch/smartbch/crosschain/types"
)

var (
	SlotContext string = strings.Repeat(string([]byte{0}), 31) + string([]byte{5})
)

func LoadUTXORecord(ctx *mevmtypes.Context, txid [32]byte, index uint32) *types.UTXORecord {
	bz := ctx.GetStorageAt(ccContractSequence, buildUTXOKey(txid, index))
	if len(bz) == 0 {
		return nil
	}
	var r types.UTXORecord
	_, err := r.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	return &r
}

func SaveUTXORecord(ctx *mevmtypes.Context, record types.UTXORecord) {
	bz, err := record.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(ccContractSequence, buildUTXOKey(record.Txid, record.Index), bz)
}

func LoadCCContext(ctx *mevmtypes.Context) *types.CCContext {
	bz := ctx.GetStorageAt(ccContractSequence, SlotContext)
	if len(bz) == 0 {
		return nil
	}
	var c types.CCContext
	_, err := c.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	return &c
}

func SaveCCContext(ctx *mevmtypes.Context, ccContext types.CCContext) {
	bz, err := ccContext.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(ccContractSequence, SlotContext, bz)
}

func buildUTXOKey(txid [32]byte, index uint32) string {
	var v [4]byte
	binary.BigEndian.PutUint32(v[:], index)
	hash := sha256.Sum256(append(txid[:], v[:]...))
	return string(hash[:])
}

func LoadCCInfo(ctx *mevmtypes.Context) (info types.CCInfo) {
	bz := ctx.GetStorageAt(ccContractSequence, SlotCCInfo)
	if bz == nil {
		return types.CCInfo{}
	}
	_, err := info.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	return
}

func SaveCCInfo(ctx *mevmtypes.Context, info types.CCInfo) {
	bz, err := info.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(ccContractSequence, SlotCCInfo, bz)
}

func LoadScriptInfo(ctx *mevmtypes.Context) (info types.ScriptInfo) {
	bz := ctx.GetStorageAt(ccContractSequence, SlotScriptInfo)
	if bz == nil {
		return types.ScriptInfo{}
	}
	_, err := info.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	return
}

func SaveScriptInfo(ctx *mevmtypes.Context, info types.ScriptInfo) {
	bz, err := info.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(ccContractSequence, SlotScriptInfo, bz)
}
