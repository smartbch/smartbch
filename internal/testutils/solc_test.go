package testutils

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func _TestCompileSolStr(t *testing.T) {
	code, rtCode, abi := MustCompileSolStr(`
// SPDX-License-Identifier: MIT
pragma solidity >=0.6.0;

contract Counter {

  int public counter;

  function update(int n) public {
    counter += n;
  }

}
`)

	require.Equal(t, code, "0x608060405234801561001057600080fd5b5060b28061001f6000396000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c806361bc221a1460375780636299a6ef14604f575b600080fd5b603d606b565b60408051918252519081900360200190f35b606960048036036020811015606357600080fd5b50356071565b005b60005481565b60008054909101905556fea26469706673582212205df2a10ba72894ded3e0a7ea8c57a79906cca125c3aafe3c979fbd57e662c01d64736f6c634300060c0033")
	require.Equal(t, rtCode, "0x6080604052348015600f57600080fd5b506004361060325760003560e01c806361bc221a1460375780636299a6ef14604f575b600080fd5b603d606b565b60408051918252519081900360200190f35b606960048036036020811015606357600080fd5b50356071565b005b60005481565b60008054909101905556fea26469706673582212205df2a10ba72894ded3e0a7ea8c57a79906cca125c3aafe3c979fbd57e662c01d64736f6c634300060c0033")

	/*
		[
		  {
			"inputs": [],
			"name": "counter",
			"outputs": [
			  {
				"internalType": "int256",
				"name": "",
				"type": "int256"
			  }
			],
			"stateMutability": "view",
			"type": "function"
		  },
		  {
			"inputs": [
			  {
				"internalType": "int256",
				"name": "n",
				"type": "int256"
			  }
			],
			"name": "update",
			"outputs": [],
			"stateMutability": "nonpayable",
			"type": "function"
		  }
		]
	*/
	require.Equal(t, 2, len(abi.Methods))
	m, err := abi.MethodById(HexToBytes("61bc221a"))
	require.NoError(t, err)
	require.Equal(t, "counter", m.Name)

	args, err := abi.Pack("counter")
	require.NoError(t, err)
	require.Equal(t, "61bc221a", hex.EncodeToString(args))

	args, err = abi.Pack("update", big.NewInt(3))
	require.NoError(t, err)
	require.Equal(t, "6299a6ef0000000000000000000000000000000000000000000000000000000000000003",
		hex.EncodeToString(args))
}
