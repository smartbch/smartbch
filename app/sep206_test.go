package app

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
)

var (
	sep206Addr        = gethcmn.HexToAddress("0x0000000000000000000000000000000000002711")
	sep206TotalSupply = big.NewInt(0).Mul(big.NewInt(21), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(24), nil)) // 21*10^24
)

var sep206ABI = testutils.MustParseABI(`
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
  "name": "owner",
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

func TestTokenInfo(t *testing.T) {
	_app := CreateTestApp()
	defer DestroyTestApp(_app)

	testCases := []struct {
		getter string
		result interface{}
	}{
		{"name", "BCH"},
		{"symbol", "BCH"},
		{"decimals", uint8(18)},
		{"totalSupply", sep206TotalSupply},
		//{"owner", gethcmn.Address{}},
	}

	for _, testCase := range testCases {
		ret := callViewMethod(t, _app, testCase.getter)
		require.Equal(t, testCase.result, ret)
	}
}

func TestTransferToExistAddr(t *testing.T) {
	privKey1, addr1 := testutils.GenKeyAndAddr()
	privKey2, addr2 := testutils.GenKeyAndAddr()
	_app := CreateTestApp(privKey1, privKey2)
	defer DestroyTestApp(_app)

	b1 := getBalance(_app, addr1)
	b2 := getBalance(_app, addr2)
	require.Equal(t, b1, callViewMethod(t, _app, "balanceOf", addr1))

	amt := big.NewInt(100)
	data1 := sep206ABI.MustPack("transfer", addr2, amt)
	tx1 := gethtypes.NewTransaction(0, sep206Addr, big.NewInt(0), 1000000, big.NewInt(0), data1)
	tx1 = ethutils.MustSignTx(tx1, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(privKey1))
	testutils.ExecTxInBlock(_app, 1, tx1)
	require.Equal(t, b1.Sub(b1, amt), callViewMethod(t, _app, "balanceOf", addr1))
	require.Equal(t, b2.Add(b2, amt), callViewMethod(t, _app, "balanceOf", addr2))
}

func TestTransferToNonExistAddr(t *testing.T) {
	privKey1, addr1 := testutils.GenKeyAndAddr()
	_, addr2 := testutils.GenKeyAndAddr()
	_app := CreateTestApp(privKey1)
	defer DestroyTestApp(_app)

	b1 := getBalance(_app, addr1)
	require.Equal(t, b1, callViewMethod(t, _app, "balanceOf", addr1))
	require.Equal(t, uint64(0), callViewMethod(t, _app, "balanceOf", addr2).(*big.Int).Uint64())

	amt := big.NewInt(100)
	data1 := sep206ABI.MustPack("transfer", addr2, amt)
	tx1 := gethtypes.NewTransaction(0, sep206Addr, big.NewInt(0), 1000000, big.NewInt(0), data1)
	tx1 = ethutils.MustSignTx(tx1, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(privKey1))
	testutils.ExecTxInBlock(_app, 1, tx1)
	require.Equal(t, b1.Sub(b1, amt), callViewMethod(t, _app, "balanceOf", addr1))
	require.Equal(t, amt, callViewMethod(t, _app, "balanceOf", addr2))
}

func TestTransferFrom(t *testing.T) {
	privKey1, addr1 := testutils.GenKeyAndAddr()
	privKey2, addr2 := testutils.GenKeyAndAddr()
	privKey3, _ := testutils.GenKeyAndAddr()
	_app := CreateTestApp(privKey1, privKey2, privKey3)
	defer DestroyTestApp(_app)

	a1 := callViewMethod(t, _app, "allowance", addr1, addr2)
	require.Equal(t, uint64(0), a1.(*big.Int).Uint64())
}

func callViewMethod(t *testing.T, _app *App, selector string, args ...interface{}) interface{} {
	data := sep206ABI.MustPack(selector, args...)
	tx := gethtypes.NewTransaction(0, sep206Addr, big.NewInt(0), 10000000, big.NewInt(1), data)
	statusCode, statusStr, output := call(_app, gethcmn.Address{}, tx)
	require.Equal(t, 0, statusCode, selector)
	require.Equal(t, "success", statusStr, selector)
	result := sep206ABI.MustUnpack(selector, output)
	require.Len(t, result, 1, selector)
	return result[0]
}
