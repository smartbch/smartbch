package api

import (
	"bytes"
	"encoding/hex"
	"math"
	"math/big"
	"regexp"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/rpc/internal/ethapi"
	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
	"github.com/smartbch/smartbch/staking"
)

// testdata/sol/contracts/basic/Counter.sol
var counterContractCreationBytecode = testutils.HexToBytes(`
608060405234801561001057600080fd5b5060b28061001f6000396000f3fe60
80604052348015600f57600080fd5b506004361060325760003560e01c806361
bc221a1460375780636299a6ef14604f575b600080fd5b603d606b565b604080
51918252519081900360200190f35b6069600480360360208110156063576000
80fd5b50356071565b005b60005481565b60008054909101905556fea2646970
6673582212205df2a10ba72894ded3e0a7ea8c57a79906cca125c3aafe3c979f
bd57e662c01d64736f6c634300060c0033
`)

var counterContractABI = ethutils.MustParseABI(`
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
`)

// testdata/sol/contracts/basic/BlockNum.sol
var blockNumContractCreationBytecode = testutils.HexToBytes(`
608060405234801561001057600080fd5b506102bd806100206000396000f3fe
608060405234801561001057600080fd5b506004361061004c5760003560e01c
806319efb11d14610051578063b51c4f961461006f578063ee82ac5e1461009f
578063f8b2cb4f146100cf575b600080fd5b6100596100ff565b604051610066
91906101f8565b60405180910390f35b61008960048036038101906100849190
61016d565b610107565b60405161009691906101f8565b60405180910390f35b
6100b960048036038101906100b49190610196565b610117565b6040516100c6
91906101dd565b60405180910390f35b6100e960048036038101906100e49190
61016d565b610122565b6040516100f691906101f8565b60405180910390f35b
600043905090565b600080823b905080915050919050565b6000814090509190
50565b60008173ffffffffffffffffffffffffffffffffffffffff1631905091
9050565b60008135905061015281610259565b92915050565b60008135905061
016781610270565b92915050565b60006020828403121561017f57600080fd5b
600061018d84828501610143565b91505092915050565b600060208284031215
6101a857600080fd5b60006101b684828501610158565b91505092915050565b
6101c881610225565b82525050565b6101d78161024f565b82525050565b6000
6020820190506101f260008301846101bf565b92915050565b60006020820190
5061020d60008301846101ce565b92915050565b600061021e8261022f565b90
50919050565b6000819050919050565b600073ffffffffffffffffffffffffff
ffffffffffffff82169050919050565b6000819050919050565b610262816102
13565b811461026d57600080fd5b50565b6102798161024f565b811461028457
600080fd5b5056fea26469706673582212208e243a5205c34129832b8e93ec6c
2b9425ab313a6295e3e3b3d69e08f684658964736f6c63430008000033
`)

var blockNumContractABI = ethutils.MustParseABI(`
[
  {
    "inputs": [],
    "name": "getHeight",
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
    "inputs": [
      {
        "internalType": "address",
        "name": "addr",
        "type": "address"
      }
    ],
    "name": "getBalance",
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
    "inputs": [
      {
        "internalType": "address",
        "name": "addr",
        "type": "address"
      }
    ],
    "name": "getCodeSize",
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
    "inputs": [
      {
        "internalType": "uint256",
        "name": "blockNumber",
        "type": "uint256"
      }
    ],
    "name": "getBlockHash",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]
`)

func TestAccounts(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()

	_app := testutils.CreateTestApp(key1, key2)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app, key1, key2)

	addrs, err := _api.Accounts()
	require.NoError(t, err)
	require.Len(t, addrs, 2)
	require.Contains(t, addrs, addr1)
	require.Contains(t, addrs, addr2)
}

func TestChainId(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	id := _api.ChainId()
	require.Equal(t, "0x1", id.String())
}

func TestGasPrice(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	ctx := _app.GetRunTxContext()
	staking.SaveMinGasPrice(ctx, 10_000_000_000, false)
	ctx.Close(true)
	_app.ExecTxsInBlock()

	require.Equal(t, "0x2540be400", _api.GasPrice().String())
}

func TestBlockNum(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	ctx := _app.GetRunTxContext()
	ctx.Db.AddBlock(&modbtypes.Block{Height: 0x100}, -1, nil)
	ctx.Db.AddBlock(nil, -1, nil) //To Flush
	ctx.Close(true)

	num, err := _api.BlockNumber()
	require.NoError(t, err)
	require.Equal(t, "0x100", num.String())
}

func TestGetBalance(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	b, err := _api.GetBalance(addr, latestBlockNumber())
	require.NoError(t, err)
	require.Equal(t, "0x989680", b.String())

	b2, err := _api.GetBalance(addr2, latestBlockNumber())
	require.NoError(t, err)
	require.Equal(t, "0x0", b2.String())
}

func TestGetTxCount(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_, addr3 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	nonce, err := _api.GetTransactionCount(addr1, latestBlockNumber())
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(0), *nonce)
	nonce, err = _api.GetTransactionCount(addr2, latestBlockNumber())
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(0), *nonce)

	for i := 0; i < 3; i++ {
		tx, _ := _app.MakeAndExecTxInBlock(key1, addr2, 100, nil)
		_app.EnsureTxSuccess(tx.Hash())

		nonce, err = _api.GetTransactionCount(addr1, latestBlockNumber())
		require.NoError(t, err)
		require.Equal(t, hexutil.Uint64(i+1), *nonce)
	}

	nonce, err = _api.GetTransactionCount(addr2, latestBlockNumber())
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(0), *nonce)

	nonce, err = _api.GetTransactionCount(addr3, latestBlockNumber())
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(0), *nonce)
}

func TestGetCode(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	ctx := _app.GetRunTxContext()
	code := bytes.Repeat([]byte{0xff}, 32)
	code = append(code, 0x0 /*version byte*/, 0x12, 0x34)
	ctx.Rbt.Set(types.GetBytecodeKey(addr), types.NewBytecodeInfo(code).Bytes())
	ctx.Close(true)
	_app.CloseTxEngineContext()
	_app.CloseTrunk()

	c, err := _api.GetCode(addr, latestBlockNumber())
	require.NoError(t, err)
	require.Equal(t, "0x1234", c.String())

	c, err = _api.GetCode(addr2, latestBlockNumber())
	require.NoError(t, err)
	require.Equal(t, "0x", c.String())
}

func TestGetStorageAt(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	ctx := _app.GetRunTxContext()
	seq := ctx.GetAccount(addr).Sequence()
	sKey := bytes.Repeat([]byte{0xab, 0xcd}, 16)
	sVal := bytes.Repeat([]byte{0x12, 0x34}, 16)
	ctx.SetStorageAt(seq, string(sKey), sVal)
	ctx.Close(true)
	_app.CloseTxEngineContext()
	_app.CloseTrunk()

	val0 := testutils.UintToBytes32(0)
	require.Equal(t, sVal, getStorageAt(_api, addr, "0x"+hex.EncodeToString(sKey), -1))
	require.Equal(t, val0, getStorageAt(_api, addr, "0x7890", -1))
	require.Equal(t, val0, getStorageAt(_api, addr2, "0x7890", -1))
}

func TestQueryBlockByNum(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	_app.StoreBlocks(
		//newMdbBlock(gethcmn.Hash{0xb0}, 0, []gethcmn.Hash{{0xc1}}),
		newMdbBlock(gethcmn.Hash{0xb1}, 1, []gethcmn.Hash{{0xc2}, {0xc3}}),
		newMdbBlock(gethcmn.Hash{0xb2}, 2, []gethcmn.Hash{{0xc4}, {0xc5}, {0xc6}, {0xc7}}),
	)

	h, err := _api.BlockNumber()
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(2), h)

	testCases := []struct {
		num     gethrpc.BlockNumber
		txIdx   hexutil.Uint
		height  hexutil.Uint64
		hash    gethcmn.Hash
		txCount hexutil.Uint
		txHash  gethcmn.Hash
	}{
		{0, 0, 0, gethcmn.Hash{0x00}, 0, gethcmn.Hash{0xc1}},
		{1, 0, 1, gethcmn.Hash{0xb1}, 2, gethcmn.Hash{0xc2}},
		{2, 0, 2, gethcmn.Hash{0xb2}, 4, gethcmn.Hash{0xc4}},
		{-1, 1, 2, gethcmn.Hash{0xb2}, 4, gethcmn.Hash{0xc5}},
	}

	for _, testCase := range testCases {
		ret, err := _api.GetBlockByNumber(testCase.num, false)
		require.NoError(t, err)
		require.Equal(t, testCase.height, ret["number"])
		require.Equal(t, hexutil.Bytes(testCase.hash[:]), ret["hash"])

		txCount := _api.GetBlockTransactionCountByNumber(testCase.num)
		require.Equal(t, testCase.txCount, *txCount)

		if testCase.txCount > 0 {
			tx, err := _api.GetTransactionByBlockNumberAndIndex(testCase.num, testCase.txIdx)
			require.NoError(t, err)
			require.Equal(t, testCase.txHash, tx.Hash)
		}
	}
}

func TestGetBlockByNumAndHash(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	hash := gethcmn.Hash{0x12, 0x34}
	block := newMdbBlock(hash, 123, []gethcmn.Hash{
		{0x56}, {0x78}, {0x90},
	})
	_app.StoreBlocks(block)

	getBlockByNumResult, err := _api.GetBlockByNumber(123, false)
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(123), getBlockByNumResult["number"])
	require.Equal(t, hexutil.Bytes(hash[:]), getBlockByNumResult["hash"])
	require.Equal(t, hexutil.Uint64(param.BlockMaxGas), getBlockByNumResult["gasLimit"])
	require.Equal(t, hexutil.Uint64(0), getBlockByNumResult["gasUsed"])
	require.Equal(t, hexutil.Bytes(nil), getBlockByNumResult["extraData"])
	require.Equal(t, "0x0000000000000000", getBlockByNumResult["nonce"].(hexutil.Bytes).String())
	require.Equal(t, []gethcmn.Hash{{0x56}, {0x78}, {0x90}}, getBlockByNumResult["transactions"])

	getBlockByHashResult, err := _api.GetBlockByHash(hash, false)
	require.NoError(t, err)
	require.Equal(t, getBlockByNumResult, getBlockByHashResult)

	getBlockByNumResultFull, err := _api.GetBlockByNumber(123, true)
	require.NoError(t, err)
	require.Len(t, getBlockByNumResultFull["transactions"].([]*rpctypes.Transaction), 3)

	getBlockByHashResultFull, err := _api.GetBlockByHash(hash, true)
	require.NoError(t, err)
	require.Equal(t, getBlockByNumResultFull, getBlockByHashResultFull)
}

func TestGetBlockByNumAndHash_notFound(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	blk, err := _api.GetBlockByNumber(99, false)
	require.NoError(t, err)
	require.Nil(t, blk)

	blk, err = _api.GetBlockByNumber(99, true)
	require.NoError(t, err)
	require.Nil(t, blk)

	blk, err = _api.GetBlockByHash(gethcmn.Hash{0x12, 0x34}, false)
	require.NoError(t, err)
	require.Nil(t, blk)

	blk, err = _api.GetBlockByHash(gethcmn.Hash{0x12, 0x34}, true)
	require.NoError(t, err)
	require.Nil(t, blk)
}

func TestGetBlockTxCountByHash(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	hash := gethcmn.Hash{0x12, 0x34}
	block := newMdbBlock(hash, 123, []gethcmn.Hash{
		{0x56}, {0x78}, {0x90},
	})
	_app.StoreBlocks(block)

	cnt := _api.GetBlockTransactionCountByHash(hash)
	require.Equal(t, hexutil.Uint(3), *cnt)

	cnt = _api.GetBlockTransactionCountByHash(gethcmn.Hash{0x56, 0x78})
	require.Nil(t, cnt)
}

func TestGetBlockTxCountByNum(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	hash := gethcmn.Hash{0x12, 0x34}
	block := newMdbBlock(hash, 123, []gethcmn.Hash{
		{0x56}, {0x78}, {0x90}, {0xAB},
	})
	_app.StoreBlocks(block)

	cnt := _api.GetBlockTransactionCountByNumber(123)
	require.Equal(t, hexutil.Uint(4), *cnt)

	cnt = _api.GetBlockTransactionCountByNumber(999)
	require.Nil(t, cnt)
}

func TestGetTxByBlockHashAndIdx(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	blkHash := gethcmn.Hash{0x12, 0x34}
	block := newMdbBlock(blkHash, 123, []gethcmn.Hash{
		{0x56}, {0x78}, {0x90}, {0xAB},
	})
	_app.StoreBlocks(block)

	tx, err := _api.GetTransactionByBlockHashAndIndex(blkHash, 2)
	require.NoError(t, err)
	require.Equal(t, gethcmn.Hash{0x90}, tx.Hash)

	tx, err = _api.GetTransactionByBlockHashAndIndex(blkHash, 99)
	require.NoError(t, err)
	require.Nil(t, tx)

	tx, err = _api.GetTransactionByBlockHashAndIndex(gethcmn.Hash{0x56, 0x78}, 99)
	require.NoError(t, err)
	require.Nil(t, tx)
}

func TestGetTxByBlockNumAndIdx(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	blkHash := gethcmn.Hash{0x12, 0x34}
	block := newMdbBlock(blkHash, 123, []gethcmn.Hash{
		{0x56}, {0x78}, {0x90}, {0xAB},
	})
	_app.StoreBlocks(block)

	tx, err := _api.GetTransactionByBlockNumberAndIndex(123, 1)
	require.NoError(t, err)
	require.Equal(t, gethcmn.Hash{0x78}, tx.Hash)

	tx, err = _api.GetTransactionByBlockNumberAndIndex(123, 99)
	require.NoError(t, err)
	require.Nil(t, tx)

	tx, err = _api.GetTransactionByBlockNumberAndIndex(999, 99)
	require.NoError(t, err)
	require.Nil(t, tx)
}

func TestGetTxByHash(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	blkHash := gethcmn.Hash{0x12, 0x34}
	block := newMdbBlock(blkHash, 123, []gethcmn.Hash{
		{0x56}, {0x78}, {0x90}, {0xAB},
	})
	_app.StoreBlocks(block)

	tx, err := _api.GetTransactionByHash(gethcmn.Hash{0x78})
	require.NoError(t, err)
	require.Equal(t, gethcmn.Hash{0x78}, tx.Hash)

	tx, err = _api.GetTransactionByHash(gethcmn.Hash{0xFF})
	require.NoError(t, err)
	require.Nil(t, tx)
}

func TestGetTxReceipt(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	blkHash := gethcmn.Hash{0x12, 0x34}
	block := testutils.NewMdbBlockBuilder().
		Hash(blkHash).Height(123).
		Tx(gethcmn.Hash{0x56}).
		Tx(gethcmn.Hash{0x78},
			types.Log{Address: gethcmn.Address{0xA1}, Topics: [][32]byte{{0xF1}, {0xF2}}},
			types.Log{Address: gethcmn.Address{0xA2}, Topics: [][32]byte{{0xF3}, {0xF4}}, Data: []byte{0xD1}}).
		Tx(gethcmn.Hash{0x90}).
		Tx(gethcmn.Hash{0xAB}).
		FailedTx(gethcmn.Hash{0xCD}, "failedTx", []byte{0xf1, 0xf2, 0xf3}).
		Build()
	_app.StoreBlocks(block)

	receipt, err := _api.GetTransactionReceipt(gethcmn.Hash{0x78})
	require.NoError(t, err)
	require.Equal(t, gethcmn.Hash{0x78}, receipt["transactionHash"])
	require.Equal(t, hexutil.Uint(0x1), receipt["status"])

	contractAddress, found := receipt["contractAddress"]
	require.True(t, found)
	require.Equal(t, nil, contractAddress)

	require.Len(t, receipt["logs"], 2)
	gethLogs := receipt["logs"].([]*gethtypes.Log)
	require.Equal(t, gethcmn.Address{0xA2}, gethLogs[1].Address)
	require.Equal(t, []byte{0xD1}, gethLogs[1].Data)

	receipt, err = _api.GetTransactionReceipt(gethcmn.Hash{0xCD})
	require.NoError(t, err)
	require.Equal(t, gethcmn.Hash{0xCD}, receipt["transactionHash"])
	require.Equal(t, hexutil.Uint(0x0), receipt["status"])
	require.Equal(t, "failedTx", receipt["statusStr"])
	require.Equal(t, "f1f2f3", receipt["outData"])
}

func TestGetTxReceipt_notFound(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	ret, err := _api.GetTransactionReceipt(gethcmn.Hash{0x78})
	require.NoError(t, err)
	require.Nil(t, ret)
}

func TestContractCreationTxToAddr(t *testing.T) {
	key, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	tx, blockNum, _ := _app.DeployContractInBlock(key, counterContractCreationBytecode)
	_app.EnsureTxSuccess(tx.Hash())

	blockResult, err := _api.GetBlockByNumber(gethrpc.BlockNumber(blockNum), true)
	require.NoError(t, err)
	require.Contains(t, testutils.ToJSON(blockResult), `"to":null`)

	blockHash := gethcmn.BytesToHash(blockResult["hash"].(hexutil.Bytes))
	blockResult, err = _api.GetBlockByHash(blockHash, true)
	require.NoError(t, err)
	require.Contains(t, testutils.ToJSON(blockResult), `"to":null`)

	txResult, err := _api.GetTransactionByHash(tx.Hash())
	require.NoError(t, err)
	require.Contains(t, testutils.ToJSON(txResult), `"to":null`)

	txResult, err = _api.GetTransactionByBlockNumberAndIndex(gethrpc.BlockNumber(blockNum), 0)
	require.NoError(t, err)
	require.Contains(t, testutils.ToJSON(txResult), `"to":null`)

	txResult, err = _api.GetTransactionByBlockHashAndIndex(blockHash, 0)
	require.NoError(t, err)
	require.Contains(t, testutils.ToJSON(txResult), `"to":null`)

	receiptResult, err := _api.GetTransactionReceipt(tx.Hash())
	require.NoError(t, err)
	require.Contains(t, testutils.ToJSON(receiptResult), `"to":null`)
}

func TestTxVRS(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	tx, blockNum := _app.MakeAndExecTxInBlock(key1, addr2, 123, nil)
	_app.EnsureTxSuccess(tx.Hash())

	blockResult, err := _api.GetBlockByNumber(gethrpc.BlockNumber(blockNum), true)
	require.NoError(t, err)
	checkTxVRS(t, tx, blockResult)

	blockHash := gethcmn.BytesToHash(blockResult["hash"].(hexutil.Bytes))
	blockResult, err = _api.GetBlockByHash(blockHash, true)
	require.NoError(t, err)
	checkTxVRS(t, tx, blockResult)

	txResult, err := _api.GetTransactionByHash(tx.Hash())
	require.NoError(t, err)
	checkTxVRS(t, tx, txResult)

	txResult, err = _api.GetTransactionByBlockNumberAndIndex(gethrpc.BlockNumber(blockNum), 0)
	require.NoError(t, err)
	checkTxVRS(t, tx, txResult)

	txResult, err = _api.GetTransactionByBlockHashAndIndex(blockHash, 0)
	require.NoError(t, err)
	checkTxVRS(t, tx, txResult)
}

func checkTxVRS(t *testing.T, tx *gethtypes.Transaction, resp interface{}) {
	v, r, s := tx.RawSignatureValues()
	respJSON := testutils.ToJSON(resp)
	require.Equal(t, v, hexutil.MustDecodeBig(regexp.MustCompile(`"v":"(0x[0-9a-fA-F]+)"`).FindStringSubmatch(respJSON)[1]), "V")
	require.Equal(t, r, hexutil.MustDecodeBig(regexp.MustCompile(`"r":"(0x[0-9a-fA-F]+)"`).FindStringSubmatch(respJSON)[1]), "R")
	require.Equal(t, s, hexutil.MustDecodeBig(regexp.MustCompile(`"s":"(0x[0-9a-fA-F]+)"`).FindStringSubmatch(respJSON)[1]), "S")
}

func TestCall_NoFromAddr(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	_, err := _api.Call(ethapi.CallArgs{}, latestBlockNumber())
	require.NoError(t, err)
}

func TestCall_Transfer(t *testing.T) {
	fromKey, fromAddr := testutils.GenKeyAndAddr()
	toKey, toAddr := testutils.GenKeyAndAddr()

	_app := testutils.CreateTestApp(fromKey, toKey)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	ret, err := _api.Call(ethapi.CallArgs{
		From:  &fromAddr,
		To:    &toAddr,
		Value: testutils.ToHexutilBig(10),
	}, latestBlockNumber())
	require.NoError(t, err)
	require.Equal(t, []byte{}, []byte(ret))

	_, err = _api.Call(ethapi.CallArgs{
		From:  &fromAddr,
		To:    &toAddr,
		Value: testutils.ToHexutilBig(math.MaxInt64),
	}, latestBlockNumber())
	require.Error(t, err)
	//require.Equal(t, []byte{}, []byte(ret))
}

func TestCall_DeployContract(t *testing.T) {
	fromKey, fromAddr := testutils.GenKeyAndAddr()

	_app := testutils.CreateTestApp(fromKey)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	ret, err := _api.Call(ethapi.CallArgs{
		From: &fromAddr,
		Data: testutils.ToHexutilBytes(counterContractCreationBytecode),
	}, latestBlockNumber())
	require.NoError(t, err)
	require.Equal(t, []byte{}, []byte(ret))
}

func TestCall_RunGetter(t *testing.T) {
	fromKey, fromAddr := testutils.GenKeyAndAddr()

	_app := testutils.CreateTestApp(fromKey)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	// deploy contract
	tx := ethutils.NewTx(0, nil, big.NewInt(0), 100000, big.NewInt(1),
		counterContractCreationBytecode)
	tx = testutils.MustSignTx(tx, _app.ChainID().ToBig(), fromKey)
	_app.ExecTxInBlock(tx)
	contractAddr := gethcrypto.CreateAddress(fromAddr, tx.Nonce())
	rtCode, err := _api.GetCode(contractAddr, latestBlockNumber())
	require.NoError(t, err)
	require.True(t, len(rtCode) > 0)

	// call contract
	data := counterContractABI.MustPack("counter")
	results, err := _api.Call(ethapi.CallArgs{
		//From: &fromAddr,
		To:   &contractAddr,
		Data: testutils.ToHexutilBytes(data),
	}, latestBlockNumber())
	require.NoError(t, err)
	require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000000",
		hex.EncodeToString(results))
}

func TestEstimateGas(t *testing.T) {
	fromKey, fromAddr := testutils.GenKeyAndAddr()

	_app := testutils.CreateTestApp(fromKey)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	ret, err := _api.EstimateGas(ethapi.CallArgs{
		From: &fromAddr,
		Data: testutils.ToHexutilBytes(counterContractCreationBytecode),
	}, nil)
	require.NoError(t, err)
	require.Equal(t, 96908, int(ret))
}

func TestCall_Transfer_Random(t *testing.T) {
	n := testutils.GetIntEvn("TEST_CALL_TRANSFER_RANDOM_COUNT", 1)
	for i := 0; i < n; i++ {
		testRandomTransfer()
	}
}

func testRandomTransfer() {
	fromKey, fromAddr := testutils.GenKeyAndAddr()
	toKey, toAddr := testutils.GenKeyAndAddr()

	_app := testutils.CreateTestApp(fromKey, toKey)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	w := sync.WaitGroup{}
	w.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			_, _ = _api.Call(ethapi.CallArgs{
				From:  &fromAddr,
				To:    &toAddr,
				Value: testutils.ToHexutilBig(10),
			}, latestBlockNumber())
			w.Done()
		}()
	}
	w.Wait()
}

func TestArchiveQuery_nonArchiveMode(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	defer _app.Destroy()
	_api := createEthAPI(_app)

	tx, _ := _app.MakeAndExecTxInBlock(key1, addr2, 1000, nil)
	_app.EnsureTxSuccess(tx.Hash())

	// no errors
	for h := gethrpc.PendingBlockNumber; h < 10; h++ {
		require.Equal(t, uint64(1), getTxCount(_api, addr1, h))
		require.Equal(t, uint64(9999000), getBalance(_api, addr1, h))
		require.Equal(t, testutils.UintToBytes32(0), getStorageAt(_api, addr1, "0x0", h))
		require.Len(t, getCode(_api, addr1, h), 0)
	}
}

func TestArchiveQuery_futureBlock(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestAppInArchiveMode(key1, key2)
	defer _app.Destroy()
	_api := createEthAPI(_app)

	tx, _ := _app.MakeAndExecTxInBlock(key1, addr2, 1000, nil)
	_app.EnsureTxSuccess(tx.Hash())

	blockNum := wrapBlockNumber(10)
	errMsg := "block has not been mined"

	_, err := _api.GetTransactionCount(addr1, blockNum)
	require.Equal(t, errMsg, err.Error())
	_, err = _api.GetBalance(addr1, blockNum)
	require.Equal(t, errMsg, err.Error())
	_, err = _api.GetCode(addr1, blockNum)
	require.Equal(t, errMsg, err.Error())
	_, err = _api.GetStorageAt(addr1, "0x0123", blockNum)
	require.Equal(t, errMsg, err.Error())
	_, err = _api.Call(rpctypes.CallArgs{}, blockNum)
	require.Equal(t, errMsg, err.Error())
	_, err = _api.EstimateGas(rpctypes.CallArgs{}, &blockNum)
	require.Equal(t, errMsg, err.Error())
}

func TestArchiveQuery_pendingBlock(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestAppInArchiveMode(key1, key2)
	defer _app.Destroy()
	_api := createEthAPI(_app)

	tx, _ := _app.MakeAndExecTxInBlock(key1, addr2, 1000, nil)
	_app.EnsureTxSuccess(tx.Hash())

	blockNum := wrapBlockNumber(gethrpc.PendingBlockNumber)
	errMsg := "pending block is not supported"

	_, err := _api.GetTransactionCount(addr1, blockNum)
	require.Equal(t, errMsg, err.Error())
	_, err = _api.GetBalance(addr1, blockNum)
	require.Equal(t, errMsg, err.Error())
	_, err = _api.GetCode(addr1, blockNum)
	require.Equal(t, errMsg, err.Error())
	_, err = _api.GetStorageAt(addr1, "0x0123", blockNum)
	require.Equal(t, errMsg, err.Error())
	_, err = _api.Call(rpctypes.CallArgs{}, blockNum)
	require.Equal(t, errMsg, err.Error())
	_, err = _api.EstimateGas(rpctypes.CallArgs{}, &blockNum)
	require.Equal(t, errMsg, err.Error())
}

func TestArchiveQuery_getTxCountAndBalance(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestAppInArchiveMode(key1, key2)
	defer _app.Destroy()
	_api := createEthAPI(_app)

	for i := int64(0); i < 5; i++ {
		tx, h := _app.MakeAndExecTxInBlock(key1, addr2, 1000, nil)
		_app.EnsureTxSuccess(tx.Hash())
		require.Equal(t, i*2+1, h) // 1, 3, 5, 7, 9, ...
	}

	for i := 0; i < 10; i++ {
		h := gethrpc.BlockNumber(i)
		require.Equal(t, uint64((h+1)/2), getTxCount(_api, addr1, h))
		require.Equal(t, uint64(10000000-1000*((h+1)/2)), getBalance(_api, addr1, h))
	}
	require.Equal(t, uint64(5), getTxCount(_api, addr1, -1))
	require.Equal(t, uint64(9995000), getBalance(_api, addr1, -1))
}

func TestArchiveQuery_getCodeAndStorageAt(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestAppInArchiveMode(key1, key2)
	defer _app.Destroy()
	_api := createEthAPI(_app)

	for i := int64(0); i < 3; i++ {
		tx, _ := _app.MakeAndExecTxInBlock(key1, addr2, 1000, nil)
		_app.EnsureTxSuccess(tx.Hash())
	}

	tx, h, counterAddr := _app.DeployContractInBlock(key1, counterContractCreationBytecode)
	_app.EnsureTxSuccess(tx.Hash())
	require.Equal(t, int64(7), h)

	tx1, h1 := _app.MakeAndExecTxInBlock(key1, counterAddr, 0,
		counterContractABI.MustPack("update", big.NewInt(111)))
	_app.EnsureTxSuccess(tx1.Hash())
	require.Equal(t, int64(9), h1)

	tx2, h2 := _app.MakeAndExecTxInBlock(key1, counterAddr, 0,
		counterContractABI.MustPack("update", big.NewInt(222)))
	_app.EnsureTxSuccess(tx2.Hash())
	require.Equal(t, int64(11), h2)

	// getCode
	for i := 0; i < 7; i++ {
		require.Len(t, getCode(_api, counterAddr, gethrpc.BlockNumber(i)), 0)
	}
	for i := 7; i < 12; i++ {
		require.Len(t, getCode(_api, counterAddr, gethrpc.BlockNumber(i)), 178)
	}

	// getStorageAt
	val0 := testutils.UintToBytes32(0)
	val1 := testutils.UintToBytes32(111)
	val3 := testutils.UintToBytes32(333)
	slot0 := "0x0"
	require.Equal(t, val0, getStorageAt(_api, counterAddr, slot0, 7))
	require.Equal(t, val0, getStorageAt(_api, counterAddr, slot0, 8))
	require.Equal(t, val1, getStorageAt(_api, counterAddr, slot0, 9))
	require.Equal(t, val1, getStorageAt(_api, counterAddr, slot0, 10))
	require.Equal(t, val3, getStorageAt(_api, counterAddr, slot0, 11))
}

func TestArchiveQuery_call(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestAppInArchiveMode(key1)
	defer _app.Destroy()
	_api := createEthAPI(_app)

	tx, h, counterAddr := _app.DeployContractInBlock(key1, counterContractCreationBytecode)
	_app.EnsureTxSuccess(tx.Hash())
	require.Equal(t, int64(1), h)

	for i := 0; i < 5; i++ {
		tx, h := _app.MakeAndExecTxInBlock(key1, counterAddr, 0,
			counterContractABI.MustPack("update", big.NewInt(int64(100+i))))
		_app.EnsureTxSuccess(tx.Hash())
		require.Equal(t, int64(3+i*2), h) // 3, 5, 7, 9, 11
	}

	data := counterContractABI.MustPack("counter")
	require.Equal(t, []byte{}, call(_api, addr1, counterAddr, data, 0))
	require.Equal(t, testutils.UintToBytes32(0), call(_api, addr1, counterAddr, data, 1))
	require.Equal(t, testutils.UintToBytes32(0), call(_api, addr1, counterAddr, data, 2))
	require.Equal(t, testutils.UintToBytes32(100), call(_api, addr1, counterAddr, data, 3))
	require.Equal(t, testutils.UintToBytes32(100), call(_api, addr1, counterAddr, data, 4))
	require.Equal(t, testutils.UintToBytes32(201), call(_api, addr1, counterAddr, data, 5))
	require.Equal(t, testutils.UintToBytes32(201), call(_api, addr1, counterAddr, data, 6))
	require.Equal(t, testutils.UintToBytes32(303), call(_api, addr1, counterAddr, data, 7))
	require.Equal(t, testutils.UintToBytes32(303), call(_api, addr1, counterAddr, data, 8))
	require.Equal(t, testutils.UintToBytes32(406), call(_api, addr1, counterAddr, data, 9))
	require.Equal(t, testutils.UintToBytes32(406), call(_api, addr1, counterAddr, data, 10))
	require.Equal(t, testutils.UintToBytes32(510), call(_api, addr1, counterAddr, data, 11))
	require.Equal(t, testutils.UintToBytes32(510), call(_api, addr1, counterAddr, data, -1))
}

func TestArchiveQuery_blockNum(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestAppInArchiveMode(key1, key2)
	defer _app.Destroy()
	_api := createEthAPI(_app)

	tx, h, blockNumAddr := _app.DeployContractInBlock(key1, blockNumContractCreationBytecode)
	_app.EnsureTxSuccess(tx.Hash())
	require.Equal(t, int64(1), h)

	for i := 0; i < 3; i++ {
		tx, h := _app.MakeAndExecTxInBlock(key1, addr2, 1000, nil)
		_app.EnsureTxSuccess(tx.Hash())
		require.Equal(t, int64(3+i*2), h) // 3, 5, 7
	}

	tx, h, counterAddr := _app.DeployContractInBlock(key1, counterContractCreationBytecode)
	_app.EnsureTxSuccess(tx.Hash())
	require.Equal(t, int64(9), h)
	require.Equal(t, uint64(9), getBlockNum(_api))

	data := blockNumContractABI.MustPack("getHeight")
	require.Equal(t, testutils.UintToBytes32(10), call(_api, addr1, blockNumAddr, data, -1)) // ??
	require.Equal(t, testutils.UintToBytes32(9), call(_api, addr1, blockNumAddr, data, 9))
	require.Equal(t, testutils.UintToBytes32(1), call(_api, addr1, blockNumAddr, data, 1))
	require.Equal(t, testutils.UintToBytes32(2), call(_api, addr1, blockNumAddr, data, 2))
	require.Equal(t, testutils.UintToBytes32(3), call(_api, addr1, blockNumAddr, data, 3))

	data = blockNumContractABI.MustPack("getBalance", addr1)
	require.Equal(t, testutils.UintToBytes32(9997000), call(_api, addr1, blockNumAddr, data, -1))
	require.Equal(t, testutils.UintToBytes32(9997000), call(_api, addr1, blockNumAddr, data, 9))
	require.Equal(t, testutils.UintToBytes32(9998000), call(_api, addr1, blockNumAddr, data, 5))
	require.Equal(t, testutils.UintToBytes32(9999000), call(_api, addr1, blockNumAddr, data, 3))
	require.Equal(t, testutils.UintToBytes32(10000000), call(_api, addr1, blockNumAddr, data, 1))

	data = blockNumContractABI.MustPack("getCodeSize", counterAddr)
	require.Equal(t, testutils.UintToBytes32(178), call(_api, addr1, blockNumAddr, data, -1))
	require.Equal(t, testutils.UintToBytes32(178), call(_api, addr1, blockNumAddr, data, 9))
	require.Equal(t, testutils.UintToBytes32(0), call(_api, addr1, blockNumAddr, data, 8))
	require.Equal(t, testutils.UintToBytes32(0), call(_api, addr1, blockNumAddr, data, 7))

	data = blockNumContractABI.MustPack("getBlockHash", big.NewInt(9))
	require.Equal(t, _app.GetBlock(9).Hash[:], call(_api, addr1, blockNumAddr, data, -1))
	//require.Equal(t, _app.GetBlock(9).Hash[:], call(_api, addr1, blockNumAddr, data, 9))

	data = blockNumContractABI.MustPack("getBlockHash", big.NewInt(7))
	require.Equal(t, _app.GetBlock(7).Hash[:], call(_api, addr1, blockNumAddr, data, -1))
	require.Equal(t, _app.GetBlock(7).Hash[:], call(_api, addr1, blockNumAddr, data, 9))
	require.Equal(t, _app.GetBlock(7).Hash[:], call(_api, addr1, blockNumAddr, data, 8))
	//require.Equal(t, _app.GetBlock(7).Hash[:], call(_api, addr1, blockNumAddr, data, 7))
}

func createEthAPI(_app *testutils.TestApp, testKeys ...string) *ethAPI {
	backend := api.NewBackend(nil, _app.App)
	return newEthAPI(backend, testKeys, _app.Logger())
}

func newMdbBlock(hash gethcmn.Hash, height int64,
	txs []gethcmn.Hash) *modbtypes.Block {

	b := testutils.NewMdbBlockBuilder().Hash(hash).Height(height)
	for _, txHash := range txs {
		b.Tx(txHash, []types.Log{
			{BlockNumber: uint64(height)},
		}...)
	}
	return b.Build()
}

func getBlockNum(_api *ethAPI) uint64 {
	n, err := _api.BlockNumber()
	if err != nil {
		panic(err)
	}
	return uint64(n)
}
func getTxCount(_api *ethAPI, addr gethcmn.Address, h gethrpc.BlockNumber) uint64 {
	txCount, err := _api.GetTransactionCount(addr, wrapBlockNumber(h))
	if err != nil {
		panic(err)
	}
	return uint64(*txCount)
}
func getBalance(_api *ethAPI, addr gethcmn.Address, h gethrpc.BlockNumber) uint64 {
	b, err := _api.GetBalance(addr, wrapBlockNumber(h))
	if err != nil {
		panic(err)
	}
	return (*b).ToInt().Uint64()
}
func getCode(_api *ethAPI, addr gethcmn.Address, h gethrpc.BlockNumber) []byte {
	c, err := _api.GetCode(addr, wrapBlockNumber(h))
	if err != nil {
		panic(err)
	}
	return c
}
func getStorageAt(_api *ethAPI, addr gethcmn.Address, key string, h gethrpc.BlockNumber) []byte {
	c, err := _api.GetStorageAt(addr, key, wrapBlockNumber(h))
	if err != nil {
		panic(err)
	}
	return c
}
func call(_api *ethAPI, from, to gethcmn.Address, data []byte, h gethrpc.BlockNumber) []byte {
	results, err := _api.Call(rpctypes.CallArgs{
		From: &from,
		To:   &to,
		Data: (*hexutil.Bytes)(&data),
	}, wrapBlockNumber(h))
	if err != nil {
		panic(err)
	}
	return results
}

func latestBlockNumber() gethrpc.BlockNumberOrHash {
	return wrapBlockNumber(gethrpc.LatestBlockNumber)
}
func wrapBlockNumber(blockNr gethrpc.BlockNumber) gethrpc.BlockNumberOrHash {
	return gethrpc.BlockNumberOrHashWithNumber(blockNr)
}
