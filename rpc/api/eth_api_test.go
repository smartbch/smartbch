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

	b, err := _api.GetBalance(addr, -1)
	require.NoError(t, err)
	require.Equal(t, "0x989680", b.String())

	b2, err := _api.GetBalance(addr2, -1)
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

	nonce, err := _api.GetTransactionCount(addr1, -1)
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(0), *nonce)
	nonce, err = _api.GetTransactionCount(addr2, -1)
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(0), *nonce)

	for i := 0; i < 3; i++ {
		tx, _ := _app.MakeAndExecTxInBlock(key1, addr2, 100, nil)
		_app.EnsureTxSuccess(tx.Hash())

		nonce, err = _api.GetTransactionCount(addr1, -1)
		require.NoError(t, err)
		require.Equal(t, hexutil.Uint64(i+1), *nonce)
	}

	nonce, err = _api.GetTransactionCount(addr2, -1)
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(0), *nonce)

	nonce, err = _api.GetTransactionCount(addr3, -1)
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

	c, err := _api.GetCode(addr, -1)
	require.NoError(t, err)
	require.Equal(t, "0x1234", c.String())

	c, err = _api.GetCode(addr2, -1)
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
	ctx.SetStorageAt(seq, string(sKey), []byte{0x12, 0x34})
	ctx.Close(true)
	_app.CloseTxEngineContext()
	_app.CloseTrunk()

	sVal, err := _api.GetStorageAt(addr, "0x"+hex.EncodeToString(sKey), -1)
	require.NoError(t, err)
	require.Equal(t, "0x1234", sVal.String())

	sVal, err = _api.GetStorageAt(addr, "0x7890", -1)
	require.NoError(t, err)
	require.Equal(t, "0x", sVal.String())

	sVal, err = _api.GetStorageAt(addr2, "0x7890", -1)
	require.NoError(t, err)
	require.Equal(t, "0x", sVal.String())
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

	_, err := _api.Call(ethapi.CallArgs{}, -1)
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
	}, -1)
	require.NoError(t, err)
	require.Equal(t, []byte{}, []byte(ret))

	_, err = _api.Call(ethapi.CallArgs{
		From:  &fromAddr,
		To:    &toAddr,
		Value: testutils.ToHexutilBig(math.MaxInt64),
	}, -1)
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
	}, -1)
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
	rtCode, err := _api.GetCode(contractAddr, -1)
	require.NoError(t, err)
	require.True(t, len(rtCode) > 0)

	// call contract
	data := counterContractABI.MustPack("counter")
	results, err := _api.Call(ethapi.CallArgs{
		//From: &fromAddr,
		To:   &contractAddr,
		Data: testutils.ToHexutilBytes(data),
	}, -1)
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
			}, -1)
			w.Done()
		}()
	}
	w.Wait()
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
