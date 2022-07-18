package crosschain

import (
	"bytes"
	"encoding/hex"

	"github.com/gcash/bchd/txscript"
	"github.com/holiman/uint256"

	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"

	"github.com/smartbch/smartbch/crosschain/types"
)

// build tx

func buildUnsignedTx(utxo types.UTXO, redeemScript []byte, p2shHash [20]byte) (string, error) {
	tx := wire.NewMsgTx(wire.TxVersion)
	// 1. build tx output
	pkScript, err := buildP2shPubkeyScript(p2shHash[:])
	if err != nil {
		panic(err)
	}
	amount := uint256.NewInt(0).Sub(uint256.NewInt(0).SetBytes32(utxo.Amount[:]), uint256.NewInt(uint64(FixedMainnetFee)))
	amount.Div(amount, uint256.NewInt(10e10))
	tx.AddTxOut(wire.NewTxOut(int64(amount.Uint64()), pkScript))
	// 2. build tx input
	in := wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  utxo.TxID,
			Index: utxo.Index,
		},
		SignatureScript: redeemScript, //store redeem script here
		Sequence:        0xffffffff,
	}
	tx.AddTxIn(&in)
	return txSerialize2Hex(tx), nil
}

func buildP2shPubkeyScript(scriptHash []byte) ([]byte, error) {
	return txscript.NewScriptBuilder().AddOp(txscript.OP_HASH160).AddData(scriptHash).AddOp(txscript.OP_EQUAL).Script()
}

func buildMultiSigRedeemScript(pubkeys []*bchutil.AddressPubKey, n int) ([]byte, error) {
	return txscript.MultiSigScript(pubkeys, n)
}

func txSerialize2Hex(tx *wire.MsgTx) string {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf.Bytes())
}
