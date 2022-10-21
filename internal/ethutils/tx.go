package ethutils

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
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

func EncodeVRS(tx *types.Transaction) [65]byte {
	v, r, s := tx.RawSignatureValues()
	r256, _ := uint256.FromBig(r)
	s256, _ := uint256.FromBig(s)

	bs := [65]byte{}
	bs[0] = byte(v.Uint64())
	copy(bs[1:33], r256.PaddedBytes(32))
	copy(bs[33:65], s256.PaddedBytes(32))
	return bs
}

func DecodeVRS(bs [65]byte) (v, r, s *big.Int) {
	v = big.NewInt(0x4e00 + int64(bs[0]))
	r = big.NewInt(0).SetBytes(bs[1:33])
	s = big.NewInt(0).SetBytes(bs[33:65])
	return
}
