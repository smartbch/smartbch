package testutils

import (
	"encoding/hex"
	"strings"
)

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
