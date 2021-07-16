package app_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	//"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/seps"
	"github.com/smartbch/smartbch/staking"
)

//func TestMain(m *testing.M) {
//	ebp.TxRunnerParallelCount = 1
//	ebp.PrepareParallelCount = 1
//}

func TestGetBalance(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	require.Equal(t, testutils.DefaultInitBalance, _app.GetBalance(addr).Uint64())
}

func TestTransferOK(t *testing.T) {
	staking.DefaultMinGasPrice = 0
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	_app.WaitLock()
	defer _app.Destroy()
	initBal := testutils.DefaultInitBalance
	require.Equal(t, initBal, _app.GetBalance(addr1).Uint64())
	require.Equal(t, initBal, _app.GetBalance(addr2).Uint64())

	tx, _ := _app.MakeAndExecTxInBlock(key1, addr2, 100, nil)
	_app.WaitMS(100)
	_app.EnsureTxSuccess(tx.Hash())

	require.Equal(t, initBal-100, _app.GetBalance(addr1).Uint64())
	require.Equal(t, initBal+100, _app.GetBalance(addr2).Uint64())
	require.Equal(t, int64(2), _app.GetLatestBlockNum())
}

func TestTransferFailed(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	defer _app.Destroy()
	initBal := testutils.DefaultInitBalance
	require.Equal(t, initBal, _app.GetBalance(addr1).Uint64())
	require.Equal(t, initBal, _app.GetBalance(addr2).Uint64())

	// insufficient balance
	tx, _ := _app.MakeAndExecTxInBlock(key1, addr2, 10000001, nil)

	// check tx status
	_app.WaitMS(100)
	_app.EnsureTxFailed(tx.Hash(), "insufficient-balance")

	require.Equal(t, initBal, _app.GetBalance(addr1).Uint64())
	require.Equal(t, initBal, _app.GetBalance(addr2).Uint64())
}

func TestBlock(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	defer _app.Destroy()

	_, h1 := _app.MakeAndExecTxInBlock(key1, addr2, 100, nil)
	_app.WaitMS(50)

	blk1 := _app.GetBlock(h1)
	require.Equal(t, h1, blk1.Number)
	require.Len(t, blk1.Transactions, 1)

	h2 := _app.ExecTxInBlock(nil)
	_app.WaitMS(50)

	blk2 := _app.GetBlock(h2)
	require.Equal(t, h2, blk2.Number)
	require.Len(t, blk2.Transactions, 0)
}

func TestCheckTx(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()
	require.Equal(t, uint64(10000000), _app.GetBalance(addr1).Uint64())

	//tx decode failed
	tx := ethutils.NewTx(1, &addr1, big.NewInt(100), 100000, big.NewInt(1), nil)
	data, _ := tx.MarshalJSON()
	res := _app.CheckTx(abci.RequestCheckTx{
		Tx:   data,
		Type: abci.CheckTxType_New,
	})
	require.Equal(t, app.CannotDecodeTx, res.Code)

	//sender decode failed
	tx = ethutils.NewTx(1, &addr1, big.NewInt(100), 100000, big.NewInt(1), nil)
	res = _app.CheckTx(abci.RequestCheckTx{
		Tx:   append(testutils.MustEncodeTx(tx), 0x01),
		Type: abci.CheckTxType_New,
	})
	require.Equal(t, app.CannotRecoverSender, res.Code)

	//tx nonce mismatch
	tx = ethutils.NewTx(1, &addr1, big.NewInt(100), 100000, big.NewInt(1), nil)
	tx = testutils.MustSignTx(tx, _app.ChainID().ToBig(), key1)
	require.Equal(t, app.AccountNonceMismatch, _app.CheckNewTxABCI(tx))

	//gas fee not pay
	tx = ethutils.NewTx(0, &addr1, big.NewInt(100), 900_0000, big.NewInt(10), nil)
	tx = testutils.MustSignTx(tx, _app.ChainID().ToBig(), key1)
	require.Equal(t, app.CannotPayGasFee, _app.CheckNewTxABCI(tx))

	//ok
	tx = ethutils.NewTx(0, &addr1, big.NewInt(100), 100000, big.NewInt(10), nil)
	tx = testutils.MustSignTx(tx, _app.ChainID().ToBig(), key1)
	require.Equal(t, uint32(0), _app.CheckNewTxABCI(tx))
}

func TestCheckTxNonce_serial(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	defer _app.Destroy()

	tx1, _ := _app.MakeAndSignTx(key1, &addr2, 1, nil)
	tx2, _ := _app.MakeAndSignTx(key1, &addr2, 2, nil)
	require.Equal(t, uint64(0), tx1.Nonce())
	require.Equal(t, uint64(0), tx2.Nonce())

	require.Equal(t, uint32(0), _app.CheckNewTxABCI(tx1))
	require.Equal(t, app.AccountNonceMismatch, _app.CheckNewTxABCI(tx2))
}

func TestCheckTx_hasPending(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	defer _app.Destroy()

	tx1, _ := _app.MakeAndSignTx(key1, &addr2, 1, nil)
	tx2, _ := _app.MakeAndSignTx(key1, &addr2, 2, nil)
	_app.AddTxsInBlock(1, tx1)
	require.Equal(t, app.HasPendingTx, _app.CheckNewTxABCI(tx2))
}

func TestCheckTx_sep206SenderSet(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	key3, addr3 := testutils.GenKeyAndAddr()
	key4, addr4 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2, key3, key4)
	defer _app.Destroy()

	// addr1 => addr2
	data := seps.PackSEP20Transfer(addr2, big.NewInt(101))
	tx1, _ := _app.MakeAndSignTx(key1, &seps.SEP206Addr, 0, data)

	// addr2 => addr3
	data = seps.PackSEP20Transfer(addr3, big.NewInt(102))
	tx2, _ := _app.MakeAndSignTx(key2, &seps.SEP206Addr, 0, data)

	// addr3 => addr4
	data = seps.PackSEP20Transfer(addr4, big.NewInt(103))
	tx3, _ := _app.MakeAndSignTx(key3, &seps.SEP206Addr, 0, data)

	// addr4 => addr1
	data = seps.PackSEP20Transfer(addr1, big.NewInt(103))
	tx4, _ := _app.MakeAndSignTx(key4, &seps.SEP206Addr, 0, data)

	_app.ExecTxsInBlock(tx1, tx2, tx3)
	_app.EnsureTxSuccess(tx1.Hash())
	_app.EnsureTxSuccess(tx2.Hash())
	_app.EnsureTxSuccess(tx3.Hash())

	require.True(t, _app.IsInSenderSet(addr1))
	require.True(t, _app.IsInSenderSet(addr2))
	require.True(t, _app.IsInSenderSet(addr3))
	require.False(t, _app.IsInSenderSet(addr4))

	checkResp := _app.RecheckTxABCI(tx1)
	require.Equal(t, app.AccountNonceMismatch, checkResp)

	checkResp = _app.RecheckTxABCI(tx4)
	require.Equal(t, abci.CodeTypeOK, checkResp)
}

func TestIncorrectNonceErr(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()

	tx1, _ := _app.MakeAndSignTx(key1, &addr2, 1, nil)
	tx2, _ := _app.MakeAndSignTx(key1, &addr2, 2, nil)

	h := _app.ExecTxsInBlock(tx1, tx2)
	require.Len(t, _app.GetBlock(h-1).Transactions, 1)
	require.Len(t, _app.GetBlock(h).Transactions, 1)
	_app.EnsureTxSuccess(tx1.Hash())
	_app.EnsureTxFailed(tx2.Hash(), "incorrect nonce")

	_app.WaitMS(200)
	txs := _app.GetTxsByAddr(addr1)
	require.Len(t, txs, 1)
	require.Equal(t, tx1.Hash().Hex(),
		"0x"+hex.EncodeToString(txs[0].Hash[:]))
}

func TestGasRefund_NoAdjustGasUsed(t *testing.T) {
	oldAdjustGasUsed := ebp.AdjustGasUsed
	ebp.AdjustGasUsed = false
	defer func() { ebp.AdjustGasUsed = oldAdjustGasUsed }()

	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	defer _app.Destroy()

	initBal := testutils.DefaultInitBalance
	require.Equal(t, initBal, _app.GetBalance(addr1).Uint64())
	require.Equal(t, initBal, _app.GetBalance(addr2).Uint64())
	tx, _ := _app.MakeAndExecTxInBlockWithGas(key1, addr2, 100, nil, 1000000, 1)
	_app.EnsureTxSuccess(tx.Hash())

	moTx := _app.GetTx(tx.Hash())
	require.Equal(t, uint64(21000), moTx.GasUsed)

	require.Equal(t, initBal-100-21000, _app.GetBalance(addr1).Uint64())
	require.Equal(t, initBal+100, _app.GetBalance(addr2).Uint64())
}

func TestGasRefund_AdjustGasUsed(t *testing.T) {
	oldAdjustGasUsed := ebp.AdjustGasUsed
	ebp.AdjustGasUsed = true
	defer func() { ebp.AdjustGasUsed = oldAdjustGasUsed }()

	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	defer _app.Destroy()

	tests := []struct {
		gasLimit uint64
		gasUsed  uint64
	}{
		{gasLimit: 21000, gasUsed: 21000},
		{gasLimit: 21001, gasUsed: 21000},
		{gasLimit: 42000, gasUsed: 21000},
		{gasLimit: 42001, gasUsed: 21000 + 21001/2},
		{gasLimit: 50000, gasUsed: 21000 + 29000/2},
		{gasLimit: 84000, gasUsed: 21000 + 63000/2},
		{gasLimit: 84001, gasUsed: 84001},
		{gasLimit: 99999, gasUsed: 99999},
	}

	for _, test := range tests {
		bal1 := _app.GetBalance(addr1).Uint64()
		bal2 := _app.GetBalance(addr2).Uint64()

		tx, _ := _app.MakeAndExecTxInBlockWithGas(key1, addr2, 100, nil, test.gasLimit, 1)
		_app.EnsureTxSuccess(tx.Hash())

		moTx := _app.GetTx(tx.Hash())
		require.Equal(t, test.gasUsed, moTx.GasUsed)

		require.Equal(t, bal1-100-test.gasUsed, _app.GetBalance(addr1).Uint64())
		require.Equal(t, bal2+100, _app.GetBalance(addr2).Uint64())
	}
}

func TestMinGas(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestAppWithInitAmt(uint256.NewInt().SetUint64(1e18), key1, key2)
	defer _app.Destroy()

	ctx := _app.GetRunTxContext()
	staking.SaveMinGasPrice(ctx, 10000, true)
	staking.SaveMinGasPrice(ctx, 10000, false)
	ctx.Close(true)
	_app.ExecTxsInBlock()

	tx, _ := _app.MakeAndSignTxWithGas(key1, &addr2, 100, nil, 100000, 1234)
	require.Equal(t, app.InvalidMinGasPrice, _app.CheckNewTxABCI(tx))
	_app.ExecTxInBlock(tx)
	_app.EnsureTxFailed(tx.Hash(), "invalid gas price")

	tx, _ = _app.MakeAndSignTxWithGas(key1, &addr2, 100, nil, 100000, 9999)
	require.Equal(t, app.InvalidMinGasPrice, _app.CheckNewTxABCI(tx))

	_app.ExecTxInBlock(tx)
	_app.EnsureTxFailed(tx.Hash(), "invalid gas price")

	tx, _ = _app.MakeAndSignTxWithGas(key1, &addr2, 100, nil, 100000, 10000)
	require.Equal(t, uint32(0), _app.CheckNewTxABCI(tx))
	_app.ExecTxInBlock(tx)
	_app.EnsureTxSuccess(tx.Hash())

	tx, _ = _app.MakeAndSignTxWithGas(key1, &addr2, 100, nil, 100000, 10001)
	require.Equal(t, uint32(0), _app.CheckNewTxABCI(tx))
	_app.ExecTxInBlock(tx)
	_app.EnsureTxSuccess(tx.Hash())

	tx, _ = _app.MakeAndSignTxWithGas(key1, &addr2, 100, nil, 100000, 12345)
	require.Equal(t, uint32(0), _app.CheckNewTxABCI(tx))
	_app.ExecTxInBlock(tx)
	_app.EnsureTxSuccess(tx.Hash())
}

func TestJson(t *testing.T) {
	//str := []byte("\"validators\":[\"PupuoOdnaRYJQUSzCsV5B6gBfkWiaI4Jmq8giG/KL0M=\",\"G0IgOw0f4hqpR0TX+ld5TzOyPI2+BuaYhjlHv6IiCHw=\",\"YdrD918WSVISQes6g5v5xI0x580OM2LMNUIRIS8EXjA=\",\"/opEYWd8xnLK95QN34+mrE666sSt/GARmJYgRUYnvb0=\",\"gM4A5vTY9vTgHOd00TTXPo7HyEHBkuIpvbUBw28DxrI=\",\"4kFUm8nRR2Tg3YCl55lOWbAGYi4fPQnHiCrWHWnEd3k=\",\"yb/5/EsybQ2rI9XkRQoJBAixvAoivV0mb9jqsEVSUj8=\",\"8MfS5Y24qXoACl45f3otSyOB1sCCgrXGX/SIPTuaC9Y=\",\"BAsO38HaA7XyMB8tAkI8ests8jdOeFe03j3QROKFVsg=\",\"We2gXsEqww2Q+NdVGbaWhR0nyrxP/FBv4TzJxNKMwb4=\"]}")

	type Val struct {
		Validators []ed25519.PubKey
	}
	v1 := Val{
		Validators: make([]ed25519.PubKey, 10),
	}
	for i := 0; i < 10; i++ {
		v1.Validators[i] = ed25519.GenPrivKey().PubKey().(ed25519.PubKey)
	}
	bz, _ := json.Marshal(v1.Validators)
	//fmt.Println(v1)
	//fmt.Printf("testValidator:%s\n", bz)
	v := Val{}
	err := json.Unmarshal(bz, &v.Validators)
	fmt.Println(v, err)
}

func TestRandomTxs(t *testing.T) {
	txCount := testutils.GetIntEvn("TEST_RANDOM_TXS_COUNT", 10)

	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	key3, addrTo1 := testutils.GenKeyAndAddr()
	key4, addrTo2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2, key3, key4)
	txLists := generateRandomTxs(txCount, _app.ChainID(), key1, key2, addrTo1, addrTo2)
	res1 := execRandomTxs(_app, txLists, addr1, addr2)
	_app.Destroy()

	_app = testutils.CreateTestApp(key1, key2, key3, key4)
	res2 := execRandomTxs(_app, txLists, addr1, addr2)
	_app.Destroy()

	require.Equal(t, res1[0], res2[0])
	require.Equal(t, res1[1], res2[1])
}

func execRandomTxs(_app *testutils.TestApp, txLists [][]*gethtypes.Transaction, from1, from2 common.Address) []uint64 {
	for i, txList := range txLists {
		_app.AddTxsInBlock(int64(i+1), txList...)
	}
	ctx := _app.GetCheckTxContext()
	defer ctx.Close(false)
	balanceFrom1 := ctx.GetAccount(from1).Balance().Uint64()
	balanceFrom2 := ctx.GetAccount(from2).Balance().Uint64()
	return []uint64{balanceFrom1, balanceFrom2}
}

func generateRandomTxs(count int, chainId *uint256.Int, key1, key2 string, to1, to2 common.Address) [][]*gethtypes.Transaction {
	rand.Seed(time.Now().UnixNano())
	lists := make([][]*gethtypes.Transaction, count)
	for k := 0; k < count; k++ {
		set := make([]*gethtypes.Transaction, 2000)
		for i := 0; i < 1000; i++ {
			nonce := uint64(rand.Int() % 200)
			value := int64(rand.Int()%100 + 1)
			tx := ethutils.NewTx(nonce, &to1, big.NewInt(value), 100000, big.NewInt(1), nil)
			tx = testutils.MustSignTx(tx, chainId.ToBig(), key1)
			set[i*2] = tx
			nonce = uint64(rand.Int() % 200)
			value = int64(rand.Int()%100 + 1)
			tx = ethutils.NewTx(nonce, &to2, big.NewInt(value), 100000, big.NewInt(1), nil)
			tx = testutils.MustSignTx(tx, chainId.ToBig(), key2)
			set[i*2+1] = tx
		}
		lists[k] = set
	}
	return lists
}
