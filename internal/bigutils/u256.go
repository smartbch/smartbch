package bigutils

import (
	"github.com/holiman/uint256"
)

func NewU256(u64 uint64) *uint256.Int {
	u256 := uint256.NewInt()
	u256.SetUint64(u64)
	return u256
}
