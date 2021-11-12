package testutils

import "bytes"

func JoinBytes(a ...[]byte) []byte {
	return bytes.Join(a, nil)
}
