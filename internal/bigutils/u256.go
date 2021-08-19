package bigutils

import (
	"math/big"
	"strings"

	"github.com/holiman/uint256"
)

func NewU256(u64 uint64) *uint256.Int {
	u256 := uint256.NewInt(0)
	u256.SetUint64(u64)
	return u256
}

func ParseU256(s string) (*uint256.Int, bool) {
	i := big.NewInt(0)
	ok := false
	if strings.HasPrefix(s, "0x") {
		i, ok = i.SetString(s[2:], 16)
	} else {
		i, ok = i.SetString(s, 10)
	}
	if ok {
		u, overflow := uint256.FromBig(i)
		return u, !overflow
	}
	return nil, false
}
