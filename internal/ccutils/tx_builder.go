package ccutils

import (
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
)

type msgTxBuilder struct {
	msgTx *wire.MsgTx
	net   *chaincfg.Params
}

func newMsgTxBuilder(net *chaincfg.Params) *msgTxBuilder {
	return &msgTxBuilder{
		msgTx: wire.NewMsgTx(2),
		net:   net,
	}
}

func (builder *msgTxBuilder) addInput(txid string, vout uint32) error {
	utxoHash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		return err
	}
	outPoint := wire.NewOutPoint(utxoHash, vout)
	txIn := wire.NewTxIn(outPoint, nil)
	builder.msgTx.AddTxIn(txIn)
	return nil
}

func (builder *msgTxBuilder) addOutput(toAddr string, outAmt int64) error {
	decodedAddr, err := bchutil.DecodeAddress(toAddr, builder.net)
	if err != nil {
		return err
	}
	destinationAddrByte, err := txscript.PayToAddrScript(decodedAddr)
	if err != nil {
		return err
	}
	txOut := wire.NewTxOut(outAmt, destinationAddrByte)
	builder.msgTx.AddTxOut(txOut)
	return nil
}
