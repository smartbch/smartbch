package bigutils

import (
	"github.com/holiman/uint256"
)

func U256FromSlice32(arr []byte) *uint256.Int {
	return uint256.NewInt(0).SetBytes32(arr)
}
