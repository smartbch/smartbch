package ethutils

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

func NewTx(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *types.Transaction {
	return types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       to,
		Value:    amount,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
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
