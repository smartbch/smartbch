package covenant

import (
	"bytes"
	"fmt"

	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
)

func MsgTxToBytes(tx *wire.MsgTx) []byte {
	var buf bytes.Buffer
	_ = tx.Serialize(&buf)
	return buf.Bytes()
}
func MsgTxFromBytes(data []byte) (*wire.MsgTx, error) {
	msg := &wire.MsgTx{}
	err := msg.Deserialize(bytes.NewReader(data))
	return msg, err
}

func SignCcCovenantTxSigHashECDSA(wifStr string, hash []byte, hashType txscript.SigHashType) ([]byte, error) {
	wif, err := bchutil.DecodeWIF(wifStr)
	if err != nil {
		return nil, err
	}
	return signCcCovenantTxSigHashECDSA(wif.PrivKey, hash, hashType)
}

func SignRedeemTxSigHashECDSA(privKey *bchec.PrivateKey, hash []byte) ([]byte, error) {
	hashType := txscript.SigHashAll | txscript.SigHashForkID
	return signCcCovenantTxSigHashECDSA(privKey, hash, hashType)
}

func signCcCovenantTxSigHashECDSA(privKey *bchec.PrivateKey,
	hash []byte, hashType txscript.SigHashType) ([]byte, error) {

	signature, err := privKey.SignECDSA(hash)
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}

	return append(signature.Serialize(), byte(hashType)), nil
}
