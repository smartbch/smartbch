package testutils

import (
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/common"
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
