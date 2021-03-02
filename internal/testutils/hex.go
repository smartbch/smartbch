package testutils

import (
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

func HexToHash32(s string) common.Hash {
	if strings.HasPrefix(s, "0x") {
		s = s[2:]
	}

	bytes, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	hash := common.Hash{}
	copy(hash[:], bytes)

	return hash
}

func HexToBytes(s string) []byte {
	if strings.HasPrefix(s, "0x") {
		s = s[2:]
	}
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", "")

	bytes, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return bytes
}
