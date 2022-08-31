package covenant

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"

	"github.com/smartbch/smartbch/param"
)

type CcCovenant struct {
	redeemScriptWithoutConstructorArgs []byte
	operatorPks                        [][]byte
	monitorPks                         [][]byte
	minerFee                           int64
	net                                *chaincfg.Params
}

func NewDefaultCcCovenant(operatorPks, monitorPks [][]byte) (*CcCovenant, error) {
	hexStr := strings.TrimPrefix(param.RedeemScriptWithoutConstructorArgs, "0x")
	hexBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}

	var bchNet *chaincfg.Params
	if param.CcBchNetwork == chaincfg.MainNetParams.Name {
		bchNet = &chaincfg.MainNetParams
	} else if param.CcBchNetwork == chaincfg.TestNet3Params.Name {
		bchNet = &chaincfg.TestNet3Params
	} else {
		return nil, errors.New("unknown BCH network: " + param.CcBchNetwork)
	}

	return NewCcCovenant(hexBytes, operatorPks, monitorPks,
		param.RedeemOrCovertMinerFee, bchNet)
}

func NewCcCovenant(
	redeemScriptWithoutConstructorArgs []byte,
	operatorPks [][]byte,
	monitorPks [][]byte,
	minerFee int64,
	net *chaincfg.Params,
) (*CcCovenant, error) {

	if err := checkPks(operatorPks, monitorPks); err != nil {
		return nil, err
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

func checkPks(operatorPks [][]byte, monitorPks [][]byte) error {
	if len(operatorPks) != param.OperatorsCount {
		return errors.New("invalid operatorPks count")
	}
	if len(monitorPks) != param.MonitorsCount {
		return errors.New("invalid monitorsPks count")
	}

	for i, pk := range operatorPks {
		if len(pk) != 33 {
			return fmt.Errorf("operatorPk#%d is not 33 bytes", i)
		}
	}
	for i, pk := range monitorPks {
		if len(pk) != 33 {
			return fmt.Errorf("monitorPk#%d is not 33 bytes", i)
		}
	}
	return nil
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

func (c CcCovenant) GetP2SHAddress20() (addr [20]byte, err error) {
	redeemScript, err := c.BuildFullRedeemScript()
	if err == nil {
		redeemHash := bchutil.Hash160(redeemScript)
		copy(addr[:], redeemHash)
	}
	return
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

func (c CcCovenant) GetP2SHAddressNew(newOperatorPks, newMonitorPks [][]byte) (string, error) {
	c2, err := NewCcCovenant(c.redeemScriptWithoutConstructorArgs,
		newOperatorPks, newMonitorPks, c.minerFee, c.net)
	if err != nil {
		return "", err
	}
	return c2.GetP2SHAddress()
}

/* redeem by user */

func (c CcCovenant) BuildRedeemByUserUnsignedTx(
	txid []byte, vout uint32, inAmt int64, // input info
	toAddr string, // output info
) (*wire.MsgTx, error) {

	builder := newMsgTxBuilder(c.net)
	if err := builder.addInput(txid, vout); err != nil {
		return nil, err
	}
	if err := builder.addOutput(toAddr, inAmt-c.minerFee); err != nil {
		return nil, err
	}

	return builder.msgTx, nil
}

func (c CcCovenant) GetRedeemByUserTxSigHash(
	txid []byte, vout uint32, inAmt int64, // input info
	toAddr string, // output info
) (*wire.MsgTx, []byte, error) {

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

func (c CcCovenant) FinishRedeemByUserTx(
	unsignedTx *wire.MsgTx,
	sigs [][]byte,
) (string, error) {
	sigScript, err := c.BuildRedeemByUserUnlockingScript(sigs)
	if err != nil {
		return "", err
	}

	inputIdx := 0
	unsignedTx.TxIn[inputIdx].SignatureScript = sigScript
	return msgTxToHex(unsignedTx), nil
}

func (c *CcCovenant) BuildRedeemByUserUnlockingScript(sigs [][]byte) ([]byte, error) {
	return c.BuildRedeemOrConvertUnlockingScript(sigs, c.operatorPks, c.monitorPks)
}

func (c *CcCovenant) BuildRedeemOrConvertUnlockingScript(
	sigs [][]byte,
	newOperatorPks [][]byte,
	newMonitorPks [][]byte,
) ([]byte, error) {

	if len(sigs) != param.MinOperatorSigCount {
		return nil, errors.New("invalid operator signature count")
	}
	err := checkPks(newOperatorPks, newMonitorPks)
	if err != nil {
		return nil, err
	}

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

/* convert by operators */

func (c CcCovenant) BuildConvertByOperatorsUnsignedTx(
	txid []byte, vout uint32, inAmt int64, // input info
	newOperatorPks [][]byte,
	newMonitorPks [][]byte,
) (*wire.MsgTx, error) {

	// toAddr
	toAddr, err := c.GetP2SHAddressNew(newOperatorPks, newMonitorPks)
	if err != nil {
		return nil, err
	}

	builder := newMsgTxBuilder(c.net)
	if err = builder.addInput(txid, vout); err != nil {
		return nil, err
	}
	if err = builder.addOutput(toAddr, inAmt-c.minerFee); err != nil {
		return nil, err
	}

	return builder.msgTx, nil
}

func (c CcCovenant) GetConvertByOperatorsTxSigHash(
	txid []byte, vout uint32, inAmt int64, // input info
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

func (c CcCovenant) FinishConvertByOperatorsTx(
	unsignedTx *wire.MsgTx,
	sigs [][]byte,
	newOperatorPks [][]byte,
	newMonitorPks [][]byte,
) (string, error) {

	sigScript, err := c.BuildConvertByOperatorsUnlockingScript(sigs, newOperatorPks, newMonitorPks)
	if err != nil {
		return "", err
	}

	inputIdx := 0
	unsignedTx.TxIn[inputIdx].SignatureScript = sigScript
	return msgTxToHex(unsignedTx), nil
}

func (c *CcCovenant) BuildConvertByOperatorsUnlockingScript(
	sigs [][]byte,
	newOperatorPks [][]byte,
	newMonitorPks [][]byte,
) ([]byte, error) {
	// TODO: check newOperatorPks & newMonitorPks
	if reflect.DeepEqual(newOperatorPks, c.operatorPks) &&
		reflect.DeepEqual(newMonitorPks, c.monitorPks) {

		return nil, fmt.Errorf("operators and monitors not changed")
	}

	return c.BuildRedeemOrConvertUnlockingScript(sigs, newOperatorPks, newMonitorPks)
}

/* convert by monitors */

func (c CcCovenant) BuildConvertByMonitorsUnsignedTx(
	txid []byte, vout uint32, inAmt int64, // input info
	newOperatorPks [][]byte,
) (*wire.MsgTx, error) {

	// toAddr
	toAddr, err := c.GetP2SHAddressNew(newOperatorPks, c.monitorPks)
	if err != nil {
		return nil, err
	}

	builder := newMsgTxBuilder(c.net)
	if err = builder.addInput(txid, vout); err != nil {
		return nil, err
	}
	builder.msgTx.TxIn[0].Sequence = param.MonitorTransferWaitBlocks
	if err = builder.addOutput(toAddr, inAmt); err != nil {
		return nil, err
	}

	return builder.msgTx, nil
}

func (c CcCovenant) GetConvertByMonitorsTxSigHash(
	txid []byte, vout uint32, inAmt int64, // input info
	newOperatorPks [][]byte,
) (*wire.MsgTx, []byte, error) {

	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, nil, err
	}

	tx, err := c.BuildConvertByMonitorsUnsignedTx(txid, vout, inAmt, newOperatorPks)
	if err != nil {
		return nil, nil, err
	}

	sigHashes := txscript.NewTxSigHashes(tx)
	hashType := txscript.SigHashSingle | txscript.SigHashAnyOneCanPay | txscript.SigHashForkID
	inputIdx := 0
	hash, err := txscript.CalcSignatureHash(redeemScript, sigHashes, hashType, tx, inputIdx, inAmt, true)
	return tx, hash, err
}

func (c CcCovenant) AddConvertByMonitorsTxMonitorSigs(
	unsignedTx *wire.MsgTx,
	sigs [][]byte, newOperatorPks [][]byte,
) (*wire.MsgTx, error) {

	sigScript, err := c.BuildConvertByMonitorsUnlockingScript(sigs, newOperatorPks)
	if err != nil {
		return unsignedTx, err
	}

	inputIdx := 0
	unsignedTx.TxIn[inputIdx].SignatureScript = sigScript

	return unsignedTx, nil
}

func (c *CcCovenant) BuildConvertByMonitorsUnlockingScript(sigs [][]byte,
	newOperatorPks [][]byte,
) ([]byte, error) {

	if len(sigs) != param.MinMonitorSigCount {
		return nil, errors.New("invalid monitor signature count")
	}
	err := checkPks(newOperatorPks, c.monitorPks)
	if err != nil {
		return nil, err
	}

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

func (c CcCovenant) AddConvertByMonitorsTxMinerFee(
	signedTx *wire.MsgTx,
	txid []byte, vout uint32, inAmt int64, // input info
	minerFee int64, changeAddr string, // miner fee
) (*wire.MsgTx, error) {

	builder := wrapMsgTx(signedTx, c.net)
	if err := builder.addInput(txid, vout); err != nil {
		return signedTx, err
	}
	if inAmt > minerFee {
		if err := builder.addOutput(changeAddr, inAmt-minerFee); err != nil {
			return signedTx, err
		}
	}

	return signedTx, nil
}

func (c CcCovenant) GetConvertByMonitorsTxSigHash2(
	txWithMinerFee *wire.MsgTx,
	inAmt int64,
	addr string,
) ([]byte, error) {
	decodedAddr, err := bchutil.DecodeAddress(addr, c.net)
	if err != nil {
		return nil, err
	}

	pkScript, err := txscript.PayToAddrScript(decodedAddr) // locking script
	if err != nil {
		return nil, err
	}

	sigHashes := txscript.NewTxSigHashes(txWithMinerFee)
	hashType := txscript.SigHashAll | txscript.SigHashForkID
	inputIdx := 1
	hash, err := txscript.CalcSignatureHash(pkScript, sigHashes, hashType, txWithMinerFee, inputIdx, inAmt, true)
	return hash, err
}

func (c CcCovenant) AddConvertByMonitorsTxMinerFeeSig(
	txWithMinerFee *wire.MsgTx,
	sig, pkData []byte,
) (*wire.MsgTx, error) {

	sigScript, err := txscript.NewScriptBuilder().AddData(sig).AddData(pkData).Script()
	if err != nil {
		return txWithMinerFee, err
	}

	inputIdx := 1
	txWithMinerFee.TxIn[inputIdx].SignatureScript = sigScript

	return txWithMinerFee, nil
}
