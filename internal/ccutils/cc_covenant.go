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
	minerFee                           int64
	net                                *chaincfg.Params
}

func NewCcCovenant(
	redeemScriptWithoutConstructorArgs []byte,
	operatorPks [][]byte,
	monitorPks [][]byte,
	minerFee int64,
	net *chaincfg.Params,
) (*CcCovenant, error) {

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
		minerFee:                           minerFee,
		net:                                net,
	}
	return ccc, nil
}

/* P2SH address */

func (c CcCovenant) BuildFullRedeemScript() ([]byte, error) {
	operatorPubkeysHash := bchutil.Hash160(bytes.Join(c.operatorPks, nil))
	monitorPubkeysHash := bchutil.Hash160(bytes.Join(c.monitorPks, nil))

	builder := txscript.NewScriptBuilder()
	builder.AddData(operatorPubkeysHash)
	builder.AddData(monitorPubkeysHash)
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

	return c.net.CashAddressPrefix + ":" + addr.EncodeAddress(), nil
}

/* redeem by user */

func (c CcCovenant) BuildRedeemByUserUnsignedTx(
	txid string, vout uint32, inAmt int64, // input info
	toAddr string, // output info
) (*wire.MsgTx, error) {

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
	txOut := wire.NewTxOut(inAmt-c.minerFee, destinationAddrByte)
	redeemTx.AddTxOut(txOut)

	//buf := bytes.NewBuffer(make([]byte, 0, redeemTx.SerializeSize()))
	//_ = redeemTx.Serialize(buf)
	return redeemTx, nil
}

func (c CcCovenant) GetRedeemByUserTxSigHash(
	txid string, vout uint32, inAmt int64, toAddr string) (*wire.MsgTx, []byte, error) {

	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, nil, err
	}

	tx, err := c.BuildRedeemByUserUnsignedTx(txid, vout, inAmt, toAddr)
	if err != nil {
		return nil, nil, err
	}

	sigHashes := txscript.NewTxSigHashes(tx)
	hashType := txscript.SigHashAll | txscript.SigHashForkID
	inputIdx := 0
	hash, err := txscript.CalcSignatureHash(redeemScript, sigHashes, hashType, tx, inputIdx, inAmt, true)
	return tx, hash, err
}

func (c CcCovenant) FinishRedeemByUserTx(unsignedTx *wire.MsgTx, sigs [][]byte) (string, error) {
	sigScript, err := c.BuildRedeemOrConvertUnlockingScript(sigs)
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

func (c *CcCovenant) BuildRedeemOrConvertUnlockingScript(sigs [][]byte) ([]byte, error) {
	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, err
	}

	builder := txscript.NewScriptBuilder()
	builder.AddData(bchutil.Hash160(bytes.Join(c.operatorPks, nil)))
	builder.AddData(bchutil.Hash160(bytes.Join(c.monitorPks, nil)))
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

/* convert by operators */

func (c CcCovenant) BuildConvertByOperatorsUnsignedTx(
	txid string, vout uint32, inAmt int64, // input info
	newOperatorPks [][]byte,
	newMonitorPks [][]byte,
) (*wire.MsgTx, error) {

	redeemTx := wire.NewMsgTx(2)

	// input
	utxoHash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		return nil, err
	}
	outPoint := wire.NewOutPoint(utxoHash, vout)
	txIn := wire.NewTxIn(outPoint, nil)
	redeemTx.AddTxIn(txIn)

	// toAddr
	c2, err := NewCcCovenant(c.redeemScriptWithoutConstructorArgs, newOperatorPks, newMonitorPks, c.minerFee, c.net)
	if err != nil {
		return nil, err
	}
	toAddr, err := c2.GetP2SHAddress()
	if err != nil {
		return nil, err
	}

	// output
	decodedAddr, err := bchutil.DecodeAddress(toAddr, c.net)
	if err != nil {
		return nil, err
	}
	destinationAddrByte, err := txscript.PayToAddrScript(decodedAddr)
	if err != nil {
		return nil, err
	}
	txOut := wire.NewTxOut(inAmt-c.minerFee, destinationAddrByte)
	redeemTx.AddTxOut(txOut)

	//buf := bytes.NewBuffer(make([]byte, 0, redeemTx.SerializeSize()))
	//_ = redeemTx.Serialize(buf)
	return redeemTx, nil
}

func (c CcCovenant) GetConvertByOperatorsTxSigHash(
	txid string, vout uint32, inAmt int64, // input info
	newOperatorPks [][]byte,
	newMonitorPks [][]byte,
) (*wire.MsgTx, []byte, error) {

	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, nil, err
	}

	tx, err := c.BuildConvertByOperatorsUnsignedTx(txid, vout, inAmt, newOperatorPks, newMonitorPks)
	if err != nil {
		return nil, nil, err
	}

	sigHashes := txscript.NewTxSigHashes(tx)
	hashType := txscript.SigHashAll | txscript.SigHashForkID
	inputIdx := 0
	hash, err := txscript.CalcSignatureHash(redeemScript, sigHashes, hashType, tx, inputIdx, inAmt, true)
	return tx, hash, err
}

func (c CcCovenant) FinishConvertByOperatorsTx(unsignedTx *wire.MsgTx, sigs [][]byte,
	newOperatorPks [][]byte,
	newMonitorPks [][]byte,
) (string, error) {

	sigScript, err := c.BuildConvertByOperatorsUnlockingScript(sigs, newOperatorPks, newMonitorPks)
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

func (c *CcCovenant) BuildConvertByOperatorsUnlockingScript(sigs [][]byte,
	newOperatorPks [][]byte,
	newMonitorPks [][]byte,
) ([]byte, error) {

	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, err
	}

	builder := txscript.NewScriptBuilder()
	builder.AddData(bchutil.Hash160(bytes.Join(newOperatorPks, nil)))
	builder.AddData(bchutil.Hash160(bytes.Join(newMonitorPks, nil)))
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

/* convert by monitors */

func (c CcCovenant) BuildConvertByMonitorsUnsignedTx(
	txid string, vout uint32, inAmt int64, // input info
	txid2 string, vout2 uint32, inAmt2 int64, // miner fee
	changeAddr string,
	newOperatorPks [][]byte,
) (*wire.MsgTx, error) {

	redeemTx := wire.NewMsgTx(2)

	// input1
	utxoHash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		return nil, err
	}
	outPoint := wire.NewOutPoint(utxoHash, vout)
	txIn := wire.NewTxIn(outPoint, nil)
	redeemTx.AddTxIn(txIn)

	// input2
	utxoHash2, err := chainhash.NewHashFromStr(txid2)
	if err != nil {
		return nil, err
	}
	outPoint2 := wire.NewOutPoint(utxoHash2, vout2)
	txIn2 := wire.NewTxIn(outPoint2, nil)
	redeemTx.AddTxIn(txIn2)

	// toAddr
	c2, err := NewCcCovenant(c.redeemScriptWithoutConstructorArgs, newOperatorPks, c.monitorPks, c.minerFee, c.net)
	if err != nil {
		return nil, err
	}
	toAddr, err := c2.GetP2SHAddress()
	if err != nil {
		return nil, err
	}

	// output1
	decodedAddr, err := bchutil.DecodeAddress(toAddr, c.net)
	if err != nil {
		return nil, err
	}
	destinationAddrByte, err := txscript.PayToAddrScript(decodedAddr)
	if err != nil {
		return nil, err
	}
	txOut := wire.NewTxOut(inAmt-c.minerFee, destinationAddrByte)
	redeemTx.AddTxOut(txOut)

	// output2
	decodedAddr2, err := bchutil.DecodeAddress(changeAddr, c.net)
	if err != nil {
		return nil, err
	}
	destinationAddrByte2, err := txscript.PayToAddrScript(decodedAddr2)
	if err != nil {
		return nil, err
	}
	txOut2 := wire.NewTxOut(inAmt2-c.minerFee, destinationAddrByte2)
	redeemTx.AddTxOut(txOut2)

	//buf := bytes.NewBuffer(make([]byte, 0, redeemTx.SerializeSize()))
	//_ = redeemTx.Serialize(buf)
	return redeemTx, nil
}

func (c CcCovenant) GetConvertByMonitorsTxSigHash(
	txid string, vout uint32, inAmt int64, // input info
	txid2 string, vout2 uint32, inAmt2 int64, // miner fee
	changeAddr string,
	newOperatorPks [][]byte,
) (*wire.MsgTx, []byte, error) {

	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, nil, err
	}

	tx, err := c.BuildConvertByMonitorsUnsignedTx(txid, vout, inAmt,
		txid2, vout2, inAmt2, changeAddr, newOperatorPks)
	if err != nil {
		return nil, nil, err
	}

	sigHashes := txscript.NewTxSigHashes(tx)
	hashType := txscript.SigHashSingle | txscript.SigHashAnyOneCanPay | txscript.SigHashForkID
	inputIdx := 0
	hash, err := txscript.CalcSignatureHash(redeemScript, sigHashes, hashType, tx, inputIdx, inAmt, true)
	return tx, hash, err
}

func (c CcCovenant) FinishConvertByMonitorsTx(unsignedTx *wire.MsgTx, sigs [][]byte,
	newOperatorPks [][]byte,
) (string, error) {

	sigScript, err := c.BuildConvertByMonitorsUnlockingScript(sigs, newOperatorPks)
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

func (c *CcCovenant) BuildConvertByMonitorsUnlockingScript(sigs [][]byte,
	newOperatorPks [][]byte,
) ([]byte, error) {

	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, err
	}

	builder := txscript.NewScriptBuilder()
	builder.AddData(bchutil.Hash160(bytes.Join(newOperatorPks, nil)))
	for i := len(c.monitorPks) - 1; i >= 0; i-- {
		builder.AddData(c.monitorPks[i])
	}
	for i := len(sigs) - 1; i >= 0; i-- {
		builder.AddData(sigs[i])
	}
	builder.AddInt64(1) // selector
	builder.AddData(redeemScript)
	return builder.Script()
}

/* signature */

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
