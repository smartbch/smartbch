package app

import (
	"math/big"
	"os"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"

	modbtypes "github.com/moeing-chain/MoeingDB/types"
	"github.com/moeing-chain/MoeingEVM/ebp"
	motypes "github.com/moeing-chain/MoeingEVM/types"
	"github.com/moeing-chain/moeing-chain/internal/bigutils"
	"github.com/moeing-chain/moeing-chain/param"
)

const (
	adsDir  = "./testdbdata"
	modbDir = "./modbdata"
)

// var logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
var nopLogger = log.NewNopLogger()

func DestroyTestApp(_app *App) {
	_app.Stop()
	_ = os.RemoveAll(adsDir)
	_ = os.RemoveAll(modbDir)
}

func CreateTestApp(keys ...string) *App {
	return CreateTestApp0(bigutils.NewU256(10000000), keys...)
}

func CreateTestApp0(testInitAmt *uint256.Int, keys ...string) *App {
	_ = os.RemoveAll(adsDir)
	_ = os.RemoveAll(modbDir)
	params := param.DefaultConfig()
	params.AppDataPath = adsDir
	params.ModbDataPath = modbDir
	testValidatorPubKey := ed25519.GenPrivKey().PubKey()
	_app := NewApp(params, bigutils.NewU256(1), nopLogger,
		testValidatorPubKey, keys, testInitAmt)
	_app.Init(nil)
	//_app.txEngine = ebp.NewEbpTxExec(10, 100, 1, 100, _app.signer)
	_app.InitChain(abci.RequestInitChain{})
	_app.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{}})
	_app.Commit()
	return _app
}

func AddBlockFotTest(_app *App, mdbBlock *modbtypes.Block) {
	_app.historyStore.AddBlock(mdbBlock, -1)
	_app.historyStore.AddBlock(nil, -1) // To Flush
	_app.publishNewBlock(mdbBlock)
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

func getBlock(_app *App, h uint64) *motypes.Block {
	ctx := _app.GetContext(RpcMode)
	defer ctx.Close(false)
	if ctx.GetLatestHeight() != int64(h) {
		time.Sleep(500 * time.Millisecond)
	}
	b, err := ctx.GetBlockByHeight(h)
	if err != nil {
		panic(err)
	}
	return b
}

func getTx(_app *App, h common.Hash) *motypes.Transaction {
	ctx := _app.GetContext(RpcMode)
	defer ctx.Close(false)
	tx, err := ctx.GetTxByHash(h)
	if err != nil {
		panic(err)
	}
	return tx
}

func call(_app *App, sender common.Address, tx *gethtypes.Transaction) (int, string, []byte) {
	runner, _ := _app.RunTxForRpc(tx, sender, false)
	return runner.Status, ebp.StatusToStr(runner.Status), runner.OutData
}
