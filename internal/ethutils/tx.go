package ethutils

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

func MustEncodeTx(tx *types.Transaction) []byte {
	if data, err := EncodeTx(tx); err == nil {
		return data
	} else {
		panic(err)
	}
}

func EncodeTx(tx *types.Transaction) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := tx.EncodeRLP(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeTx(data []byte) (*types.Transaction, error) {
	tx := &types.Transaction{}
	err := tx.DecodeRLP(rlp.NewStream(bytes.NewReader(data), 0))
	return tx, err
}

func MustSignTx(tx *types.Transaction,
	chainID *big.Int, key *ecdsa.PrivateKey) *types.Transaction {

	if tx, err := SignTx(tx, chainID, key); err == nil {
		return tx
	} else {
		panic(err)
	}
}

func SignTx(tx *types.Transaction,
	chainID *big.Int, key *ecdsa.PrivateKey) (*types.Transaction, error) {

	signer := types.NewEIP155Signer(chainID)
	txHash := signer.Hash(tx)
	sig, err := crypto.Sign(txHash[:], key)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(signer, sig)
}
