package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/staking"
	"github.com/smartbch/smartbch/staking/types"
)

//func TestMain(m *testing.M) {
//	ebp.TxRunnerParallelCount = 1
//	ebp.PrepareParallelCount = 1
//}

func TestGetBalance(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key)
	defer DestroyTestApp(_app)
	require.Equal(t, uint64(10000000), getBalance(_app, addr).Uint64())
}

func TestTransferOK(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key1, key2)
	defer DestroyTestApp(_app)
	require.Equal(t, uint64(10000000), getBalance(_app, addr1).Uint64())
	require.Equal(t, uint64(10000000), getBalance(_app, addr2).Uint64())

	tx := gethtypes.NewTransaction(0, addr2, big.NewInt(100), 100000, big.NewInt(1), nil)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))

	testutils.ExecTxInBlock(_app, 1, tx)
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, uint64(10000000-100-21000), getBalance(_app, addr1).Uint64())
	require.Equal(t, uint64(10000000+100), getBalance(_app, addr2).Uint64())

	n := _app.GetLatestBlockNum()
	require.Equal(t, int64(2), n)

	blk1 := getBlock(_app, 1)
	require.Equal(t, int64(1), blk1.Number)
	require.Len(t, blk1.Transactions, 1)

	// check tx status
	moeTx := getTx(_app, tx.Hash())
	require.Equal(t, [32]byte(tx.Hash()), moeTx.Hash)
	require.Equal(t, uint64(1), moeTx.Status)
}

func TestTransferFailed(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key1, key2)
	defer DestroyTestApp(_app)
	require.Equal(t, uint64(10000000), getBalance(_app, addr1).Uint64())
	require.Equal(t, uint64(10000000), getBalance(_app, addr2).Uint64())

	// insufficient balance
	tx := gethtypes.NewTransaction(0, addr2, big.NewInt(10000001), 100000, big.NewInt(1), nil)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	testutils.ExecTxInBlock(_app, 1, tx)

	require.Equal(t, uint64(10000000-21000), getBalance(_app, addr1).Uint64())
	require.Equal(t, uint64(10000000), getBalance(_app, addr2).Uint64())

	// check tx status
	time.Sleep(100 * time.Millisecond)
	moeTx := getTx(_app, tx.Hash())
	require.Equal(t, [32]byte(tx.Hash()), moeTx.Hash)
	require.Equal(t, gethtypes.ReceiptStatusFailed, moeTx.Status)
	require.Equal(t, "balance-not-enough", moeTx.StatusStr)
}

func TestBlock(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key1, key2)
	defer DestroyTestApp(_app)

	tx := gethtypes.NewTransaction(0, addr2, big.NewInt(100), 100000, big.NewInt(1), nil)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	testutils.ExecTxInBlock(_app, 1, tx)
	time.Sleep(50 * time.Millisecond)

	blk1 := getBlock(_app, 1)
	require.Equal(t, int64(1), blk1.Number)
	require.Len(t, blk1.Transactions, 1)

	testutils.ExecTxInBlock(_app, 3, nil)
	time.Sleep(50 * time.Millisecond)

	blk3 := getBlock(_app, 3)
	require.Equal(t, int64(3), blk3.Number)
	require.Len(t, blk3.Transactions, 0)
}

func TestCheckTx(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key1)
	defer DestroyTestApp(_app)
	require.Equal(t, uint64(10000000), getBalance(_app, addr1).Uint64())

	//tx decode failed
	tx := gethtypes.NewTransaction(1, addr1, big.NewInt(100), 100000, big.NewInt(1), nil)
	data, _ := tx.MarshalJSON()
	res := _app.CheckTx(abci.RequestCheckTx{
		Tx:   data,
		Type: abci.CheckTxType_New,
	})
	require.Equal(t, CannotDecodeTx, res.Code)

	//sender decode failed
	tx = gethtypes.NewTransaction(1, addr1, big.NewInt(100), 100000, big.NewInt(1), nil)
	res = _app.CheckTx(abci.RequestCheckTx{
		Tx:   append(ethutils.MustEncodeTx(tx), 0x01),
		Type: abci.CheckTxType_New,
	})
	require.Equal(t, CannotRecoverSender, res.Code)

	//tx nonce mismatch
	tx = gethtypes.NewTransaction(1, addr1, big.NewInt(100), 100000, big.NewInt(1), nil)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	res = _app.CheckTx(abci.RequestCheckTx{
		Tx:   ethutils.MustEncodeTx(tx),
		Type: abci.CheckTxType_New,
	})
	require.Equal(t, AccountNonceMismatch, res.Code)

	//gas fee not pay
	tx = gethtypes.NewTransaction(0, addr1, big.NewInt(100), 1000000000, big.NewInt(1), nil)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	res = _app.CheckTx(abci.RequestCheckTx{
		Tx:   ethutils.MustEncodeTx(tx),
		Type: abci.CheckTxType_New,
	})
	require.Equal(t, CannotPayGasFee, res.Code)

	//ok
	tx = gethtypes.NewTransaction(0, addr1, big.NewInt(100), 100000, big.NewInt(1), nil)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	res = _app.CheckTx(abci.RequestCheckTx{
		Tx:   ethutils.MustEncodeTx(tx),
		Type: abci.CheckTxType_New,
	})
	require.Equal(t, uint32(0), res.Code)
}

func TestRandomTxs(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	key3, addrTo1 := testutils.GenKeyAndAddr()
	key4, addrTo2 := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key1, key2, key3, key4)
	txLists := generateRandomTxs(100, _app.chainId, key1, key2, addrTo1, addrTo2)
	res1 := execRandomTxs(_app, txLists, addr1, addr2)
	DestroyTestApp(_app)

	_app = CreateTestApp(key1, key2, key3, key4)
	res2 := execRandomTxs(_app, txLists, addr1, addr2)
	DestroyTestApp(_app)

	require.Equal(t, res1[0], res2[0])
	require.Equal(t, res1[1], res2[1])
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

func TestStaking(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, _ := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key1, key2)
	defer DestroyTestApp(_app)

	//config test param
	staking.InitialStakingAmount = uint256.NewInt().SetUint64(1)
	staking.MinimumStakingAmount = uint256.NewInt().SetUint64(0)

	//test create validator through deliver tx
	var data [100]byte
	copy(data[:4], staking.SelectorCreateValidator[:])
	data[35] = 1 //rewardTo: address(1)
	data[67] = 2 //introduction: 2
	data[99] = 1 //pubkey: 1
	tx := gethtypes.NewTransaction(0, staking.StakingContractAddress, big.NewInt(100), 0, big.NewInt(1), data[:])
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	testutils.ExecTxInBlock(_app, 1, tx)
	time.Sleep(50 * time.Millisecond)
	ctx := _app.GetContext(RunTxMode)
	stakingAcc, info := staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	require.Equal(t, uint64(100+staking.GasOfStakingExternalOp*1 /*gasUsedFee distribute to validators*/), stakingAcc.Balance().Uint64())
	require.Equal(t, 2, len(info.Validators))
	require.True(t, bytes.Equal(addr1[:], info.Validators[1].Address[:]))
	require.Equal(t, uint8(1), info.Validators[1].Pubkey[31])
	require.Equal(t, uint64(100), uint256.NewInt().SetBytes(info.Validators[1].StakedCoins[:]).Uint64())

	//test edit validator
	copy(data[:4], staking.SelectorEditValidator[:])
	data[67] = 3 //change introduction
	tx = gethtypes.NewTransaction(1, staking.StakingContractAddress, big.NewInt(0), 0, big.NewInt(1), data[:])
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	testutils.ExecTxInBlock(_app, 3, tx)
	time.Sleep(50 * time.Millisecond)
	ctx = _app.GetContext(RunTxMode)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	require.Equal(t, 2, len(info.Validators))
	require.Equal(t, uint8(3), info.Validators[1].Introduction[31])

	// test switchEpoch
	e := &types.Epoch{
		ValMapByPubkey: make(map[[32]byte]*types.Nomination),
	}
	var pubkey [32]byte
	copy(pubkey[:], _app.testValidatorPubKey.Bytes())
	e.ValMapByPubkey[pubkey] = &types.Nomination{
		Pubkey:         pubkey,
		NominatedCount: 2,
	}
	_app.watcher.EpochChan <- e
	testutils.ExecTxInBlock(_app, 5, nil)
	ctx = _app.GetContext(RunTxMode)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	require.Equal(t, 1, len(info.Validators))
	require.Equal(t, int64(2), info.Validators[0].VotingPower)
}

func execRandomTxs(_app *App, txLists [][]*gethtypes.Transaction, from1, from2 common.Address) []uint64 {
	for i, txList := range txLists {
		_app.BeginBlock(abci.RequestBeginBlock{
			Header: tmproto.Header{
				Height:          int64(i + 1),
				ProposerAddress: _app.testValidatorPubKey.Address(),
			},
		})
		for _, tx := range txList {
			_app.DeliverTx(abci.RequestDeliverTx{
				Tx: ethutils.MustEncodeTx(tx),
			})
		}
		_app.EndBlock(abci.RequestEndBlock{})
		_app.Commit()
		_app.mtx.Lock()
		_app.mtx.Unlock()
	}
	ctx := _app.GetContext(checkTxMode)
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
			tx := gethtypes.NewTransaction(nonce, to1, big.NewInt(value), 100000, big.NewInt(1), nil)
			tx = ethutils.MustSignTx(tx, chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
			set[i*2] = tx
			nonce = uint64(rand.Int() % 200)
			value = int64(rand.Int()%100 + 1)
			tx = gethtypes.NewTransaction(nonce, to2, big.NewInt(value), 100000, big.NewInt(1), nil)
			tx = ethutils.MustSignTx(tx, chainId.ToBig(), ethutils.MustHexToPrivKey(key2))
			set[i*2+1] = tx
		}
		lists[k] = set
	}
	return lists
}
