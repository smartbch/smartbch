package testutils

import "math/big"

func BigIntAddI64(x *big.Int, y int64) *big.Int {
	return big.NewInt(0).Add(x, big.NewInt(y))
}
