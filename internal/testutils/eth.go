package testutils

import (
	"crypto/ecdsa"
	"math/big"

	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/smartbch/smartbch/internal/ethutils"
)

func MustEncodeTx(tx *gethtypes.Transaction) []byte {
	if data, err := ethutils.EncodeTx(tx); err == nil {
		return data
	} else {
		panic(err)
	}
}

func MustSignTx(tx *gethtypes.Transaction,
	chainID *big.Int, privKey string) *gethtypes.Transaction {

	key := MustHexToPrivKey(privKey)
	if tx, err := ethutils.SignTx(tx, chainID, key); err == nil {
		return tx
	} else {
		panic(err)
	}
}

func MustHexToPrivKey(key string) *ecdsa.PrivateKey {
	if k, _, err := ethutils.HexToPrivKey(key); err == nil {
		return k
	} else {
		panic(err)
	}
}
