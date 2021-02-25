package app

import (
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/moeing-chain/MoeingEVM/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/moeing-chain/moeing-chain/internal/ethutils"
	"github.com/moeing-chain/moeing-chain/internal/testutils"
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
	require.Equal(t, uint64(10000000-100-21000), getBalance(_app, addr1).Uint64())
	require.Equal(t, uint64(10000000+100), getBalance(_app, addr2).Uint64())

	n := _app.GetLatestBlockNum()
	require.Equal(t, int64(2), n)

	ctx := _app.GetContext(RpcMode)
	defer ctx.Close(false)

	blk1 := getBlock(_app, 1)
	require.Equal(t, int64(1), blk1.Number)
	require.Len(t, blk1.Transactions, 0)

	blk2 := getBlock(_app, 2)
	require.Equal(t, int64(2), blk2.Number)
	require.Len(t, blk2.Transactions, 1)

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

	tx := gethtypes.NewTransaction(0, addr2, big.NewInt(10000001), 100000, big.NewInt(1), nil)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	testutils.ExecTxInBlock(_app, 1, tx)

	require.Equal(t, uint64(10000000-21000), getBalance(_app, addr1).Uint64())
	require.Equal(t, uint64(10000000), getBalance(_app, addr2).Uint64())

	// check tx status
	moeTx := getTx(_app, tx.Hash())
	require.Equal(t, [32]byte(tx.Hash()), moeTx.Hash)
	require.Equal(t, uint64(0), moeTx.Status)
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

	blk2 := getBlock(_app, 2)
	require.Equal(t, int64(2), blk2.Number)
	require.Len(t, blk2.Transactions, 1)

	testutils.ExecTxInBlock(_app, 3, nil)
	time.Sleep(50 * time.Millisecond)

	blk4 := getBlock(_app, 4)
	require.Equal(t, int64(4), blk4.Number)
	require.Len(t, blk4.Transactions, 0)
}

func TestDeployContract(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key)
	defer DestroyTestApp(_app)

	creationBytecode := testutils.HexToBytes(`
608060405234801561001057600080fd5b5060cc8061001f6000396000f3fe60
80604052348015600f57600080fd5b506004361060325760003560e01c806361
bc221a1460375780636299a6ef146053575b600080fd5b603d607e565b604051
8082815260200191505060405180910390f35b607c6004803603602081101560
6757600080fd5b81019080803590602001909291905050506084565b005b6000
5481565b8060008082825401925050819055505056fea2646970667358221220
37865cfcfd438966956583c78d31220c05c0f1ebfd116aced883214fcb1096c6
64736f6c634300060c0033
`)
	deployedBytecode := testutils.HexToBytes(`
6080604052348015600f57600080fd5b506004361060325760003560e01c8063
61bc221a1460375780636299a6ef146053575b600080fd5b603d607e565b6040
518082815260200191505060405180910390f35b607c60048036036020811015
606757600080fd5b81019080803590602001909291905050506084565b005b60
005481565b8060008082825401925050819055505056fea26469706673582212
2037865cfcfd438966956583c78d31220c05c0f1ebfd116aced883214fcb1096
c664736f6c634300060c0033
`)
	tx := gethtypes.NewContractCreation(0, big.NewInt(0), 100000, big.NewInt(1), creationBytecode)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key))

	testutils.ExecTxInBlock(_app, 1, tx)
	contractAddr := gethcrypto.CreateAddress(addr, tx.Nonce())
	code := getCode(_app, contractAddr)
	require.Equal(t, deployedBytecode, code)
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

func execRandomTxs(_app *App, txLists [][]*gethtypes.Transaction, from1, from2 common.Address) []uint64 {
	for i, txList := range txLists {
		_app.BeginBlock(abci.RequestBeginBlock{
			Header: tmproto.Header{Height: int64(i + 1)},
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

// helpers

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

func getBalance(_app *App, addr common.Address) *big.Int {
	ctx := _app.GetContext(RpcMode)
	defer ctx.Close(false)
	b, err := ctx.GetBalance(addr, -1)
	if err != nil {
		panic(err)
	}
	return b.ToBig()
}

func getCode(_app *App, addr common.Address) []byte {
	ctx := _app.GetContext(RpcMode)
	defer ctx.Close(false)
	codeInfo := ctx.GetCode(addr)
	if codeInfo == nil {
		return nil
	}
	return codeInfo.BytecodeSlice()
}

func getBlock(_app *App, h uint64) *types.Block {
	ctx := _app.GetContext(RpcMode)
	defer ctx.Close(false)
	b, err := ctx.GetBlockByHeight(h)
	if err != nil {
		panic(err)
	}
	return b
}

func getTx(_app *App, h common.Hash) *types.Transaction {
	ctx := _app.GetContext(RpcMode)
	defer ctx.Close(false)
	tx, err := ctx.GetTxByHash(h)
	if err != nil {
		panic(err)
	}
	return tx
}
