package app

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
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

func TestEventSigs(t *testing.T) {
	require.Equal(t, "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
		sep206ABI.GetABI().Events["Transfer"].ID.Hex())
	require.Equal(t, "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925",
		sep206ABI.GetABI().Events["Approval"].ID.Hex())
}

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

func TestTransferToExistingAddr(t *testing.T) {
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
	tx1 = testutils.MustSignTx(tx1, _app.chainId.ToBig(), privKey1)
	testutils.ExecTxInBlock(_app, 1, tx1)
	require.Equal(t, b1.Sub(b1, amt), callViewMethod(t, _app, "balanceOf", addr1))
	require.Equal(t, b2.Add(b2, amt), callViewMethod(t, _app, "balanceOf", addr2))
}

func TestTransferToNonExistingAddr(t *testing.T) {
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
	tx1 = testutils.MustSignTx(tx1, _app.chainId.ToBig(), privKey1)
	testutils.ExecTxInBlock(_app, 1, tx1)
	require.Equal(t, b1.Sub(b1, amt), callViewMethod(t, _app, "balanceOf", addr1))
	require.Equal(t, amt, callViewMethod(t, _app, "balanceOf", addr2))
}

func TestAllowance(t *testing.T) {
	ownerKey, ownerAddr := testutils.GenKeyAndAddr()
	spenderKey, spenderAddr := testutils.GenKeyAndAddr()
	_app := CreateTestApp(ownerKey, spenderKey)
	defer DestroyTestApp(_app)

	a0 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(0), a0.(*big.Int).Uint64())

	data1 := sep206ABI.MustPack("approve", spenderAddr, big.NewInt(12345))
	tx1 := gethtypes.NewTransaction(0, sep206Addr, big.NewInt(0), 1000000, big.NewInt(0), data1)
	tx1 = testutils.MustSignTx(tx1, _app.chainId.ToBig(), ownerKey)
	testutils.ExecTxInBlock(_app, 1, tx1)
	checkTx(t, _app, 1, tx1.Hash())
	a1 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(12345), a1.(*big.Int).Uint64())

	data2 := sep206ABI.MustPack("increaseAllowance", spenderAddr, big.NewInt(123))
	tx2 := gethtypes.NewTransaction(1, sep206Addr, big.NewInt(0), 1000000, big.NewInt(0), data2)
	tx2 = testutils.MustSignTx(tx2, _app.chainId.ToBig(), ownerKey)
	testutils.ExecTxInBlock(_app, 3, tx2)
	checkTx(t, _app, 3, tx2.Hash())
	a2 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(12468), a2.(*big.Int).Uint64())

	data3 := sep206ABI.MustPack("decreaseAllowance", spenderAddr, big.NewInt(456))
	tx3 := gethtypes.NewTransaction(2, sep206Addr, big.NewInt(0), 1000000, big.NewInt(0), data3)
	tx3 = testutils.MustSignTx(tx3, _app.chainId.ToBig(), ownerKey)
	testutils.ExecTxInBlock(_app, 5, tx3)
	checkTx(t, _app, 5, tx3.Hash())
	a3 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(12012), a3.(*big.Int).Uint64())
}

func TestTransferFrom(t *testing.T) {
	ownerKey, ownerAddr := testutils.GenKeyAndAddr()
	spenderKey, spenderAddr := testutils.GenKeyAndAddr()
	_, receiptAddr := testutils.GenKeyAndAddr()
	_app := CreateTestApp(ownerKey, spenderKey)
	defer DestroyTestApp(_app)

	data1 := sep206ABI.MustPack("approve", spenderAddr, big.NewInt(12345))
	tx1 := gethtypes.NewTransaction(0, sep206Addr, big.NewInt(0), 1000000, big.NewInt(0), data1)
	tx1 = testutils.MustSignTx(tx1, _app.chainId.ToBig(), ownerKey)
	testutils.ExecTxInBlock(_app, 1, tx1)
	checkTx(t, _app, 1, tx1.Hash())
	a1 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(12345), a1.(*big.Int).Uint64())

	data2 := sep206ABI.MustPack("transferFrom", ownerAddr, receiptAddr, big.NewInt(345))
	tx2 := gethtypes.NewTransaction(0, sep206Addr, big.NewInt(0), 1000000, big.NewInt(0), data2)
	tx2 = testutils.MustSignTx(tx2, _app.chainId.ToBig(), spenderKey)
	testutils.ExecTxInBlock(_app, 3, tx2)
	checkTx(t, _app, 3, tx2.Hash())
	a2 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(12000), a2.(*big.Int).Uint64())
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

func checkTx(t *testing.T, _app *App, h int64, txHash gethcmn.Hash) {
	blk := getBlock(_app, uint64(h))
	require.Equal(t, h, blk.Number)
	require.Len(t, blk.Transactions, 1)
	txInBlk := getTx(_app, blk.Transactions[0])
	require.Equal(t, gethtypes.ReceiptStatusSuccessful, txInBlk.Status)
	require.Equal(t, "success", txInBlk.StatusStr)
	require.Equal(t, txHash, gethcmn.Hash(txInBlk.Hash))
}
