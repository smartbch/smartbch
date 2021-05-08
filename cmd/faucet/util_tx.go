package main

import (
	"crypto/ecdsa"
	"math/big"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/smartbch/smartbch/internal/ethutils"
)

var (
	chainID  = big.NewInt(10001)
	gasPrice = big.NewInt(0)
	gasLimit = uint64(1000000)
	amt      = big.NewInt(10000000000000000)
)

func makeAndSignTx(privKey *ecdsa.PrivateKey, nonce uint64, toAddr gethcmn.Address) ([]byte, error) {
	txData := &gethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       &toAddr,
		Value:    amt,
		Data:     nil,
	}
	tx := gethtypes.NewTx(txData)
	tx, err := ethutils.SignTx(tx, chainID, privKey)
	if err != nil {
		return nil, err
	}

	return ethutils.EncodeTx(tx)
}
