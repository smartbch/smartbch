package testutils

import (
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

func HexToHash32(s string) common.Hash {
	s = strings.TrimPrefix(s, "0x")
	bytes, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	hash := common.Hash{}
	copy(hash[:], bytes)

	return hash
}

func HexToBytes(s string) []byte {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.ReplaceAll(s, "\n", "")

	bytes, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return bytes
}

func HexToU256(s string) *uint256.Int {
	n, err := uint256.FromHex(s)
	if err != nil {
		panic(err)
	}
	return n
}
