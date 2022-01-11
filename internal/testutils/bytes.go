package testutils

import (
	"bytes"

	"github.com/holiman/uint256"
)

func JoinBytes(a ...[]byte) []byte {
	return bytes.Join(a, nil)
}

func UintToBytes32(n uint64) []byte {
	return uint256.NewInt(n).PaddedBytes(32)
}
