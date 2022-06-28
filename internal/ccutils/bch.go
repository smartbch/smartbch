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

func GetMultiSigP2SHAddr(redeemScriptWithoutConstructorArgs string,
	operatorPks []string, monitorPks []string) (string, error) {

	redeemScript, err := GetMultiSigRedeemScript(redeemScriptWithoutConstructorArgs, operatorPks, monitorPks)
	if err != nil {
		return "", err
	}
	//println("redeemScript:", hex.EncodeToString(redeemScript))

	// calculate the hash160 of the redeem script
	redeemHash := bchutil.Hash160(redeemScript)
	//println("redeemScriptHash160:", hex.EncodeToString(redeemHash))

	// if using Bitcoin main net then pass &chaincfg.MainNetParams as second argument
	addr, err := bchutil.NewAddressScriptHashFromHash(redeemHash, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}

	return addr.EncodeAddress(), nil
}

func GetMultiSigRedeemScript(redeemScriptWithoutConstructorArgs string,
	operatorPks []string, monitorPks []string) ([]byte, error) {

	builder := txscript.NewScriptBuilder()

	for i := len(monitorPks) - 1; i >= 0; i-- {
		pk, err := hex.DecodeString(monitorPks[i])
		if err != nil {
			return nil, fmt.Errorf("failed to decode monitorPk#%d", i)
		}
		if len(pk) != 33 {
			return nil, fmt.Errorf("len of monitorPk#%d is not 33", i)
		}

		builder.AddData(pk)
	}

	for i := len(operatorPks) - 1; i >= 0; i-- {
		pk, err := hex.DecodeString(operatorPks[i])
		if err != nil {
			return nil, fmt.Errorf("failed to decode operatorPk#%d", i)
		}
		if len(pk) != 33 {
			return nil, fmt.Errorf("len of operatorPk#%d is not 33", i)
		}

		builder.AddData(pk)
	}

	ops, err := hex.DecodeString(redeemScriptWithoutConstructorArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to decode redeemScriptWithoutConstructorArgs")
	}
	builder.AddOps(ops)

	return builder.Script()
}

func MakeMultiSigUnsignedRedeemTx(txid string, vout uint32, toAddr string, outAmt int64) (*wire.MsgTx, error) {
	//prevOutValue := int64(10000)
	//redeemOutValue := int64(6000)

	redeemTx := wire.NewMsgTx(wire.TxVersion)

	// you should provide your UTXO hash
	utxoHash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		return nil, err
	}

	// and add the index of the UTXO
	outPoint := wire.NewOutPoint(utxoHash, vout)
	txIn := wire.NewTxIn(outPoint, nil)
	redeemTx.AddTxIn(txIn)

	// adding the output to tx
	decodedAddr, err := bchutil.DecodeAddress(toAddr, &chaincfg.TestNet3Params)
	if err != nil {
		return nil, err
	}
	destinationAddrByte, err := txscript.PayToAddrScript(decodedAddr)
	if err != nil {
		return nil, err
	}

	// adding the destination address and the amount to the transaction
	redeemTxOut := wire.NewTxOut(outAmt, destinationAddrByte)
	redeemTx.AddTxOut(redeemTxOut)

	//buf := bytes.NewBuffer(make([]byte, 0, redeemTx.SerializeSize()))
	//_ = redeemTx.Serialize(buf)
	return redeemTx, nil
}

//const (
//	inputIdx = 0
//	hashType = txscript.SigHashAll | txscript.SigHashForkID
//)

func GetSigHash(tx *wire.MsgTx, idx int, subScript []byte,
	hashType txscript.SigHashType, prevOutAmt int64) ([]byte, error) {

	// If the forkID was not passed in with the hashtype then add it here
	if hashType&txscript.SigHashForkID != txscript.SigHashForkID {
		hashType |= txscript.SigHashForkID
	}

	sigHashes := txscript.NewTxSigHashes(tx)
	hash, err := txscript.CalcSignatureHash(subScript, sigHashes, hashType, tx, idx, prevOutAmt, true)
	return hash, err
}

//func SignRedeemTx(redeemTx *wire.MsgTx, redeemScript []byte, prevOutValue int64, key *bchec.PrivateKey) ([]byte, error) {
//	return txscript.RawTxInECDSASignature(redeemTx, 0, redeemScript, txscript.SigHashAll|txscript.SigHashForkID, key, prevOutValue)
//}

func SignRedeemTxSigHashECDSA(wifStr string, hash []byte, hashType txscript.SigHashType) ([]byte, error) {
	wif, err := bchutil.DecodeWIF(wifStr)
	if err != nil {
		return nil, err
	}

	signature, err := wif.PrivKey.SignECDSA(hash)
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}

	return append(signature.Serialize(), byte(hashType)), nil
}

func FixMultiSigUnsignedRedeemTx(redeemTx *wire.MsgTx, redeemScript []byte,
	sigs [][]byte, pkh []byte) (string, error) {

	signature := txscript.NewScriptBuilder()
	//signature.AddOp(txscript.OP_FALSE)
	signature.AddData(pkh)
	for i := len(sigs) - 1; i >= 0; i-- {
		signature.AddData(sigs[i])
	}
	signature.AddInt64(0) // selector
	signature.AddData(redeemScript)
	signatureScript, err := signature.Script()
	if err != nil {
		// Handle the error.
		return "", err
	}

	redeemTx.TxIn[0].SignatureScript = signatureScript

	var signedTx bytes.Buffer
	_ = redeemTx.Serialize(&signedTx)

	hexSignedTx := hex.EncodeToString(signedTx.Bytes())

	return hexSignedTx, nil
}
