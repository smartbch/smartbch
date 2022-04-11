package ethutils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

func DecodeVRS(bs [65]byte) (v, r, s *big.Int) {
	v = big.NewInt(int64(bs[0]))
	r = big.NewInt(0).SetBytes(bs[1:33])
	s = big.NewInt(0).SetBytes(bs[33:65])
	return
}
