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

func DeleteUTXORecord(ctx *mevmtypes.Context, txid [32]byte, index uint32) {
	ctx.DeleteStorageAt(ccContractSequence, buildUTXOKey(txid, index))
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
