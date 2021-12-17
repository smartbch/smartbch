package bigutils

import (
	"math/big"

	"github.com/holiman/uint256"
)

func ConvertBig(v *big.Int) *uint256.Int {
	u, overflow := uint256.FromBig(v)
	if overflow {
		panic("big.Int overflow")
	}
	return u
}

func BigIntFromSlice32(arr []byte) *big.Int {
	bz := arr
	if (arr[0] & 128) != 0 { // prevent overflow to negative
		bz = append([]byte{0}, arr...)
	}
	res := &big.Int{}
	res.SetBytes(bz)
	return res
}

func BigIntToSlice32(v *big.Int) []byte {
	var arr [33]byte
	v.FillBytes(arr[:])
	return arr[1:]
}

func U256FromSlice32(arr []byte) *uint256.Int {
	return uint256.NewInt(0).SetBytes32(arr)
}

func U256ToSlice32(v *uint256.Int) []byte {
	var arr [32]byte
	v.WriteToArray32(&arr)
	return arr[:]
}
