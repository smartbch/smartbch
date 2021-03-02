package app

import (
	"os"

	"github.com/holiman/uint256"
	modbtypes "github.com/moeing-chain/MoeingDB/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

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
