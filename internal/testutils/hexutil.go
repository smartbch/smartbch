package testutils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func ToHexutilU64(n uint64) *hexutil.Uint64 {
	return (*hexutil.Uint64)(&n)
}

func ToHexutilBig(n int64) *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(n))
}

func ToHexutilBytes(s []byte) *hexutil.Bytes {
	return (*hexutil.Bytes)(&s)
}
