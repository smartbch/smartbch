package testutils

import (
	"encoding/json"
	"math/big"
	"os"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/smartbch/moeingevm/ebp"
	motypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/param"
)

const (
	adsDir  = "./testdbdata"
	modbDir = "./modbdata"
)

const (
	defaultGasLimit = 1000000
)

// var logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
var nopLogger = log.NewNopLogger()

type TestApp struct {
	*app.App
}

func CreateTestApp(keys ...string) *TestApp {
	return CreateTestApp0(bigutils.NewU256(10000000), keys...)
}

func CreateTestApp0(testInitAmt *uint256.Int, keys ...string) *TestApp {
	_ = os.RemoveAll(adsDir)
	_ = os.RemoveAll(modbDir)
	params := param.DefaultConfig()
	params.AppDataPath = adsDir
	params.ModbDataPath = modbDir
	testValidatorPubKey := ed25519.GenPrivKey().PubKey()
	_app := app.NewApp(params, bigutils.NewU256(1), nopLogger,
		testValidatorPubKey)
	_app.Init(nil)
	//_app.txEngine = ebp.NewEbpTxExec(10, 100, 1, 100, _app.signer)
	genesisData := app.GenesisData{
		Alloc: KeysToGenesisAlloc(testInitAmt, keys),
	}
	appStateBytes, _ := json.Marshal(genesisData)

	_app.InitChain(abci.RequestInitChain{AppStateBytes: appStateBytes})
	_app.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{
		ProposerAddress: testValidatorPubKey.Address(),
	}})
	_app.Commit()
	return &TestApp{_app}
}

func (_app *TestApp) Destroy() {
	_app.Stop()
	_ = os.RemoveAll(adsDir)
	_ = os.RemoveAll(modbDir)
}

func (_app *TestApp) GetNonce(addr gethcmn.Address) uint64 {
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)
	if acc := ctx.GetAccount(addr); acc != nil {
		return acc.Nonce()
	}
	return 0
}

func (_app *TestApp) GetBalance(addr gethcmn.Address) *big.Int {
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)
	if acc := ctx.GetAccount(addr); acc != nil {
		return acc.Balance().ToBig()
	}
	return nil
}

func (_app *TestApp) GetStorageAt(addr gethcmn.Address, key []byte) []byte {
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)
	if acc := ctx.GetAccount(addr); acc != nil {
		return ctx.GetStorageAt(acc.Sequence(), string(key))
	}
	return nil
}

func (_app *TestApp) GetCode(addr gethcmn.Address) []byte {
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)
	if codeInfo := ctx.GetCode(addr); codeInfo != nil {
		return codeInfo.BytecodeSlice()
	}
	return nil
}

func (_app *TestApp) GetBlock(h uint64) *motypes.Block {
	ctx := _app.GetRpcContext()
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

func (_app *TestApp) GetTx(h gethcmn.Hash) *motypes.Transaction {
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)
	tx, err := ctx.GetTxByHash(h)
	if err != nil {
		panic(err)
	}
	return tx
}

func (_app *TestApp) GetTxsByAddr(addr gethcmn.Address) []*motypes.Transaction {
	ctx := _app.GetHistoryOnlyContext()
	defer ctx.Close(false)
	txs, err := ctx.QueryTxByAddr(addr, 1, uint32(_app.BlockNum())+1)
	if err != nil {
		panic(err)
	}
	return txs
}

func (_app *TestApp) GetToAddressCount(addr gethcmn.Address) int64 {
	ctx := _app.GetHistoryOnlyContext()
	defer ctx.Close(false)
	return ctx.GetToAddressCount(addr)
}
func (_app *TestApp) GetSep20FromAddressCount(contract, addr gethcmn.Address) int64 {
	ctx := _app.GetHistoryOnlyContext()
	defer ctx.Close(false)
	return ctx.GetSep20FromAddressCount(contract, addr)
}
func (_app *TestApp) GetSep20ToAddressCount(contract, addr gethcmn.Address) int64 {
	ctx := _app.GetHistoryOnlyContext()
	defer ctx.Close(false)
	return ctx.GetSep20ToAddressCount(contract, addr)
}

func (_app *TestApp) MakeAndSignTx(hexPrivKey string,
	toAddr *gethcmn.Address, val int64, data []byte, gasPrice int64) (*gethtypes.Transaction, gethcmn.Address) {

	privKey, _, err := ethutils.HexToPrivKey(hexPrivKey)
	if err != nil {
		panic(err)
	}

	addr := ethutils.PrivKeyToAddr(privKey)
	nonce := _app.GetNonce(addr)
	chainID := _app.ChainID().ToBig()

	txData := &gethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(gasPrice),
		Gas:      defaultGasLimit,
		To:       toAddr,
		Value:    big.NewInt(val),
		Data:     data,
	}
	tx := gethtypes.NewTx(txData)
	tx, err = ethutils.SignTx(tx, chainID, privKey)
	if err != nil {
		panic(err)
	}

	return tx, addr
}

func (_app *TestApp) Call(sender, contractAddr gethcmn.Address, data []byte) (int, string, []byte) {
	tx := ethutils.NewTx(0, &contractAddr, big.NewInt(0), defaultGasLimit, big.NewInt(0), data)
	runner, _ := _app.RunTxForRpc(tx, sender, false)
	return runner.Status, ebp.StatusToStr(runner.Status), runner.OutData
}
func (_app *TestApp) EstimateGas(sender gethcmn.Address, tx *gethtypes.Transaction) (int, string, int64) {
	runner, estimatedGas := _app.RunTxForRpc(tx, sender, true)
	return runner.Status, ebp.StatusToStr(runner.Status), estimatedGas
}

func (_app *TestApp) DeployContractInBlock(height int64, privKey string, data []byte) (*gethtypes.Transaction, gethcmn.Address) {
	tx, addr := _app.MakeAndSignTx(privKey, nil, 0, data, 0)
	_app.ExecTxInBlock(height, tx)
	contractAddr := gethcrypto.CreateAddress(addr, tx.Nonce())
	return tx, contractAddr
}

func (_app *TestApp) MakeAndExecTxInBlock(height int64, privKey string,
	toAddr gethcmn.Address, val int64, data []byte) *gethtypes.Transaction {

	return _app.MakeAndExecTxInBlockWithGasPrice(height, privKey, toAddr, val, data, 0)
}
func (_app *TestApp) MakeAndExecTxInBlockWithGasPrice(height int64, privKey string,
	toAddr gethcmn.Address, val int64, data []byte, gasPrice int64) *gethtypes.Transaction {

	tx, _ := _app.MakeAndSignTx(privKey, &toAddr, val, data, gasPrice)
	_app.ExecTxInBlock(height, tx)
	return tx
}

func (_app *TestApp) ExecTxInBlock(height int64, tx *gethtypes.Transaction) {
	_app.BeginBlock(abci.RequestBeginBlock{
		Header: tmproto.Header{
			Height:          height,
			Time:            time.Now(),
			ProposerAddress: _app.TestValidatorPubkey().Address(),
		},
	})
	if tx != nil {
		_app.DeliverTx(abci.RequestDeliverTx{
			Tx: MustEncodeTx(tx),
		})
	}
	_app.EndBlock(abci.RequestEndBlock{Height: height})
	_app.Commit()
	_app.WaitLock()

	_app.BeginBlock(abci.RequestBeginBlock{
		Header: tmproto.Header{Height: height + 1},
	})
	_app.DeliverTx(abci.RequestDeliverTx{})
	_app.EndBlock(abci.RequestEndBlock{Height: height + 1})
	_app.Commit()
	_app.WaitLock()
}
