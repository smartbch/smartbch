package app

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
)

var _sep101ABI = testutils.MustParseABI(`
[
{
  "inputs": [
	{
	  "internalType": "uint256",
	  "name": "key",
	  "type": "uint256"
	},
	{
	  "internalType": "bytes",
	  "name": "value",
	  "type": "bytes"
	}
  ],
  "name": "set",
  "outputs": [],
  "stateMutability": "nonpayable",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "uint256",
	  "name": "key",
	  "type": "uint256"
	}
  ],
  "name": "get",
  "outputs": [
	{
	  "internalType": "bytes",
	  "name": "",
	  "type": "bytes"
	}
  ],
  "stateMutability": "view",
  "type": "function"
}
]
`)

func TestSEP101(t *testing.T) {
	privKey, addr := testutils.GenKeyAndAddr()
	_app := CreateTestApp(privKey)
	defer DestroyTestApp(_app)

	contractAddr := gethcmn.HexToAddress("0x0000000000000000000000000000000000010002")
	key := big.NewInt(0x789)
	val := bytes.Repeat([]byte{0x12, 0x34}, 500)

	// call set()
	data, err := _sep101ABI.Pack("set", key, val)
	require.NoError(t, err)
	tx1 := gethtypes.NewTransaction(0, contractAddr,
		big.NewInt(0), 10000000, big.NewInt(1), data)
	tx1 = ethutils.MustSignTx(tx1, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(privKey))
	testutils.ExecTxInBlock(_app, 1, tx1)

	blk1 := getBlock(_app, 1)
	require.Equal(t, int64(1), blk1.Number)
	require.Len(t, blk1.Transactions, 1)
	txInBlk1 := getTx(_app, blk1.Transactions[0])
	require.Equal(t, gethtypes.ReceiptStatusSuccessful, txInBlk1.Status)
	require.Equal(t, tx1.Hash(), gethcmn.Hash(txInBlk1.Hash))

	// call get()
	data, err = _sep101ABI.Pack("get", key)
	require.NoError(t, err)
	tx2 := gethtypes.NewTransaction(0, contractAddr,
		big.NewInt(0), 10000000, big.NewInt(1), data)
	statusCode, statusStr, output := call(_app, addr, tx2)
	require.Equal(t, 0, statusCode)
	require.Equal(t, "success", statusStr)
	require.Equal(t, val, output)
}
