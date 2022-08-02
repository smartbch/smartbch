package ccutils

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
)

func msgTxToHex(tx *wire.MsgTx) string {
	var buf bytes.Buffer
	_ = tx.Serialize(&buf)
	return hex.EncodeToString(buf.Bytes())
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
