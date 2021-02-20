package ethutils

import (
	"crypto/ecdsa"
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func MustHexToPrivKey(key string) *ecdsa.PrivateKey {
	if k, err := HexToPrivKey(key); err == nil {
		return k
	} else {
		panic(err)
	}
}

func HexToPrivKey(key string) (*ecdsa.PrivateKey, error) {
	if strings.HasPrefix(key, "0x") {
		key = key[2:]
	}
	data, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}
	return crypto.ToECDSA(data)
}

func PrivKeyToAddr(key *ecdsa.PrivateKey) common.Address {
	return crypto.PubkeyToAddress(key.PublicKey)
}
