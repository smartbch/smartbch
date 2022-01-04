package seps_test

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/seps"
)

var (
	sep206Addr        = seps.SEP206Addr
	sep206ABI         = seps.SEP20ABI
	sep206TotalSupply = big.NewInt(0).Mul(big.NewInt(21), big.NewInt(0).Exp(big.NewInt(10), big.NewInt(24), nil)) // 21*10^24
)

func TestEventSigs(t *testing.T) {
	require.Equal(t, "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
		sep206ABI.GetABI().Events["Transfer"].ID.Hex())
	require.Equal(t, "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925",
		sep206ABI.GetABI().Events["Approval"].ID.Hex())
}

func TestTokenInfo(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()

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
	_app := testutils.CreateTestApp(privKey1, privKey2)
	defer _app.Destroy()

	b1 := _app.GetBalance(addr1)
	b2 := _app.GetBalance(addr2)
	require.Equal(t, b1, callViewMethod(t, _app, "balanceOf", addr1))

	amt := big.NewInt(100)
	data1 := sep206ABI.MustPack("transfer", addr2, amt)
	_app.MakeAndExecTxInBlock(privKey1, sep206Addr, 0, data1)
	require.Equal(t, b1.Sub(b1, amt), callViewMethod(t, _app, "balanceOf", addr1))
	require.Equal(t, b2.Add(b2, amt), callViewMethod(t, _app, "balanceOf", addr2))
}

func TestTransferToNonExistingAddr(t *testing.T) {
	privKey1, addr1 := testutils.GenKeyAndAddr()
	_, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(privKey1)
	defer _app.Destroy()

	b1 := _app.GetBalance(addr1)
	require.Equal(t, b1, callViewMethod(t, _app, "balanceOf", addr1))
	require.Equal(t, uint64(0), callViewMethod(t, _app, "balanceOf", addr2).(*big.Int).Uint64())

	amt := big.NewInt(100)
	data1 := sep206ABI.MustPack("transfer", addr2, amt)
	_app.MakeAndExecTxInBlock(privKey1, sep206Addr, 0, data1)
	require.Equal(t, b1.Sub(b1, amt), callViewMethod(t, _app, "balanceOf", addr1))
	require.Equal(t, amt, callViewMethod(t, _app, "balanceOf", addr2))
}

func TestAllowance(t *testing.T) {
	ownerKey, ownerAddr := testutils.GenKeyAndAddr()
	spenderKey, spenderAddr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(ownerKey, spenderKey)
	defer _app.Destroy()

	a0 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(0), a0.(*big.Int).Uint64())

	data1 := sep206ABI.MustPack("approve", spenderAddr, big.NewInt(12345))
	tx1, h1 := _app.MakeAndExecTxInBlock(ownerKey, sep206Addr, 0, data1)
	checkTx(t, _app, h1, tx1.Hash())
	a1 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(12345), a1.(*big.Int).Uint64())

	data2 := sep206ABI.MustPack("increaseAllowance", spenderAddr, big.NewInt(123))
	tx2, h2 := _app.MakeAndExecTxInBlock(ownerKey, sep206Addr, 0, data2)
	checkTx(t, _app, h2, tx2.Hash())
	a2 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(12468), a2.(*big.Int).Uint64())

	data3 := sep206ABI.MustPack("decreaseAllowance", spenderAddr, big.NewInt(456))
	tx3, h3 := _app.MakeAndExecTxInBlock(ownerKey, sep206Addr, 0, data3)
	checkTx(t, _app, h3, tx3.Hash())
	a3 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(12012), a3.(*big.Int).Uint64())
}

func TestTransferFrom(t *testing.T) {
	ownerKey, ownerAddr := testutils.GenKeyAndAddr()
	spenderKey, spenderAddr := testutils.GenKeyAndAddr()
	_, receiptAddr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(ownerKey, spenderKey)
	defer _app.Destroy()

	data1 := sep206ABI.MustPack("approve", spenderAddr, big.NewInt(12345))
	tx1, h1 := _app.MakeAndExecTxInBlock(ownerKey, sep206Addr, 0, data1)
	checkTx(t, _app, h1, tx1.Hash())
	a1 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(12345), a1.(*big.Int).Uint64())

	data2 := sep206ABI.MustPack("transferFrom", ownerAddr, receiptAddr, big.NewInt(345))
	tx2, h2 := _app.MakeAndExecTxInBlock(spenderKey, sep206Addr, 0, data2)
	checkTx(t, _app, h2, tx2.Hash())
	a2 := callViewMethod(t, _app, "allowance", ownerAddr, spenderAddr)
	require.Equal(t, uint64(12000), a2.(*big.Int).Uint64())
}

func TestTransferFrom_drainBCH(t *testing.T) {
	ownerKey, ownerAddr := testutils.GenKeyAndAddr()
	spenderKey, spenderAddr := testutils.GenKeyAndAddr()
	_, receiptAddr := testutils.GenKeyAndAddr()

	initAmt := testutils.HexToU256("0xde0b6b3a7640000") // 1 BCH
	_app := testutils.CreateTestAppWithArgs(testutils.TestAppInitArgs{
		InitAmt:  initAmt,
		PrivKeys: []string{ownerKey, spenderKey},
	})
	defer _app.Destroy()

	data1 := sep206ABI.MustPack("approve", spenderAddr, initAmt.ToBig())
	tx1, _ := _app.MakeAndExecTxInBlock(ownerKey, sep206Addr, 0, data1)
	_app.EnsureTxSuccess(tx1.Hash())

	data2 := sep206ABI.MustPack("transferFrom", ownerAddr, receiptAddr, initAmt.ToBig())
	tx2, _ := _app.MakeAndExecTxInBlock(spenderKey, sep206Addr, 0, data2)
	_app.EnsureTxSuccess(tx2.Hash())
	require.Equal(t, uint64(0), _app.GetBalance(ownerAddr).Uint64())
}

func TestTransferFrom_mustLeaveMargin(t *testing.T) {
	ownerKey, ownerAddr := testutils.GenKeyAndAddr()
	spenderKey, spenderAddr := testutils.GenKeyAndAddr()
	_, receiptAddr := testutils.GenKeyAndAddr()

	startHeight := int64(28012340)
	initAmt := testutils.HexToU256("0xde0b6b3a7640000") // 1 BCH
	_app := testutils.CreateTestAppWithArgs(testutils.TestAppInitArgs{
		StartHeight: &startHeight,
		InitAmt:     initAmt,
		PrivKeys:    []string{ownerKey, spenderKey},
	})
	defer _app.Destroy()

	data1 := sep206ABI.MustPack("approve", spenderAddr, initAmt.ToBig())
	tx1, _ := _app.MakeAndExecTxInBlock(ownerKey, sep206Addr, 0, data1)
	_app.EnsureTxSuccess(tx1.Hash())

	data2 := sep206ABI.MustPack("transferFrom", ownerAddr, receiptAddr, initAmt.ToBig())
	tx2, _ := _app.MakeAndExecTxInBlock(spenderKey, sep206Addr, 0, data2)
	_app.EnsureTxFailed(tx2.Hash(), "insufficient-balance")
	require.Equal(t, initAmt.ToBig(), _app.GetBalance(ownerAddr))

	minMargin := testutils.HexToU256("0x38d7ea4c68000") // 0.001BCH
	data3 := sep206ABI.MustPack("transferFrom", ownerAddr, receiptAddr, initAmt.Sub(initAmt, minMargin).ToBig())
	tx3, _ := _app.MakeAndExecTxInBlock(spenderKey, sep206Addr, 0, data3)
	_app.EnsureTxSuccess(tx3.Hash())
	require.Equal(t, minMargin.ToBig(), _app.GetBalance(ownerAddr))
}

func TestTransferEvent(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	defer _app.Destroy()

	//_, contractAddr := _app.DeployContractInBlock(1, key1, _myTokenCreationBytecode)
	//require.NotEmpty(t, _app.GetCode(contractAddr))
	contractAddr := sep206Addr

	// addr1 => addr2
	data := sep206ABI.MustPack("transfer", addr2, big.NewInt(100))
	tx1, _ := _app.MakeAndExecTxInBlock(key1, contractAddr, 0, data)

	_app.WaitMS(200)
	tx1Query := _app.GetTx(tx1.Hash())
	require.Equal(t, "success", tx1Query.StatusStr)
	require.Len(t, tx1Query.Logs, 1)
	require.Len(t, tx1Query.Logs[0].Topics, 3)

	// event Transfer(address indexed _from, address indexed _to, uint256 _value)
	log0 := tx1Query.Logs[0]
	require.Equal(t, sep206ABI.GetABI().Events["Transfer"].ID.Hex(), "0x"+hex.EncodeToString(log0.Topics[0][:]))
	require.Equal(t, strings.ToLower(addr1.Hex()), "0x"+hex.EncodeToString(log0.Topics[1][12:]))
	require.Equal(t, strings.ToLower(addr2.Hex()), "0x"+hex.EncodeToString(log0.Topics[2][12:]))
	require.Equal(t, []interface{}{big.NewInt(100)}, sep206ABI.MustUnpack("Transfer", log0.Data))
}

func TestApproveEvent(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	defer _app.Destroy()

	//_, contractAddr := _app.DeployContractInBlock(1, key1, _myTokenCreationBytecode)
	//require.NotEmpty(t, _app.GetCode(contractAddr))
	contractAddr := sep206Addr

	// addr1 => addr2
	data := sep206ABI.MustPack("approve", addr2, big.NewInt(123))
	tx1, _ := _app.MakeAndExecTxInBlock(key1, contractAddr, 0, data)

	_app.WaitMS(200)
	tx1Query := _app.GetTx(tx1.Hash())
	require.Equal(t, "success", tx1Query.StatusStr)
	require.Len(t, tx1Query.Logs, 1)
	require.Len(t, tx1Query.Logs[0].Topics, 3)

	// event Approval(address indexed _owner, address indexed _spender, uint256 _value)
	log0 := tx1Query.Logs[0]
	require.Equal(t, sep206ABI.GetABI().Events["Approval"].ID.Hex(), "0x"+hex.EncodeToString(log0.Topics[0][:]))
	require.Equal(t, strings.ToLower(addr1.Hex()), "0x"+hex.EncodeToString(log0.Topics[1][12:]))
	require.Equal(t, strings.ToLower(addr2.Hex()), "0x"+hex.EncodeToString(log0.Topics[2][12:]))
	require.Equal(t, []interface{}{big.NewInt(123)}, sep206ABI.MustUnpack("Transfer", log0.Data))
}

func callViewMethod(t *testing.T, _app *testutils.TestApp, selector string, args ...interface{}) interface{} {
	data := sep206ABI.MustPack(selector, args...)
	statusCode, statusStr, output := _app.Call(gethcmn.Address{}, sep206Addr, data)
	require.Equal(t, 0, statusCode, selector)
	require.Equal(t, "success", statusStr, selector)
	result := sep206ABI.MustUnpack(selector, output)
	require.Len(t, result, 1, selector)
	return result[0]
}

func checkTx(t *testing.T, _app *testutils.TestApp, h int64, txHash gethcmn.Hash) {
	blk := _app.GetBlock(h)
	require.Equal(t, h, blk.Number)
	require.Len(t, blk.Transactions, 1)
	txInBlk := _app.GetTx(blk.Transactions[0])
	require.Equal(t, gethtypes.ReceiptStatusSuccessful, txInBlk.Status)
	require.Equal(t, "success", txInBlk.StatusStr)
	require.Equal(t, txHash, gethcmn.Hash(txInBlk.Hash))
}
