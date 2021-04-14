package app

import (
	"bytes"
	"encoding/json"
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"sync"
	"sync/atomic"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/holiman/uint256"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	cryptoenc "github.com/tendermint/tendermint/crypto/encoding"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingads"
	"github.com/smartbch/moeingads/store"
	"github.com/smartbch/moeingads/store/rabbit"
	"github.com/smartbch/moeingdb/modb"
	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/moeingevm/types"

	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/staking"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

var _ abcitypes.Application = (*App)(nil)

var (
	DefaultNodeHome = os.ExpandEnv("$HOME/.smartbchd")
	DefaultCLIHome  = os.ExpandEnv("$HOME/.smartbchcli")
)

type ContextMode uint8

const (
	checkTxMode     ContextMode = iota
	RunTxMode       ContextMode = iota
	RpcMode         ContextMode = iota
	HistoryOnlyMode ContextMode = iota
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

	//store
	root         *store.RootStore
	historyStore modbtypes.DB

	//refresh with block
	currHeight     int64
	checkHeight    int64
	trunk          *store.TrunkStore
	checkTrunk     *store.TrunkStore
	block          *types.Block
	blockInfo      atomic.Value // to store *types.BlockInfo
	lastCommitInfo [][]byte
	lastProposer   [20]byte
	lastGasUsed    uint64
	lastGasRefund  uint256.Int
	lastGasFee     uint256.Int

	// feeds
	chainFeed event.Feed
	logsFeed  event.Feed
	scope     event.SubscriptionScope

	//engine
	txEngine    ebp.TxExecutor
	reorderSeed int64

	//watcher
	watcher *staking.Watcher

	//util
	signer gethtypes.Signer
	logger log.Logger

	//genesis data
	currValidators []*stakingtypes.Validator
	validators     []ed25519.PubKey

	//test
	testValidatorPubKey crypto.PubKey
}

func NewApp(config *param.ChainConfig, chainId *uint256.Int, logger log.Logger,
	testValidatorPubKey crypto.PubKey) *App {

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
	app.historyStore.SetMaxEntryCount(config.RpcEthGetLogsMaxResults)

	app.trunk = app.root.GetTrunkStore().(*store.TrunkStore)
	app.checkTrunk = app.root.GetReadOnlyTrunkStore().(*store.TrunkStore)
	app.chainId = chainId
	app.signer = gethtypes.NewEIP155Signer(app.chainId.ToBig())
	app.txEngine = ebp.NewEbpTxExec(10, 100, 32, 100, app.signer)
	app.logger = logger.With("module", "app")
	//todo: lastHeight = latest previous bch mainnet 2016x blocks
	app.watcher = staking.NewWatcher(0, nil) //todo: add bch mainnet client
	go app.watcher.Run()

	ctx := app.GetContext(RunTxMode)
	prevBlk := ctx.GetCurrBlockBasicInfo()
	app.block = &types.Block{}
	if prevBlk != nil {
		app.block.Number = prevBlk.Number
		app.currHeight = app.block.Number
	}
	//init PredefinedSystemContractExecutors before tx execute
	ebp.PredefinedSystemContractExecutor = &staking.StakingContractExecutor{}
	ebp.PredefinedSystemContractExecutor.Init(ctx)

	_, stakingInfo := staking.LoadStakingAcc(*ctx)
	app.currValidators = stakingInfo.GetActiveValidators(staking.MinimumStakingAmount)
	for _, val := range app.currValidators {
		fmt.Printf("validator:%v\n", val.Address)
	}
	ctx.Close(true)
	app.testValidatorPubKey = testValidatorPubKey
	return app
}

func (app *App) Init(blk *types.Block) {
	if blk != nil {
		app.block = blk
		app.currHeight = app.block.Number
	}
	//fmt.Printf("!!!!!!get block in newapp:%v,%d\n", app.block.StateRoot, app.block.Number)
	app.root.SetHeight(app.currHeight + 1)
	if app.currHeight != 0 {
		app.reload()
	} else {
		app.txEngine.SetContext(app.GetContext(RunTxMode))
	}
}

func (app *App) reload() {
	app.txEngine.SetContext(app.GetContext(RunTxMode))
	if app.block != nil {
		app.mtx.Lock()
		bi := app.syncBlockInfo()
		app.postCommit(bi)
	}
}

func (app *App) Info(req abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{
		LastBlockHeight:  app.block.Number,
		LastBlockAppHash: app.root.GetRootHash(),
	}
}

func (app *App) SetOption(option abcitypes.RequestSetOption) abcitypes.ResponseSetOption {
	return abcitypes.ResponseSetOption{}
}

func (app *App) Query(req abcitypes.RequestQuery) abcitypes.ResponseQuery {
	return abcitypes.ResponseQuery{Code: abcitypes.CodeTypeOK}
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
		return abcitypes.ResponseCheckTx{Code: CannotRecoverSender, Info: "invalid sender"}
	}
	acc, err := ctx.CheckNonce(sender, tx.Nonce())
	if err != nil {
		return abcitypes.ResponseCheckTx{Code: AccountNonceMismatch, Info: "bad nonce: " + err.Error()}
	}
	gasPrice, _ := uint256.FromBig(tx.GasPrice())
	err = ctx.DeductTxFee(sender, acc, tx.Gas(), gasPrice)
	if err != nil {
		return abcitypes.ResponseCheckTx{Code: CannotPayGasFee, Info: "failed to deduct tx fee"}
	}
	dirty = true
	app.logger.Debug("leave check tx!")
	return abcitypes.ResponseCheckTx{Code: abcitypes.CodeTypeOK}
}

//TODO: if the last height is not 0, we must run app.txEngine.Execute(&bi) here!!
func (app *App) InitChain(req abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	app.logger.Debug("enter init chain!, id=", req.ChainId)
	app.logger.Debug("leave init chain!")

	var genesisValidators []*stakingtypes.Validator
	if len(req.AppStateBytes) != 0 {
		fmt.Printf("appstate:%s\n", req.AppStateBytes)
		genesisData := GenesisData{}
		err := json.Unmarshal(req.AppStateBytes, &genesisData)
		if err != nil {
			panic(err)
		}

		app.createGenesisAccs(genesisData.Alloc)
		genesisValidators = genesisData.Validators
	}

	if len(genesisValidators) != 0 {
		app.currValidators = genesisValidators
		ctx := app.GetContext(RunTxMode)
		stakingAcc := ctx.GetAccount(staking.StakingContractAddress)
		if stakingAcc == nil {
			panic("Cannot find staking contract")
		}
		info := stakingtypes.StakingInfo{
			CurrEpochNum:   0,
			Validators:     app.currValidators,
			PendingRewards: make([]*stakingtypes.PendingReward, len(app.currValidators)),
		}
		for i := range info.PendingRewards {
			info.PendingRewards[i] = &stakingtypes.PendingReward{}
		}
		staking.SaveStakingInfo(*ctx, stakingAcc, info)
		ctx.Close(true)
	} else /*todo: for single node test*/ {
		ctx := app.GetContext(RunTxMode)
		stakingAcc := ctx.GetAccount(staking.StakingContractAddress)
		if stakingAcc == nil {
			panic("Cannot find staking contract")
		}
		info := stakingtypes.StakingInfo{
			CurrEpochNum:   0,
			Validators:     make([]*stakingtypes.Validator, 1),
			PendingRewards: make([]*stakingtypes.PendingReward, 1),
		}
		info.Validators[0] = &stakingtypes.Validator{}
		copy(info.Validators[0].Address[:], app.testValidatorPubKey.Address())
		copy(info.Validators[0].Pubkey[:], app.testValidatorPubKey.Bytes())
		info.PendingRewards[0] = &stakingtypes.PendingReward{}
		copy(info.PendingRewards[0].Address[:], app.testValidatorPubKey.Address())
		staking.SaveStakingInfo(*ctx, stakingAcc, info)
		ctx.Close(true)
	}

	vals := make([]abcitypes.ValidatorUpdate, len(app.currValidators))
	if len(app.currValidators) != 0 {
		for i, v := range app.currValidators {
			p, _ := cryptoenc.PubKeyToProto(ed25519.PubKey(v.Pubkey[:]))
			vals[i] = abcitypes.ValidatorUpdate{
				PubKey: p,
				Power:  1,
			}
			fmt.Printf("inichain validator:%s\n", p.String())
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

func (app *App) createGenesisAccs(alloc gethcore.GenesisAlloc) {
	if len(alloc) == 0 {
		return
	}

	rbt := rabbit.NewRabbitStore(app.trunk)

	for addr, acc := range alloc {
		amt, _ := uint256.FromBig(acc.Balance)
		k := types.GetAccountKey(addr)
		v := types.ZeroAccountInfo()
		v.UpdateBalance(amt)
		rbt.Set(k, v.Bytes())
		app.logger.Info("Air drop " + amt.String() + " to " + addr.Hex())
	}

	rbt.Close()
	rbt.WriteBack()
}

func (app *App) BeginBlock(req abcitypes.RequestBeginBlock) abcitypes.ResponseBeginBlock {
	app.logger.Debug("enter begin block!")
	app.block = &types.Block{
		Number:    req.Header.Height,
		Timestamp: req.Header.Time.Unix(),
		Size:      int64(req.Size()),
	}
	copy(app.lastProposer[:], req.Header.ProposerAddress)
	for _, v := range req.LastCommitInfo.GetVotes() {
		if v.SignedLastBlock {
			app.lastCommitInfo = append(app.lastCommitInfo, v.Validator.Address) //this is validator consensus address
		}
	}
	copy(app.block.ParentHash[:], req.Header.LastBlockId.Hash)
	copy(app.block.TransactionsRoot[:], req.Header.DataHash) //TODO changed to committed tx hash
	app.reorderSeed = 0
	if len(req.Header.DataHash) >= 8 {
		app.reorderSeed = int64(binary.LittleEndian.Uint64(req.Header.DataHash[0:7]))
	}
	copy(app.block.Miner[:], req.Header.ProposerAddress)
	copy(app.block.Hash[:], req.Hash) // Just use tendermint's block hash
	copy(app.block.StateRoot[:], req.Header.AppHash[:])
	//fmt.Printf("!!!!!!app block hash:%v\n", app.block.StateRoot)
	//TODO: slash req.ByzantineValidators
	app.currHeight = req.Header.Height
	//if app.currHeight == 1 {
	//	app.root.SetHeight(app.currHeight)
	//	app.trunk = app.root.GetTrunkStore().(*store.TrunkStore)
	//	app.checkTrunk = app.root.GetReadOnlyTrunkStore().(*store.TrunkStore)
	//}
	app.logger.Debug("leave begin block!")
	return abcitypes.ResponseBeginBlock{}
}

func (app *App) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	app.logger.Debug("enter deliver tx!", "txlen", len(req.Tx))
	app.block.Size += int64(req.Size())
	tx, err := ethutils.DecodeTx(req.Tx)
	if err == nil {
		app.txEngine.CollectTx(tx)
	}
	app.logger.Debug("leave deliver tx!")
	return abcitypes.ResponseDeliverTx{Code: abcitypes.CodeTypeOK}
}

func (app *App) EndBlock(req abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	app.logger.Debug("enter end block!")
	select {
	case epoch := <-app.watcher.EpochChan:
		fmt.Printf("get new epoch in endblock, its startHeight is:%d\n", epoch.StartHeight)
		if app.block.Timestamp > epoch.EndTime+100*10*60 /*100 * 10min*/ {
			ctx := app.GetContext(RunTxMode)
			app.currValidators = staking.SwitchEpoch(ctx, epoch)
			ctx.Close(true)
		}
	default:
		//fmt.Println("no new epoch")
	}
	vals := make([]abcitypes.ValidatorUpdate, len(app.currValidators))
	if len(app.currValidators) != 0 {
		for i, v := range app.currValidators {
			p, _ := cryptoenc.PubKeyToProto(ed25519.PubKey(v.Pubkey[:]))
			vals[i] = abcitypes.ValidatorUpdate{
				PubKey: p,
				Power:  1,
			}
			fmt.Printf("endblock validator:%v\n", v.Address)
		}
	} else {
		pk, _ := cryptoenc.PubKeyToProto(app.testValidatorPubKey)
		vals = append(vals, abcitypes.ValidatorUpdate{
			PubKey: pk,
			Power:  1,
		})
	}
	app.logger.Debug("leave end block!")
	return abcitypes.ResponseEndBlock{
		ValidatorUpdates: vals,
	}
}

func (app *App) Commit() abcitypes.ResponseCommit {
	app.logger.Debug("enter commit!", "txs", app.txEngine.CollectTxsCount())
	app.mtx.Lock()

	//distribute previous block gas gee
	ctx := app.GetContext(RunTxMode)
	_, info := staking.LoadStakingAcc(*ctx)
	pubkeyMapByAddr := make(map[[20]byte][32]byte)
	for _, v := range info.Validators {
		pubkeyMapByAddr[v.Address] = v.Pubkey
	}
	voters := make([][32]byte, len(app.lastCommitInfo))
	var tmpAddr [20]byte
	for i, c := range app.lastCommitInfo {
		copy(tmpAddr[:], c)
		voters[i] = pubkeyMapByAddr[tmpAddr]
	}
	var blockReward = app.lastGasFee
	if !app.lastGasFee.IsZero() {
		if !app.lastGasRefund.IsZero() {
			err := ebp.SubSystemAccBalance(ctx, &app.lastGasRefund)
			if err != nil {
				panic(err)
			}
		}
	}
	//invariant check for fund safe
	sysB := ebp.GetSystemBalance(ctx)
	if app.txEngine.StandbyQLen() != 0 {
		if sysB.Cmp(uint256.NewInt()) <= 0 {
			panic("system account balance should have some pending gas fee")
		}
	} else {
		if sysB.Cmp(&app.lastGasFee) < 0 {
			panic("system balance not enough!")
		}
		// distribute extra balance to validators
		blockReward = *sysB
	}
	if !blockReward.IsZero() {
		err := ebp.TransferFromSystemAccToBlackHoleAcc(ctx, &blockReward)
		if err != nil {
			//todo: be careful
			panic(err)
		}
	}
	staking.DistributeFee(*ctx, &blockReward, pubkeyMapByAddr[app.lastProposer], voters)
	ctx.Close(true)

	app.txEngine.Prepare(app.reorderSeed)
	app.refresh()
	bi := app.syncBlockInfo()
	go app.postCommit(bi)
	app.logger.Debug("leave commit!")
	return abcitypes.ResponseCommit{
		Data: append([]byte{}, app.block.StateRoot[:]...),
	}
}

func (app *App) syncBlockInfo() *types.BlockInfo {
	bi := &types.BlockInfo{
		Coinbase:  app.block.Miner,
		Number:    app.block.Number,
		Timestamp: app.block.Timestamp,
		ChainId:   app.chainId.Bytes32(),
		Hash:      app.block.Hash,
	}
	app.blockInfo.Store(bi)
	return bi
}

func (app *App) postCommit(bi *types.BlockInfo) {
	app.logger.Debug("enter post commit!")
	defer app.mtx.Unlock()
	app.txEngine.Execute(bi)
	app.lastGasUsed, app.lastGasRefund, app.lastGasFee = app.txEngine.GasUsedInfo()
	app.logger.Debug("leave post commit!")
}

func (app *App) refresh() {
	//close old
	app.checkTrunk.Close(false)

	ctx := app.GetContext(RunTxMode)
	prevBlkInfo := ctx.GetCurrBlockBasicInfo()
	ctx.SetCurrBlockBasicInfo(app.block)
	//fmt.Printf("!!!!!!set block in refresh:%v,%d\n", app.block.StateRoot, app.block.Number)
	ctx.Close(true)
	app.trunk.Close(true)

	appHash := app.root.GetRootHash()
	copy(app.block.StateRoot[:], appHash)

	//jump block which prev height = 0
	if prevBlkInfo != nil {
		//use current block commit app hash as prev history block stateRoot
		prevBlkInfo.StateRoot = app.block.StateRoot
		prevBlkInfo.GasUsed = app.lastGasUsed
		blk := modbtypes.Block{
			Height: prevBlkInfo.Number,
		}
		prevBlkInfo.Transactions = make([][32]byte, len(app.txEngine.CommittedTxs()))
		for i, tx := range app.txEngine.CommittedTxs() {
			prevBlkInfo.Transactions[i] = tx.Hash
		}
		blkInfo, err := prevBlkInfo.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}
		copy(blk.BlockHash[:], prevBlkInfo.Hash[:])
		blk.BlockInfo = blkInfo
		blk.TxList = make([]modbtypes.Tx, len(app.txEngine.CommittedTxs()))
		var zeroValue [32]byte
		for i, tx := range app.txEngine.CommittedTxs() {
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
	}
	//make new
	app.lastProposer = app.block.Miner
	app.lastCommitInfo = app.lastCommitInfo[:0]
	app.root.SetHeight(app.currHeight + 1)
	app.trunk = app.root.GetTrunkStore().(*store.TrunkStore)
	app.checkTrunk = app.root.GetReadOnlyTrunkStore().(*store.TrunkStore)
	app.txEngine.SetContext(app.GetContext(RunTxMode))
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
			})
		}
	}
	return logs
}

func (app *App) ListSnapshots(snapshots abcitypes.RequestListSnapshots) abcitypes.ResponseListSnapshots {
	return abcitypes.ResponseListSnapshots{}
}

func (app *App) OfferSnapshot(snapshot abcitypes.RequestOfferSnapshot) abcitypes.ResponseOfferSnapshot {
	return abcitypes.ResponseOfferSnapshot{}
}

func (app *App) LoadSnapshotChunk(chunk abcitypes.RequestLoadSnapshotChunk) abcitypes.ResponseLoadSnapshotChunk {
	return abcitypes.ResponseLoadSnapshotChunk{}
}

func (app *App) ApplySnapshotChunk(chunk abcitypes.RequestApplySnapshotChunk) abcitypes.ResponseApplySnapshotChunk {
	return abcitypes.ResponseApplySnapshotChunk{}
}

func (app *App) Stop() {
	app.historyStore.Close()
	app.root.Close()
	app.scope.Close()
}

func (app *App) GetContext(mode ContextMode) *types.Context {
	c := types.NewContext(uint64(app.currHeight), nil, nil)
	if mode == checkTxMode {
		r := rabbit.NewRabbitStore(app.checkTrunk)
		c = c.WithRbt(&r)
	} else if mode == RunTxMode {
		r := rabbit.NewRabbitStore(app.trunk)
		c = c.WithRbt(&r)
		c = c.WithDb(app.historyStore)
	} else if mode == RpcMode {
		r := rabbit.NewReadOnlyRabbitStore(app.root)
		c = c.WithRbt(&r)
		c = c.WithDb(app.historyStore)
	} else if mode == HistoryOnlyMode {
		c = c.WithRbt(nil) // no need
		c = c.WithDb(app.historyStore)
	} else {
		panic("MoeingError: invalid context mode")
	}
	return c
}

func (app *App) RunTxForRpc(gethTx *gethtypes.Transaction, sender gethcmn.Address, estimateGas bool) (*ebp.TxRunner, int64) {
	txToRun := &types.TxToRun{}
	txToRun.FromGethTx(gethTx, sender, uint64(app.currHeight))
	ctx := app.GetContext(RpcMode)
	defer ctx.Close(false)
	runner := &ebp.TxRunner{
		Ctx: *ctx,
		Tx:  txToRun,
	}
	bi := app.blockInfo.Load().(*types.BlockInfo)
	estimateResult := ebp.RunTxForRpc(bi, estimateGas, runner)
	return runner, estimateResult
}

// SubscribeChainEvent registers a subscription of ChainEvent.
func (app *App) SubscribeChainEvent(ch chan<- types.ChainEvent) event.Subscription {
	return app.scope.Track(app.chainFeed.Subscribe(ch))
}

// SubscribeLogsEvent registers a subscription of []*types.Log.
func (app *App) SubscribeLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription {
	return app.scope.Track(app.logsFeed.Subscribe(ch))
}

func (app *App) GetLatestBlockNum() int64 {
	return app.currHeight
}

func (app *App) ChainID() *uint256.Int {
	return app.chainId
}

// used by unit tests

func (app *App) CloseTrunk() {
	app.trunk.Close(true)
}
func (app *App) CloseTxEngineContext() {
	app.txEngine.Context().Close(false)
}

func (app *App) Logger() log.Logger {
	return app.logger
}

func (app *App) WaitLock() {
	app.mtx.Lock()
	app.mtx.Unlock()
}

func (app *App) TestValidatorPubkey() crypto.PubKey {
	return app.testValidatorPubKey
}
