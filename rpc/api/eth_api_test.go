package api

import (
	"bytes"
	"encoding/hex"
	"math"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"

	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/types"

	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/rpc/internal/ethapi"
)

const counterContract = `
// SPDX-License-Identifier: MIT
pragma solidity >=0.6.0;

contract Counter {

int public counter;

function update(int n) public {
  counter += n;
}

}
`

// code, rtCode, _ := testutils.MustCompileSolStr(counterContract)
var counterContractCreationBytecode = testutils.HexToBytes(`
608060405234801561001057600080fd5b5060b28061001f6000396000f3fe60
80604052348015600f57600080fd5b506004361060325760003560e01c806361
bc221a1460375780636299a6ef14604f575b600080fd5b603d606b565b604080
51918252519081900360200190f35b6069600480360360208110156063576000
80fd5b50356071565b005b60005481565b60008054909101905556fea2646970
6673582212205df2a10ba72894ded3e0a7ea8c57a79906cca125c3aafe3c979f
bd57e662c01d64736f6c634300060c0033
`)
var counterContractRuntimeBytecode = testutils.HexToBytes(`
6080604052348015600f57600080fd5b506004361060325760003560e01c8063
61bc221a1460375780636299a6ef14604f575b600080fd5b603d606b565b6040
8051918252519081900360200190f35b60696004803603602081101560635760
0080fd5b50356071565b005b60005481565b60008054909101905556fea26469
706673582212205df2a10ba72894ded3e0a7ea8c57a79906cca125c3aafe3c97
9fbd57e662c01d64736f6c634300060c0033
`)
var counterContractABI = testutils.MustParseABI(`
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

func TestBlockNum(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	ctx := _app.GetRunTxContext()
	ctx.Db.AddBlock(&modbtypes.Block{Height: 0x100}, -1)
	ctx.Db.AddBlock(nil, -1) //To Flush
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
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	acc := types.ZeroAccountInfo()
	acc.UpdateNonce(78)
	ctx := _app.GetRunTxContext()
	ctx.SetAccount(addr, acc)
	ctx.Close(true)
	_app.CloseTxEngineContext()
	_app.CloseTrunk()

	nonce, err := _api.GetTransactionCount(addr, 0)
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(78), *nonce)
}

func TestGetCode(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	ctx := _app.GetRunTxContext()
	code := bytes.Repeat([]byte{0xff}, 32)
	code = append(code, 0x12, 0x34)
	ctx.SetCode(addr, types.NewBytecodeInfo(code))
	ctx.Close(true)
	_app.CloseTxEngineContext()
	_app.CloseTrunk()

	c, err := _api.GetCode(addr, 0)
	require.NoError(t, err)
	require.Equal(t, "0x1234", c.String())
}

func TestGetStorageAt(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	ctx := _app.GetRunTxContext()
	code := bytes.Repeat([]byte{0xff}, 32)
	code = append(code, 0x12, 0x34)

	seq := ctx.GetAccount(addr).Sequence()
	sKeyHex := strings.Repeat("abcd", 16)
	sKey, err := hex.DecodeString(sKeyHex)
	require.NoError(t, err)

	ctx.SetStorageAt(seq, string(sKey), []byte{0x12, 0x34})
	ctx.Close(true)
	_app.CloseTxEngineContext()
	_app.CloseTrunk()

	sVal, err := _api.GetStorageAt(addr, "0x"+sKeyHex, 0)
	require.NoError(t, err)
	require.Equal(t, "0x1234", sVal.String())
}

func TestGetBlockByHash(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	hash := gethcmn.Hash{0x12, 0x34}
	block := newMdbBlock(hash, 123, nil)
	ctx := _app.GetRunTxContext()
	ctx.StoreBlock(block)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)

	block2, err := _api.GetBlockByHash(hash, true)
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(123), block2["number"])
	require.Equal(t, hexutil.Bytes(hash[:]), block2["hash"])
	require.Equal(t, hexutil.Uint64(200000000), block2["gasLimit"])
	require.Equal(t, hexutil.Uint64(0), block2["gasUsed"])
	require.Equal(t, hexutil.Bytes(nil), block2["extraData"])
}

func TestGetBlockByHash_notFound(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	blk, err := _api.GetBlockByHash(gethcmn.Hash{0x12, 0x34}, true)
	require.NoError(t, err)
	require.Nil(t, blk)
}

func TestGetBlockByNum(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	hash := gethcmn.Hash{0x12, 0x34}
	block := newMdbBlock(hash, 123, []gethcmn.Hash{
		{0x56}, {0x78}, {0x90},
	})
	ctx := _app.GetRunTxContext()
	ctx.StoreBlock(block)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)

	block2, err := _api.GetBlockByNumber(123, true)
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(123), block2["number"])
	require.Equal(t, hexutil.Bytes(hash[:]), block2["hash"])
	require.Equal(t, hexutil.Uint64(200000000), block2["gasLimit"])
	require.Equal(t, hexutil.Uint64(0), block2["gasUsed"])
	require.Equal(t, hexutil.Bytes(nil), block2["extraData"])
	require.Equal(t, "0x0000000000000000", block2["nonce"].(hexutil.Bytes).String())
	require.Len(t, block2["transactions"], 3)
}

func TestGetBlockByNum_notFound(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	blk, err := _api.GetBlockByNumber(99, true)
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
	ctx := _app.GetRunTxContext()
	ctx.StoreBlock(block)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)

	cnt := _api.GetBlockTransactionCountByHash(hash)
	require.Equal(t, hexutil.Uint(3), *cnt)
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
	ctx := _app.GetRunTxContext()
	ctx.StoreBlock(block)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)

	cnt := _api.GetBlockTransactionCountByNumber(123)
	require.Equal(t, hexutil.Uint(4), *cnt)
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
	ctx := _app.GetRunTxContext()
	ctx.StoreBlock(block)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)

	tx, err := _api.GetTransactionByBlockHashAndIndex(blkHash, 2)
	require.NoError(t, err)
	require.Equal(t, gethcmn.Hash{0x90}, tx.Hash)
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
	ctx := _app.GetRunTxContext()
	ctx.StoreBlock(block)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)

	tx, err := _api.GetTransactionByBlockNumberAndIndex(123, 1)
	require.NoError(t, err)
	require.Equal(t, gethcmn.Hash{0x78}, tx.Hash)
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
	ctx := _app.GetRunTxContext()
	ctx.StoreBlock(block)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)

	tx, err := _api.GetTransactionByHash(gethcmn.Hash{0x78})
	require.NoError(t, err)
	require.Equal(t, gethcmn.Hash{0x78}, tx.Hash)
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
		Build()
	ctx := _app.GetRunTxContext()
	ctx.StoreBlock(block)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)

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
}

func TestCall_NoFromAddr(t *testing.T) {
	_app := testutils.CreateTestApp()
	_app.WaitLock()
	defer _app.Destroy()
	_api := createEthAPI(_app)

	_, err := _api.Call(ethapi.CallArgs{}, 0)
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
	}, 0)
	require.NoError(t, err)
	require.Equal(t, []byte{}, []byte(ret))

	ret, err = _api.Call(ethapi.CallArgs{
		From:  &fromAddr,
		To:    &toAddr,
		Value: testutils.ToHexutilBig(math.MaxInt64),
	}, 0)
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
	}, 0)
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
	tx := gethtypes.NewContractCreation(0, big.NewInt(0), 100000, big.NewInt(1),
		counterContractCreationBytecode)
	tx = testutils.MustSignTx(tx, _app.ChainID().ToBig(), fromKey)
	_app.ExecTxInBlock(tx)
	contractAddr := gethcrypto.CreateAddress(fromAddr, tx.Nonce())
	rtCode, err := _api.GetCode(contractAddr, 0)
	require.NoError(t, err)
	require.True(t, len(rtCode) > 0)

	// call contract
	data := counterContractABI.MustPack("counter")
	results, err := _api.Call(ethapi.CallArgs{
		//From: &fromAddr,
		To:   &contractAddr,
		Data: testutils.ToHexutilBytes(data),
	}, 0)
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
	})
	require.NoError(t, err)
	require.Equal(t, 96908, int(ret))
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

func TestCall_Transfer_Random(t *testing.T) {
	for i := 0; i < 50; i++ {
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
			}, 0)
			w.Done()
		}()
	}
	w.Wait()
}
