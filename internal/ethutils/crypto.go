package ethutils

import (
	"crypto/ecdsa"
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func HexToPrivKey(key string) (*ecdsa.PrivateKey, []byte, error) {
	key = strings.TrimSpace(key)
	if strings.HasPrefix(key, "0x") {
		key = key[2:]
	}
	data, err := hex.DecodeString(key)
	if err != nil {
		return nil, nil, err
	}
	privKey, err := crypto.ToECDSA(data)
	return privKey, data, err
}

func PrivKeyToAddr(key *ecdsa.PrivateKey) common.Address {
	return crypto.PubkeyToAddress(key.PublicKey)
}
