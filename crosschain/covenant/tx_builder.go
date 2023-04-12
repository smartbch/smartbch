package covenant

import (
	"encoding/hex"

	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
)

const (
	dustAmt = 546
)

type msgTxBuilder struct {
	msgTx *wire.MsgTx
	net   *chaincfg.Params
	err   error
}

func wrapMsgTx(msgTx *wire.MsgTx, net *chaincfg.Params) *msgTxBuilder {
	return &msgTxBuilder{
		msgTx: msgTx,
		net:   net,
	}
}
func newMsgTxBuilder(net *chaincfg.Params) *msgTxBuilder {
	return &msgTxBuilder{
		msgTx: wire.NewMsgTx(2),
		net:   net,
	}
}

func (builder *msgTxBuilder) addInput(txid []byte, vout uint32) *msgTxBuilder {
	// use NewHashFromStr() to byte-reverse txid !!!
	utxoHash, err := chainhash.NewHashFromStr(hex.EncodeToString(txid))
	if err != nil {
		builder.err = err
		return builder
	}

	outPoint := wire.NewOutPoint(utxoHash, vout)
	txIn := wire.NewTxIn(outPoint, nil)
	builder.msgTx.AddTxIn(txIn)
	return builder
}

func (builder *msgTxBuilder) addOutput(toAddr string, outAmt int64) *msgTxBuilder {
	decodedAddr, err := bchutil.DecodeAddress(toAddr, builder.net)
	if err != nil {
		builder.err = err
		return builder
	}

	destinationAddrByte, err := txscript.PayToAddrScript(decodedAddr)
	if err != nil {
		builder.err = err
		return builder
	}

	txOut := wire.NewTxOut(outAmt, destinationAddrByte)
	builder.msgTx.AddTxOut(txOut)
	return builder
}

func (builder *msgTxBuilder) addChange(toAddr string, changeAmt int64) *msgTxBuilder {
	if changeAmt > dustAmt {
		return builder.addOutput(toAddr, changeAmt)
	}
	return builder
}

func (builder *msgTxBuilder) build() (*wire.MsgTx, error) {
	return builder.msgTx, builder.err
}
