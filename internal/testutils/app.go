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
	DefaultGasLimit    = 1000000
	DefaultInitBalance = uint64(10000000)
)

// var logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
var nopLogger = log.NewNopLogger()

type TestApp struct {
	*app.App
}

func CreateTestApp(keys ...string) *TestApp {
	return CreateTestApp0(bigutils.NewU256(DefaultInitBalance), keys...)
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
	//_app.Init(nil)
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

func (_app *TestApp) WaitMS(n int64) {
	time.Sleep(time.Duration(n) * time.Millisecond)
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

func (_app *TestApp) GetBlock(h int64) *motypes.Block {
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)
	if ctx.GetLatestHeight() != h {
		_app.WaitMS(500)
	}
	b, err := ctx.GetBlockByHeight(uint64(h))
	if err != nil {
		panic(err)
	}
	return b
}

func (_app *TestApp) GetTx(h gethcmn.Hash) (tx *motypes.Transaction) {
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)

	var err error
	for i := 0; i < 10; i++ { // retry ten times
		tx, err = ctx.GetTxByHash(h)
		if err == nil {
			return
		}
		_app.WaitMS(300)
	}
	if err != nil {
		panic(err)
	}
	return nil
}

func (_app *TestApp) GetTxsByAddr(addr gethcmn.Address) []*motypes.Transaction {
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)
	txs, err := ctx.QueryTxByAddr(addr, 1, uint32(_app.BlockNum())+1)
	if err != nil {
		panic(err)
	}
	return txs
}

func (_app *TestApp) GetToAddressCount(addr gethcmn.Address) int64 {
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)
	return ctx.GetToAddressCount(addr)
}
func (_app *TestApp) GetSep20FromAddressCount(contract, addr gethcmn.Address) int64 {
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)
	return ctx.GetSep20FromAddressCount(contract, addr)
}
func (_app *TestApp) GetSep20ToAddressCount(contract, addr gethcmn.Address) int64 {
	ctx := _app.GetRpcContext()
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
		Gas:      DefaultGasLimit,
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
	tx := ethutils.NewTx(0, &contractAddr, big.NewInt(0), DefaultGasLimit, big.NewInt(0), data)
	runner, _ := _app.RunTxForRpc(tx, sender, false)
	return runner.Status, ebp.StatusToStr(runner.Status), runner.OutData
}
func (_app *TestApp) EstimateGas(sender gethcmn.Address, tx *gethtypes.Transaction) (int, string, int64) {
	runner, estimatedGas := _app.RunTxForRpc(tx, sender, true)
	return runner.Status, ebp.StatusToStr(runner.Status), estimatedGas
}

func (_app *TestApp) DeployContractInBlock(privKey string, data []byte) (*gethtypes.Transaction, int64, gethcmn.Address) {
	tx, addr := _app.MakeAndSignTx(privKey, nil, 0, data, 0)
	h := _app.ExecTxInBlock(tx)
	contractAddr := gethcrypto.CreateAddress(addr, tx.Nonce())
	return tx, h, contractAddr
}

func (_app *TestApp) MakeAndExecTxInBlock(privKey string,
	toAddr gethcmn.Address, val int64, data []byte) (*gethtypes.Transaction, int64) {

	return _app.MakeAndExecTxInBlockWithGasPrice(privKey, toAddr, val, data, 0)
}
func (_app *TestApp) MakeAndExecTxInBlockWithGasPrice(privKey string,
	toAddr gethcmn.Address, val int64, data []byte, gasPrice int64) (*gethtypes.Transaction, int64) {

	tx, _ := _app.MakeAndSignTx(privKey, &toAddr, val, data, gasPrice)
	h := _app.ExecTxInBlock(tx)
	return tx, h
}

func (_app *TestApp) ExecTxInBlock(tx *gethtypes.Transaction) int64 {
	if tx == nil {
		return _app.ExecTxsInBlock()
	}
	return _app.ExecTxsInBlock(tx)
}

func (_app *TestApp) ExecTxsInBlock(txs ...*gethtypes.Transaction) int64 {
	height := _app.BlockNum() + 1
	_app.AddTxsInBlock(height, txs...)
	_app.WaitNextBlock(height)
	return height
}

func (_app *TestApp) AddTxsInBlock(height int64, txs ...*gethtypes.Transaction) int64 {
	_app.BeginBlock(abci.RequestBeginBlock{
		Header: tmproto.Header{
			Height:          height,
			Time:            time.Now(),
			ProposerAddress: _app.TestValidatorPubkey().Address(),
		},
	})
	for _, tx := range txs {
		_app.DeliverTx(abci.RequestDeliverTx{
			Tx: MustEncodeTx(tx),
		})
	}
	_app.EndBlock(abci.RequestEndBlock{Height: height})
	_app.Commit()
	_app.WaitLock()
	return height
}
func (_app *TestApp) WaitNextBlock(currHeight int64) {
	_app.BeginBlock(abci.RequestBeginBlock{
		Header: tmproto.Header{Height: currHeight + 1},
	})
	_app.DeliverTx(abci.RequestDeliverTx{})
	_app.EndBlock(abci.RequestEndBlock{Height: currHeight + 1})
	_app.Commit()
	_app.WaitLock()
}

func (_app *TestApp) EnsureTxSuccess(hash gethcmn.Hash) {
	tx := _app.GetTx(hash)
	if tx.Status != gethtypes.ReceiptStatusSuccessful || tx.StatusStr != "success" {
		panic("tx is failed: " + tx.StatusStr)
	}
}
func (_app *TestApp) EnsureTxFailed(hash gethcmn.Hash, msg string) {
	tx := _app.GetTx(hash)
	if tx.Status != gethtypes.ReceiptStatusFailed {
		panic("tx is success")
	}
	if tx.StatusStr != msg {
		panic("expected " + msg + ", got " + tx.StatusStr)
	}
}

func (_app *TestApp) CheckNewTxABCI(tx *gethtypes.Transaction) uint32 {
	res := _app.CheckTx(abci.RequestCheckTx{
		Tx:   MustEncodeTx(tx),
		Type: abci.CheckTxType_New,
	})
	return res.Code
}
