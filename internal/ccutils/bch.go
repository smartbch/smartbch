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

func GetCcCovenantP2SHAddr(redeemScriptWithoutConstructorArgs string,
	operatorPks []string, monitorPks []string) (string, error) {

	redeemScript, err := GetCcCovenantFullRedeemScript(redeemScriptWithoutConstructorArgs, operatorPks, monitorPks)
	if err != nil {
		return "", err
	}
	//println("redeemScript:", hex.EncodeToString(redeemScript))

	// calculate the hash160 of the redeem script
	redeemHash := bchutil.Hash160(redeemScript)
	//println("redeemScriptHash160:", hex.EncodeToString(redeemHash))

	// if using Bitcoin main net then pass &chaincfg.MainNetParams as second argument
	addr, err := bchutil.NewAddressScriptHashFromHash(redeemHash, &chaincfg.TestNet3Params)
	if err != nil {
		return "", err
	}

	return addr.EncodeAddress(), nil
}

func GetCcCovenantFullRedeemScript(redeemScriptWithoutConstructorArgs string,
	operatorPks []string, monitorPks []string) ([]byte, error) {

	operatorPubkeys, err := joinHexPks(operatorPks)
	if err != nil {
		return nil, err
	}
	monitorPubkeys, err := joinHexPks(monitorPks)
	if err != nil {
		return nil, err
	}

	builder := txscript.NewScriptBuilder()
	builder.AddData(bchutil.Hash160(monitorPubkeys))
	builder.AddData(bchutil.Hash160(operatorPubkeys))

	ops, err := hex.DecodeString(redeemScriptWithoutConstructorArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to decode redeemScriptWithoutConstructorArgs")
	}
	builder.AddOps(ops)

	return builder.Script()
}

func joinHexPks(hexPks []string) ([]byte, error) {
	var allPkBytes []byte
	for i, pkHex := range hexPks {
		pkBytes, err := hex.DecodeString(pkHex)
		if err != nil {
			return nil, fmt.Errorf("failed to decode pk: %s", pkHex)
		}
		if len(pkBytes) != 33 {
			return nil, fmt.Errorf("len of pk#%d is not 33", i)
		}

		allPkBytes = append(allPkBytes, pkBytes...)
	}
	return allPkBytes, nil
}

func MakeCcCovenantUnsignedRedeemTx(txid string, vout uint32, toAddr string, outAmt int64) (*wire.MsgTx, error) {
	redeemTx := wire.NewMsgTx(2)

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

func SignCcCovenantTxSigHashECDSA(wifStr string, hash []byte, hashType txscript.SigHashType) ([]byte, error) {
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

func FixCcCovenantUnsignedRedeemTx(redeemTx *wire.MsgTx, redeemScript []byte,
	pks [][]byte, sigs [][]byte, pkh []byte) (string, error) {

	signature := txscript.NewScriptBuilder()
	signature.AddOp(txscript.OP_TRUE)
	signature.AddData(pkh)
	for i := len(pks) - 1; i >= 0; i-- {
		signature.AddData(pks[i])
	}
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
	println("sigScript:", hex.EncodeToString(signatureScript))

	redeemTx.TxIn[0].SignatureScript = signatureScript

	var signedTx bytes.Buffer
	_ = redeemTx.Serialize(&signedTx)

	hexSignedTx := hex.EncodeToString(signedTx.Bytes())

	return hexSignedTx, nil
}
