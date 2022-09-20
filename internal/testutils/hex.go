package testutils

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

func HexToBytes(s string) []byte {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.ReplaceAll(s, "\n", "")
	return common.Hex2Bytes(s)
}

func HexToU256(s string) *uint256.Int {
	n, err := uint256.FromHex(s)
	if err != nil {
		panic(err)
	}
	return n
}
