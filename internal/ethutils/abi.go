package ethutils

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type ABIWrapper struct {
	_abi abi.ABI
}

func (a ABIWrapper) GetABI() abi.ABI {
	return a._abi
}

func (a ABIWrapper) Pack(name string, args ...interface{}) ([]byte, error) {
	return a._abi.Pack(name, args...)
}
func (a ABIWrapper) MustPack(name string, args ...interface{}) []byte {
	bytes, err := a._abi.Pack(name, args...)
	if err != nil {
		panic(err)
	}
	return bytes
}

func (a ABIWrapper) MustUnpack(name string, data []byte) []interface{} {
	ret, err := a._abi.Unpack(name, data)
	if err != nil {
		panic(err)
	}
	return ret
}

func MustParseABI(abiJSON string) ABIWrapper {
	_abi, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		panic(err)
	}
	return ABIWrapper{_abi}
}
