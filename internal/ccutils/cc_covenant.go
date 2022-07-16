package ccutils

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
)

type CcCovenant struct {
	redeemScriptWithoutConstructorArgs []byte
	operatorPks                        [][]byte
	monitorPks                         [][]byte
	net                                *chaincfg.Params
}

func NewCcCovenant(
	redeemScriptWithoutConstructorArgs []byte,
	operatorPks [][]byte,
	monitorPks [][]byte,
	net *chaincfg.Params) (*CcCovenant, error) {

	for i, pk := range operatorPks {
		if len(pk) != 33 {
			return nil, fmt.Errorf("operatorPk#%d is not 33 bytes", i)
		}
	}
	for i, pk := range monitorPks {
		if len(pk) != 33 {
			return nil, fmt.Errorf("monitorPk#%d is not 33 bytes", i)
		}
	}

	ccc := &CcCovenant{
		redeemScriptWithoutConstructorArgs: redeemScriptWithoutConstructorArgs,
		operatorPks:                        operatorPks,
		monitorPks:                         monitorPks,
		net:                                net,
	}
	return ccc, nil
}

func (c CcCovenant) BuildFullRedeemScript() ([]byte, error) {
	operatorPubkeysHash := bchutil.Hash160(bytes.Join(c.operatorPks, nil))
	monitorPubkeysHash := bchutil.Hash160(bytes.Join(c.monitorPks, nil))

	builder := txscript.NewScriptBuilder()
	builder.AddData(monitorPubkeysHash)
	builder.AddData(operatorPubkeysHash)
	builder.AddOps(c.redeemScriptWithoutConstructorArgs)

	return builder.Script()
}

func (c CcCovenant) GetP2SHAddress() (string, error) {
	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return "", err
	}

	redeemHash := bchutil.Hash160(redeemScript)
	addr, err := bchutil.NewAddressScriptHashFromHash(redeemHash, c.net)
	if err != nil {
		return "", err
	}

	return addr.EncodeAddress(), nil
}

func (c CcCovenant) BuildUnsignedRedeemTx(
	txid string, vout uint32, /*prevOutAmt int64,*/
	toAddr string, outAmt int64) (*wire.MsgTx, error) {

	redeemTx := wire.NewMsgTx(2)

	// input
	utxoHash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		return nil, err
	}
	outPoint := wire.NewOutPoint(utxoHash, vout)
	txIn := wire.NewTxIn(outPoint, nil)
	redeemTx.AddTxIn(txIn)

	// output
	decodedAddr, err := bchutil.DecodeAddress(toAddr, c.net)
	if err != nil {
		return nil, err
	}
	destinationAddrByte, err := txscript.PayToAddrScript(decodedAddr)
	if err != nil {
		return nil, err
	}
	txOut := wire.NewTxOut(outAmt, destinationAddrByte)
	redeemTx.AddTxOut(txOut)

	//buf := bytes.NewBuffer(make([]byte, 0, redeemTx.SerializeSize()))
	//_ = redeemTx.Serialize(buf)
	return redeemTx, nil
}

func (c CcCovenant) GetRedeemTxSigHash(
	txid string, vout uint32, prevOutAmt int64,
	toAddr string, outAmt int64) (*wire.MsgTx, []byte, error) {

	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, nil, err
	}

	tx, err := c.BuildUnsignedRedeemTx(txid, vout, toAddr, outAmt)
	if err != nil {
		return nil, nil, err
	}

	sigHashes := txscript.NewTxSigHashes(tx)
	hashType := txscript.SigHashAll | txscript.SigHashForkID
	inputIdx := 0
	hash, err := txscript.CalcSignatureHash(redeemScript, sigHashes, hashType, tx, inputIdx, prevOutAmt, true)
	return tx, hash, err
}

func (c CcCovenant) FinishRedeemTx(unsignedTx *wire.MsgTx,
	sigs [][]byte, pkh []byte) (string, error) {

	sigScript, err := c.BuildRedeemSigScript(sigs, pkh)
	if err != nil {
		return "", err
	}

	inputIdx := 0
	unsignedTx.TxIn[inputIdx].SignatureScript = sigScript

	var signedTx bytes.Buffer
	_ = unsignedTx.Serialize(&signedTx)

	hexSignedTx := hex.EncodeToString(signedTx.Bytes())
	return hexSignedTx, nil
}

func (c *CcCovenant) BuildRedeemSigScript(sigs [][]byte, pkh []byte) ([]byte, error) {
	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, err
	}

	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_TRUE)
	builder.AddData(pkh)
	for i := len(c.operatorPks) - 1; i >= 0; i-- {
		builder.AddData(c.operatorPks[i])
	}
	for i := len(sigs) - 1; i >= 0; i-- {
		builder.AddData(sigs[i])
	}
	builder.AddInt64(0) // selector
	builder.AddData(redeemScript)
	return builder.Script()
}

func SignCcCovenantTxSigHashECDSA(wifStr string, hash []byte, hashType txscript.SigHashType) ([]byte, error) {
	wif, err := bchutil.DecodeWIF(wifStr)
	if err != nil {
		return nil, err
	}

	signature, err := wif.PrivKey.SignECDSA(hash)
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}

	return append(signature.Serialize(), byte(hashType)), nil
}
