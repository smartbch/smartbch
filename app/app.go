package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/holiman/uint256"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	cryptoenc "github.com/tendermint/tendermint/crypto/encoding"
	"github.com/tendermint/tendermint/libs/log"

	moeingads "github.com/moeing-chain/MoeingADS"
	"github.com/moeing-chain/MoeingADS/store"
	"github.com/moeing-chain/MoeingADS/store/rabbit"
	"github.com/moeing-chain/MoeingDB/modb"
	modbtypes "github.com/moeing-chain/MoeingDB/types"
	"github.com/moeing-chain/MoeingEVM/ebp"
	"github.com/moeing-chain/MoeingEVM/types"

	"github.com/moeing-chain/moeing-chain/internal/ethutils"
	"github.com/moeing-chain/moeing-chain/param"
)

var _ abcitypes.Application = (*App)(nil)

var (
	DefaultNodeHome = os.ExpandEnv("$HOME/.moeingd")
	DefaultCLIHome  = os.ExpandEnv("$HOME/.moeingcli")
)

type ContextMode uint8

const (
	checkTxMode ContextMode = iota
	RunTxMode   ContextMode = iota
	RpcMode     ContextMode = iota
)

const (
	CannotDecodeTx       uint32 = 101
	CannotRecoverSender  uint32 = 102
	SenderNotFound       uint32 = 103
	AccountNonceMismatch uint32 = 104
	CannotPayGasFee      uint32 = 105
)

type App struct {
	mtx sync.Mutex

	//config
	chainId *uint256.Int
	Config  *param.ChainConfig

	//store
	root         *store.RootStore
	historyStore modbtypes.DB

	//refresh with block
	currHeight  int64
	checkHeight int64
	Trunk       *store.TrunkStore
	CheckTrunk  *store.TrunkStore
	block       *types.Block

	// feeds
	chainFeed event.Feed
	logsFeed  event.Feed
	scope     event.SubscriptionScope

	//engine
	TxEngine ebp.TxExecutor

	//util
	signer gethtypes.Signer
	logger log.Logger

	//genesis data
	Validators []ed25519.PubKey

	//test
	testValidatorPubKey crypto.PubKey
	testKeys            []string
	testInitAmt         *uint256.Int
}

func NewApp(config *param.ChainConfig, chainId *uint256.Int, logger log.Logger,
	testValidatorPubKey crypto.PubKey, testKeys []string, testInitAmt *uint256.Int) *App {

	app := &App{}
	first := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	last := []byte{255, 255, 255, 255, 255, 255, 255, 255}
	mads, err := moeingads.NewMoeingADS(config.AppDataPath, false, [][]byte{first, last})
	if err != nil {
		panic(err)
	}
	app.root = store.NewRootStore(mads, nil)

	//app.historyStore = &modb.MockMoDB{}
	modbDir := config.ModbDataPath
	if _, err := os.Stat(modbDir); os.IsNotExist(err) {
		_ = os.MkdirAll(path.Join(modbDir, "data"), 0700)
		app.historyStore = modb.CreateEmptyMoDB(modbDir, [8]byte{1, 2, 3, 4, 5, 6, 7, 8})
	} else {
		app.historyStore = modb.NewMoDB(modbDir)
	}

	app.Trunk = app.root.GetTrunkStore().(*store.TrunkStore)
	app.CheckTrunk = app.root.GetReadOnlyTrunkStore().(*store.TrunkStore)
	app.chainId = chainId
	app.signer = gethtypes.NewEIP155Signer(app.chainId.ToBig())
	app.TxEngine = ebp.NewEbpTxExec(10, 100, 32, 100, app.signer)
	app.Config = config
	app.logger = logger.With("module", "app")

	ctx := app.GetContext(RunTxMode)
	app.block = ctx.GetCurrBlockBasicInfo()
	app.Validators = ctx.GetCurrValidators()
	for _, val := range app.Validators {
		fmt.Printf("validator:%s\n", val.Address().String())
	}
	ctx.Close(false)
	if app.block != nil {
		app.currHeight = app.block.Number
	} else {
		app.block = &types.Block{
			Number: 0,
			Hash:   [32]byte{},
		}
	}
	fmt.Printf("!!!!!!get block in newapp:%v,%d\n", app.block.StateRoot, app.block.Number)
	app.root.SetHeight(app.currHeight + 1)
	if app.currHeight != 0 {
		app.reload()
	} else {
		app.TxEngine.SetContext(app.GetContext(RunTxMode))
	}

	app.testValidatorPubKey = testValidatorPubKey
	app.testKeys = testKeys
	app.testInitAmt = testInitAmt
	return app
}

func (app *App) Info(req abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{
		LastBlockHeight:  app.block.Number,
		LastBlockAppHash: app.root.GetRootHash(),
	}
}

func (app *App) CurrentBlock() *types.Block {
	return app.block
}

func (app *App) SetCurrentBlock(b *types.Block) {
	app.block = b
}

func (app *App) BeginBlock(req abcitypes.RequestBeginBlock) abcitypes.ResponseBeginBlock {
	app.logger.Debug("enter begin block!")
	app.block = &types.Block{
		Number:    req.Header.Height,
		Timestamp: req.Header.Time.Unix(),
		Size:      int64(req.Size()),
	}
	copy(app.block.ParentHash[:], req.Header.LastBlockId.Hash)
	copy(app.block.TransactionsRoot[:], req.Header.DataHash) //TODO changed to committed tx hash
	copy(app.block.Miner[:], req.Header.ProposerAddress)
	copy(app.block.Hash[:], req.Hash) // Just use tendermint's block hash
	copy(app.block.StateRoot[:], req.Header.AppHash[:])
	fmt.Printf("!!!!!!app block hash:%v\n", app.block.StateRoot)
	//TODO: slash req.ByzantineValidators
	app.currHeight = req.Header.Height
	//if app.currHeight == 1 {
	//	app.root.SetHeight(app.currHeight)
	//	app.Trunk = app.root.GetTrunkStore().(*store.TrunkStore)
	//	app.CheckTrunk = app.root.GetReadOnlyTrunkStore().(*store.TrunkStore)
	//}
	app.logger.Debug("leave begin block!")
	return abcitypes.ResponseBeginBlock{}
}

func (app *App) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	app.logger.Debug("enter deliver tx!", "txlen", len(req.Tx))
	app.block.Size += int64(req.Size())
	tx, err := ethutils.DecodeTx(req.Tx)
	if err == nil {
		app.TxEngine.CollectTx(tx)
	}
	app.logger.Debug("leave deliver tx!")
	return abcitypes.ResponseDeliverTx{Code: abcitypes.CodeTypeOK}
}

func (app *App) EndBlock(req abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	app.logger.Debug("enter end block!")
	app.logger.Debug("leave end block!")

	vals := make([]abcitypes.ValidatorUpdate, len(app.Validators))
	if len(app.Validators) != 0 {
		for i, v := range app.Validators {
			p, _ := cryptoenc.PubKeyToProto(v)
			vals[i] = abcitypes.ValidatorUpdate{
				PubKey: p,
				Power:  1,
			}
			fmt.Printf("endblock validator:%s\n", v.Address().String())
		}
	} else {
		pk, _ := cryptoenc.PubKeyToProto(app.testValidatorPubKey)
		vals = append(vals, abcitypes.ValidatorUpdate{
			PubKey: pk,
			Power:  1,
		})
	}
	return abcitypes.ResponseEndBlock{
		ValidatorUpdates: vals,
	}
}

func (app *App) postCommit() {
	app.logger.Debug("enter post commit!")
	defer app.mtx.Unlock()
	bi := types.BlockInfo{
		Number:    app.block.Number,
		Timestamp: app.block.Timestamp,
	}
	copy(bi.Hash[:], app.block.Hash[:])
	copy(bi.Coinbase[:], app.block.Miner[:])
	//app.chainId.FillBytes(bi.ChainId[:])
	app.TxEngine.Execute(&bi)
	app.logger.Debug("leave post commit!")
}

func (app *App) Commit() abcitypes.ResponseCommit {
	app.logger.Debug("enter commit!", "txs", app.TxEngine.CollectTxsCount())
	app.mtx.Lock()
	app.TxEngine.Prepare(app.block)
	app.Refresh()
	go app.postCommit()
	app.logger.Debug("leave commit!")
	return abcitypes.ResponseCommit{
		Data: append([]byte{}, app.block.StateRoot[:]...),
	}
}

func (app *App) Refresh() {
	//close old
	app.CheckTrunk.Close(false)

	ctx := app.GetContext(RunTxMode)
	ctx.SetCurrBlockBasicInfo(app.block)
	fmt.Printf("!!!!!!set block in refresh:%v,%d\n", app.block.StateRoot, app.block.Number)
	ctx.SetCurrValidators(app.Validators)
	ctx.Close(true)
	app.Trunk.Close(true)

	appHash := app.root.GetRootHash()
	copy(app.block.StateRoot[:], appHash)

	//todo: store current block here in root.db directly
	//app.root.Set([]byte{types.CURR_BLOCK_KEY}, app.block.SerializeBasicInfo())
	//ctx := app.GetContext(RunTxMode)
	//ctx.SetCurrBlockBasicInfo(app.block)
	//fmt.Printf("!!!!!!set block in refresh:%v,%d\n", app.block.StateRoot, app.block.Number)
	//ctx.Close(true)
	//app.Trunk.Close(true)

	//write back current commit infos to history store
	blk := modbtypes.Block{
		Height: app.block.Number,
	}
	app.block.Transactions = make([][32]byte, len(app.TxEngine.CommittedTxs()))
	for i, tx := range app.TxEngine.CommittedTxs() {
		app.block.Transactions[i] = tx.Hash
	}
	blkInfo, err := app.block.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	copy(blk.BlockHash[:], app.block.Hash[:])
	blk.BlockInfo = blkInfo
	blk.TxList = make([]modbtypes.Tx, len(app.TxEngine.CommittedTxs()))
	var zeroValue [32]byte
	for i, tx := range app.TxEngine.CommittedTxs() {
		t := modbtypes.Tx{}
		copy(t.HashId[:], tx.Hash[:])
		copy(t.SrcAddr[:], tx.From[:])
		if !bytes.Equal(tx.Value[:], zeroValue[:]) {
			copy(t.DstAddr[:], tx.To[:])
		}
		txContent, err := tx.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}
		t.Content = txContent
		t.LogList = make([]modbtypes.Log, len(tx.Logs))
		for j, l := range tx.Logs {
			copy(t.LogList[j].Address[:], l.Address[:])
			if len(l.Topics) != 0 {
				t.LogList[j].Topics = make([][32]byte, len(l.Topics))
			}
			for k, topic := range l.Topics {
				copy(t.LogList[j].Topics[k][:], topic[:])
			}
		}
		blk.TxList[i] = t
	}
	app.historyStore.AddBlock(&blk, -1)
	app.publishNewBlock(&blk)

	//make new
	app.root.SetHeight(app.currHeight + 1)
	app.Trunk = app.root.GetTrunkStore().(*store.TrunkStore)
	app.CheckTrunk = app.root.GetReadOnlyTrunkStore().(*store.TrunkStore)
	app.TxEngine.SetContext(app.GetContext(RunTxMode))
}

func (app *App) reload() {
	app.TxEngine.SetContext(app.GetContext(RunTxMode))
	if app.block != nil {
		app.mtx.Lock()
		app.postCommit()
	}
}

func (app *App) publishNewBlock(mdbBlock *modbtypes.Block) {
	if mdbBlock == nil {
		return
	}
	chainEvent := types.ChainEvent{
		Hash: mdbBlock.BlockHash,
		BlockHeader: &types.Header{
			Number:    uint64(mdbBlock.Height),
			BlockHash: mdbBlock.BlockHash,
			// TODO: fill more fields
		},
		Block: mdbBlock,
		Logs:  collectAllGethLogs(mdbBlock),
	}
	app.chainFeed.Send(chainEvent)
	if len(chainEvent.Logs) > 0 {
		app.logsFeed.Send(chainEvent.Logs)
	}
}

func collectAllGethLogs(mdbBlock *modbtypes.Block) []*gethtypes.Log {
	logs := make([]*gethtypes.Log, 0, 8)
	for _, mdbTx := range mdbBlock.TxList {
		for _, mdbLog := range mdbTx.LogList {
			logs = append(logs, &gethtypes.Log{
				Address: mdbLog.Address,
				Topics:  types.ToGethHashes(mdbLog.Topics),
				// TODO: fill more fields
			})
		}
	}
	return logs
}

func (app *App) CheckTx(req abcitypes.RequestCheckTx) abcitypes.ResponseCheckTx {
	app.logger.Debug("enter check tx!")
	ctx := app.GetContext(checkTxMode)
	dirty := false
	defer func(dirtyPtr *bool) {
		ctx.Close(*dirtyPtr)
	}(&dirty)

	tx := &gethtypes.Transaction{}
	err := tx.DecodeRLP(rlp.NewStream(bytes.NewReader(req.Tx), 0))
	if err != nil {
		return abcitypes.ResponseCheckTx{Code: CannotDecodeTx}
	}
	sender, err := app.signer.Sender(tx)
	if err != nil {
		return abcitypes.ResponseCheckTx{Code: CannotRecoverSender, Info: "bad sig"}
	}
	acc, err := ctx.CheckNonce(sender, tx.Nonce())
	if err != nil {
		return abcitypes.ResponseCheckTx{Code: AccountNonceMismatch, Info: "bad nonce"}
	}
	gasprice, _ := uint256.FromBig(tx.GasPrice())
	err = ctx.DeductTxFee(sender, acc, tx.Gas(), gasprice)
	if err != nil {
		return abcitypes.ResponseCheckTx{Code: CannotPayGasFee, Info: "failed to deduct tx fee"}
	}
	dirty = true
	app.logger.Debug("leave check tx!")
	return abcitypes.ResponseCheckTx{Code: abcitypes.CodeTypeOK}
}

func (app *App) Query(req abcitypes.RequestQuery) abcitypes.ResponseQuery {
	return abcitypes.ResponseQuery{Code: abcitypes.CodeTypeOK}
}

//TODO: if the last height is not 0, we must run app.TxEngine.Execute(&bi) here!!
func (app *App) InitChain(req abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	app.logger.Debug("enter init chain!, id=", req.ChainId)
	app.createTestAccs()
	app.logger.Debug("leave init chain!")

	if len(req.AppStateBytes) != 0 {
		fmt.Printf("appstate:%s\n", req.AppStateBytes)
		err := json.Unmarshal(req.AppStateBytes, &app.Validators)
		if err != nil {
			panic(err)
		}
	}

	vals := make([]abcitypes.ValidatorUpdate, len(app.Validators))
	if len(app.Validators) != 0 {
		for i, v := range app.Validators {
			p, _ := cryptoenc.PubKeyToProto(v)
			vals[i] = abcitypes.ValidatorUpdate{
				PubKey: p,
				Power:  1,
			}
			fmt.Printf("inichain validator:%s\n", v.Address().String())
		}
	} else {
		pk, _ := cryptoenc.PubKeyToProto(app.testValidatorPubKey)
		vals = append(vals, abcitypes.ValidatorUpdate{
			PubKey: pk,
			Power:  1,
		})
	}
	return abcitypes.ResponseInitChain{
		Validators: vals,
	}
}

func (app *App) createTestAccs() {
	if len(app.testKeys) == 0 {
		return
	}

	rbt := rabbit.NewRabbitStore(app.Trunk)
	amt := app.testInitAmt

	for _, keyHex := range app.testKeys {
		if key, err := ethutils.HexToPrivKey(keyHex); err == nil {
			addr := ethutils.PrivKeyToAddr(key)
			k := types.GetAccountKey(addr)
			v := types.ZeroAccountInfo()
			v.UpdateBalance(amt)
			rbt.Set(k, v.Bytes())

			app.logger.Info("Air drop " + amt.String() + " to " + addr.Hex())
		}
	}

	rbt.Close()
	rbt.WriteBack()
}

func (app *App) SetOption(option abcitypes.RequestSetOption) abcitypes.ResponseSetOption {
	return abcitypes.ResponseSetOption{}
}

func (app *App) ChainID() *uint256.Int {
	return app.chainId
}

func (app *App) Stop() {
	app.historyStore.Close()
	app.root.Close()
	app.scope.Close()
}

func (app *App) ListSnapshots(snapshots abcitypes.RequestListSnapshots) abcitypes.ResponseListSnapshots {
	panic("implement me")
}

func (app *App) OfferSnapshot(snapshot abcitypes.RequestOfferSnapshot) abcitypes.ResponseOfferSnapshot {
	panic("implement me")
}

func (app *App) LoadSnapshotChunk(chunk abcitypes.RequestLoadSnapshotChunk) abcitypes.ResponseLoadSnapshotChunk {
	panic("implement me")
}

func (app *App) ApplySnapshotChunk(chunk abcitypes.RequestApplySnapshotChunk) abcitypes.ResponseApplySnapshotChunk {
	panic("implement me")
}

func (app *App) GetContext(mode ContextMode) *types.Context {
	c := types.NewContext(uint64(app.currHeight), nil, nil)
	if mode == checkTxMode {
		r := rabbit.NewRabbitStore(app.CheckTrunk)
		c = c.WithRbt(&r)
	} else if mode == RunTxMode {
		r := rabbit.NewRabbitStore(app.Trunk)
		c = c.WithRbt(&r)
		c = c.WithDb(app.historyStore)
	} else if mode == RpcMode {
		r := rabbit.NewReadOnlyRabbitStore(app.root)
		c = c.WithRbt(&r)
		c = c.WithDb(app.historyStore)
	} else {
		panic("MoeingError: invalid context mode")
	}
	return c
}

func (app *App) GetLatestBlockNum() int64 {
	return app.currHeight
}

// SubscribeChainEvent registers a subscription of ChainEvent.
func (app *App) SubscribeChainEvent(ch chan<- types.ChainEvent) event.Subscription {
	return app.scope.Track(app.chainFeed.Subscribe(ch))
}

// SubscribeLogsEvent registers a subscription of []*types.Log.
func (app *App) SubscribeLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription {
	return app.scope.Track(app.logsFeed.Subscribe(ch))
}

func (app *App) Logger() log.Logger {
	return app.logger
}

func (app *App) RunTxForRpc(gethTx *gethtypes.Transaction, sender common.Address, estimateGas bool) (*ebp.TxRunner, int64) {
	txToRun := &types.TxToRun{}
	txToRun.FromGethTx(gethTx, sender, uint64(app.currHeight))
	ctx := app.GetContext(RpcMode)
	defer ctx.Close(false)
	runner := &ebp.TxRunner{
		Ctx: *ctx,
		Tx:  txToRun,
	}
	bi := &types.BlockInfo{
		Coinbase:  app.block.Miner,
		Number:    app.block.Number,
		Timestamp: app.block.Timestamp,
	}
	estimateResult := ebp.RunTxForRpc(bi, estimateGas, runner)
	return runner, estimateResult
}

func (app *App) WaitLock() {
	app.mtx.Lock()
	app.mtx.Unlock()
}

func (app *App) TestKeys() []string {
	return app.testKeys
}

//func (app *App) EstimateGas(tx *types.BasicTx) int64 {
//}
//
//func (app *App) GasPrice() *uint256.Int { //TODO
//}
//
//func (app *App) GetBalance(addr common.Address) *uint256.Int {
//}
//
//func (app *App) GetCode() []byte {
//}
//
//func (app *App) GetStorageAt(addr common.Address, key []byte) []byte {
//}
//
//func (app *App) GetNonce(addr common.Address) int64 {
//}
//
//func (app *App) GetNonceInCheckTx(addr common.Address) int64 {
//}
//
