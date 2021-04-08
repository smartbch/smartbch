package testutils

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type ABI4Test struct {
	_abi abi.ABI
}

func (a ABI4Test) Pack(name string, args ...interface{}) ([]byte, error) {
	return a._abi.Pack(name, args...)
}
func (a ABI4Test) MustPack(name string, args ...interface{}) []byte {
	bytes, err := a._abi.Pack(name, args...)
	if err != nil {
		panic(err)
	}
	return bytes
}

func (a ABI4Test) MustUnpack(name string, data []byte) []interface{} {
	ret, err := a._abi.Unpack(name, data)
	if err != nil {
		panic(err)
	}
	return ret
}

func MustParseABI(abiJSON string) ABI4Test {
	_abi, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		panic(err)
	}
	return ABI4Test{_abi}
}
