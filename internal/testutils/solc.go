package testutils

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/compiler"
)

func MustCompileSolStr(s string) (code, rtCode string, _abi abi.ABI) {
	var err error
	code, rtCode, _abi, err = CompileSolStr(s)
	if err != nil {
		panic(err)
	}
	return
}

func CompileSolStr(s string) (code, rtCode string, _abi abi.ABI, err error) {
	ret, err := compiler.CompileSolidityString("solc", s)
	if err != nil {
		return "", "", _abi, err
	}

	if len(ret) == 0 {
		return "", "", _abi, errors.New("no compiled contract")
	}
	if len(ret) > 1 {
		return "", "", _abi, errors.New("more than one compiled contracts")
	}
	for _, v := range ret {
		code, rtCode = v.Code, v.RuntimeCode
		abiJSON := ToPrettyJSON(v.Info.AbiDefinition)
		_abi, err = abi.JSON(strings.NewReader(abiJSON))
		return
	}

	panic("unreachable")
}

func MustParseABI(abiJSON string) abi.ABI {
	_abi, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		panic(err)
	}
	return _abi
}
