package ethutils

import (
	"crypto/ecdsa"
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func HexToPrivKey(key string) (*ecdsa.PrivateKey, []byte, error) {
	key = strings.TrimSpace(key)
	key = strings.TrimPrefix(key, "0x")
	data, err := hex.DecodeString(key)
	if err != nil {
		return nil, nil, err
	}
	privKey, err := crypto.ToECDSA(data)
	return privKey, data, err
}
