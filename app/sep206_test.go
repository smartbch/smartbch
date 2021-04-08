package app

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
)

var _sep206ABI = testutils.MustParseABI(`
[
{
  "anonymous": false,
  "inputs": [
	{
	  "indexed": true,
	  "internalType": "address",
	  "name": "_owner",
	  "type": "address"
	},
	{
	  "indexed": true,
	  "internalType": "address",
	  "name": "_spender",
	  "type": "address"
	},
	{
	  "indexed": false,
	  "internalType": "uint256",
	  "name": "_value",
	  "type": "uint256"
	}
  ],
  "name": "Approval",
  "type": "event"
},
{
  "anonymous": false,
  "inputs": [
	{
	  "indexed": true,
	  "internalType": "address",
	  "name": "_from",
	  "type": "address"
	},
	{
	  "indexed": true,
	  "internalType": "address",
	  "name": "_to",
	  "type": "address"
	},
	{
	  "indexed": false,
	  "internalType": "uint256",
	  "name": "_value",
	  "type": "uint256"
	}
  ],
  "name": "Transfer",
  "type": "event"
},
{
  "inputs": [],
  "name": "name",
  "outputs": [
	{
	  "internalType": "string",
	  "name": "",
	  "type": "string"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [],
  "name": "symbol",
  "outputs": [
	{
	  "internalType": "string",
	  "name": "",
	  "type": "string"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [],
  "name": "decimals",
  "outputs": [
	{
	  "internalType": "uint8",
	  "name": "",
	  "type": "uint8"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [],
  "name": "totalSupply",
  "outputs": [
	{
	  "internalType": "uint256",
	  "name": "",
	  "type": "uint256"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [],
  "name": "getOwner",
  "outputs": [
	{
	  "internalType": "address",
	  "name": "",
	  "type": "address"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_owner",
	  "type": "address"
	}
  ],
  "name": "balanceOf",
  "outputs": [
	{
	  "internalType": "uint256",
	  "name": "balance",
	  "type": "uint256"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_to",
	  "type": "address"
	},
	{
	  "internalType": "uint256",
	  "name": "_value",
	  "type": "uint256"
	}
  ],
  "name": "transfer",
  "outputs": [
	{
	  "internalType": "bool",
	  "name": "success",
	  "type": "bool"
	}
  ],
  "stateMutability": "nonpayable",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_from",
	  "type": "address"
	},
	{
	  "internalType": "address",
	  "name": "_to",
	  "type": "address"
	},
	{
	  "internalType": "uint256",
	  "name": "_value",
	  "type": "uint256"
	}
  ],
  "name": "transferFrom",
  "outputs": [
	{
	  "internalType": "bool",
	  "name": "success",
	  "type": "bool"
	}
  ],
  "stateMutability": "nonpayable",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_spender",
	  "type": "address"
	},
	{
	  "internalType": "uint256",
	  "name": "_value",
	  "type": "uint256"
	}
  ],
  "name": "approve",
  "outputs": [
	{
	  "internalType": "bool",
	  "name": "success",
	  "type": "bool"
	}
  ],
  "stateMutability": "nonpayable",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_owner",
	  "type": "address"
	},
	{
	  "internalType": "address",
	  "name": "_spender",
	  "type": "address"
	}
  ],
  "name": "allowance",
  "outputs": [
	{
	  "internalType": "uint256",
	  "name": "remaining",
	  "type": "uint256"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_spender",
	  "type": "address"
	},
	{
	  "internalType": "uint256",
	  "name": "_delta",
	  "type": "uint256"
	}
  ],
  "name": "increaseAllowance",
  "outputs": [
	{
	  "internalType": "bool",
	  "name": "success",
	  "type": "bool"
	}
  ],
  "stateMutability": "nonpayable",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_spender",
	  "type": "address"
	},
	{
	  "internalType": "uint256",
	  "name": "_delta",
	  "type": "uint256"
	}
  ],
  "name": "decreaseAllowance",
  "outputs": [
	{
	  "internalType": "bool",
	  "name": "success",
	  "type": "bool"
	}
  ],
  "stateMutability": "nonpayable",
  "type": "function"
}
]
`)

func TestFakeERC20(t *testing.T) {
	privKey, addr := testutils.GenKeyAndAddr()
	_app := CreateTestApp(privKey)
	defer DestroyTestApp(_app)

	// see testdata/seps/contracts/FakeERC20.sol
	creationBytecode := testutils.HexToBytes(`
608060405234801561001057600080fd5b5061017c806100206000396000f3fe
608060405234801561001057600080fd5b506004361061002b5760003560e01c
806306fdde0314610030575b600080fd5b61003861004e565b60405161004591
906100c4565b60405180910390f35b6060604051806040016040528060038152
6020017f42434800000000000000000000000000000000000000000000000000
00000000815250905090565b6000610096826100e6565b6100a081856100f156
5b93506100b0818560208601610102565b6100b981610135565b840191505092
915050565b600060208201905081810360008301526100de818461008b565b90
5092915050565b600081519050919050565b6000828252602082019050929150
50565b60005b8381101561012057808201518184015260208101905061010556
5b8381111561012f576000848401525b50505050565b6000601f19601f830116
905091905056fea2646970667358221220a5eecc856d49e4ae8faff58b8fe5ae
09c6b3e07d3b8f8c85fc2e328e8253a39664736f6c63430008000033
`)

	// deploy contract
	tx1 := gethtypes.NewContractCreation(0, big.NewInt(0), 1000000, big.NewInt(1), creationBytecode)
	tx1 = ethutils.MustSignTx(tx1, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(privKey))

	testutils.ExecTxInBlock(_app, 1, tx1)
	contractAddr := gethcrypto.CreateAddress(addr, tx1.Nonce())
	code := getCode(_app, contractAddr)
	require.True(t, len(code) > 0)

	// call name()
	data := _sep206ABI.MustPack("name")
	tx2 := gethtypes.NewTransaction(0, contractAddr, big.NewInt(0), 10000000, big.NewInt(1), data)
	statusCode, statusStr, output := call(_app, addr, tx2)
	require.Equal(t, 0, statusCode)
	require.Equal(t, "success", statusStr)
	require.Equal(t, []interface{}{"BCH"}, _sep206ABI.MustUnpack("name", output))
}

func TestTokenInfo(t *testing.T) {
	privKey, addr := testutils.GenKeyAndAddr()
	_app := CreateTestApp(privKey)
	defer DestroyTestApp(_app)

	contractAddr := gethcmn.HexToAddress("0x0000000000000000000000000000000000002712")

	testCases := []struct {
		getter string
		result interface{}
	}{
		{"name", "BCH"},
		{"symbol", "BCH"},
		//{"decimals", 18},
		//{"totalSupply", big.NewInt(0).Mul(big.NewInt(21), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(24), nil))},
	}

	for _, testCase := range testCases {
		data := _sep206ABI.MustPack(testCase.getter)
		tx := gethtypes.NewTransaction(0, contractAddr, big.NewInt(0), 10000000, big.NewInt(1), data)
		statusCode, statusStr, output := call(_app, addr, tx)
		require.Equal(t, 0, statusCode, testCase.getter)
		require.Equal(t, "success", statusStr)
		require.Equal(t, []interface{}{testCase.result}, _sep206ABI.MustUnpack(testCase.getter, output))
	}
}
