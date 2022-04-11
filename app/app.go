package app

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path"
	"sync/atomic"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/smartbch/moeingads"
	"github.com/smartbch/moeingads/store"
	"github.com/smartbch/moeingads/store/rabbit"
	"github.com/smartbch/moeingdb/modb"
	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/moeingevm/types"

	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/staking"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

type App struct {
	Config  *param.ChainConfig
	ChainId *uint256.Int

	//store
	Mads         *moeingads.MoeingADS
	Root         *store.RootStore
	Trunk        *store.TrunkStore
	HistoryStore modbtypes.DB

	currHeight    int64
	block         *types.Block
	blockInfo     atomic.Value // to store *types.BlockInfo
	StateProducer IStateProducer

	Logger log.Logger
}

func NewApp(config *param.ChainConfig, logger log.Logger) *App {
	app := &App{}

	app.Config = config
	app.ChainId = uint256.NewInt(param.ChainID)
	app.Logger = logger.With("module", "app")

	app.Root, app.Mads = CreateRootStore(config)
	app.HistoryStore = CreateHistoryStore(config, app.Logger.With("module", "modb"))
	app.currHeight = app.HistoryStore.GetLatestHeight()
	app.Logger.Debug("storeHeight", app.currHeight)
	if app.currHeight == 0 {
		app.InitGenesisState()
	}
	app.StateProducer = NewRpcClient(config.AppConfig.SmartBchRPCUrl, "", "", "application/json", app.Logger.With("module", "client"))
	go app.Run(app.currHeight)
	return app
}

func CreateRootStore(config *param.ChainConfig) (*store.RootStore, *moeingads.MoeingADS) {
	first := [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
	last := [8]byte{255, 255, 255, 255, 255, 255, 255, 255}
	mads, err := moeingads.NewMoeingADS(config.AppConfig.AppDataPath, config.AppConfig.ArchiveMode,
		[][]byte{first[:], last[:]})
	if err != nil {
		panic(err)
	}
	root := store.NewRootStore(mads, func(k []byte) bool {
		return len(k) >= 1 && k[0] > (128+64) //only cache the standby queue
	})
	return root, mads
}

func CreateHistoryStore(config *param.ChainConfig, logger log.Logger) (historyStore modbtypes.DB) {
	modbDir := config.AppConfig.ModbDataPath
	if config.AppConfig.UseLiteDB {
		historyStore = modb.NewLiteDB(modbDir)
	} else {
		if _, err := os.Stat(modbDir); os.IsNotExist(err) {
			_ = os.MkdirAll(path.Join(modbDir, "data"), 0700)
			var seed [8]byte // use current time as moeingdb's hash seed
			binary.LittleEndian.PutUint64(seed[:], uint64(time.Now().UnixNano()))
			historyStore = modb.CreateEmptyMoDB(modbDir, seed, logger)
		} else {
			historyStore = modb.NewMoDB(modbDir, logger)
		}
		historyStore.SetMaxEntryCount(config.AppConfig.RpcEthGetLogsMaxResults)
	}
	return
}

func (app *App) InitGenesisState() {
	genFile := app.Config.AppConfig.GenesisFilePath
	genDoc := &tmtypes.GenesisDoc{}
	if _, err := os.Stat(genFile); err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
	} else {
		genDoc, err = tmtypes.GenesisDocFromFile(genFile)
		if err != nil {
			panic(err)
		}
	}
	genesisData := GenesisData{}
	err := json.Unmarshal(genDoc.AppState, &genesisData)
	if err != nil {
		panic(err)
	}
	app.Logger.Debug("InitGenesisState", "genesis data", string(genDoc.AppState))
	app.Trunk = app.Root.GetTrunkStore(app.Config.AppConfig.TrunkCacheSize).(*store.TrunkStore)
	app.Root.SetHeight(0)
	app.createGenesisAccounts(genesisData.Alloc)
	genesisValidators := genesisData.stakingValidators()
	if len(genesisValidators) == 0 {
		panic("no genesis validator in genesis.json")
	}
	//store all genesis validators even if it is inactive
	ctx := app.GetRunTxContext()
	exe := staking.NewStakingContractExecutor(app.Logger.With("module", "staking"))
	exe.Init(ctx)
	staking.AddGenesisValidatorsIntoStakingInfo(ctx, genesisValidators)
	ctx.Close(true)
	app.Trunk.Close(true)
}

func (app *App) createGenesisAccounts(alloc gethcore.GenesisAlloc) {
	if len(alloc) == 0 {
		return
	}

	rbt := rabbit.NewRabbitStore(app.Trunk)

	app.Logger.Info("air drop", "accounts", len(alloc))
	for addr, acc := range alloc {
		amt, _ := uint256.FromBig(acc.Balance)
		k := types.GetAccountKey(addr)
		v := types.ZeroAccountInfo()
		v.UpdateBalance(amt)
		rbt.Set(k, v.Bytes())
		//app.Logger.Info("Air drop " + amt.String() + " to " + addr.Hex())
	}

	rbt.Close()
	rbt.WriteBack()
}

func (app *App) Run(storeHeight int64) {
	//1. fetch blocks until catch up leader.
	latestHeight := app.catchupLeader(storeHeight)
	// Run 2 times to catch blocks mint amount 1st catchupLeader running.
	latestHeight = app.catchupLeader(latestHeight)
	//2. keep sync with leader.
	for {
		latestHeight = app.updateState(latestHeight + 1)
		time.Sleep(100 * time.Millisecond)
	}
}

func (app *App) updateState(height int64) int64 {
	blk, err := app.StateProducer.GetSyncBlock(uint64(height))
	if err != nil {
		app.Logger.Debug("updateState failed", "wantedHeight", height, "error", err)
		return height - 1
	}
	app.Root.SetHeight(height)
	store.SyncUpdateTo(blk.UpdateOfADS, app.Root)
	app.HistoryStore.AddBlock(&blk.Block, -1, blk.Txid2sigMap)
	app.currHeight = blk.Height
	app.block = &types.Block{}
	_, err = app.block.UnmarshalMsg(blk.BlockInfo)
	if err != nil {
		panic(err)
	}
	app.syncBlockInfo()
	app.Logger.Debug("updateState done", "latestHeight", height)
	return height
}

func (app *App) catchupLeader(storeHeight int64) int64 {
	latestHeight, err := app.StateProducer.GeLatestBlockHeight()
	for err != nil {
		app.Logger.Debug("catchupLeader failed", "error:", err)
		time.Sleep(3 * time.Second)
		latestHeight, err = app.StateProducer.GeLatestBlockHeight()
	}
	app.Logger.Debug("catchupLeader", "latestHeight", latestHeight)
	for h := storeHeight + 1; h <= latestHeight; h++ {
		h = app.updateState(h)
	}
	return latestHeight
}

func (app *App) syncBlockInfo() *types.BlockInfo {
	bi := &types.BlockInfo{
		Coinbase:  app.block.Miner,
		Number:    app.block.Number,
		Timestamp: app.block.Timestamp,
		ChainId:   app.ChainId.Bytes32(),
		Hash:      app.block.Hash,
	}
	app.blockInfo.Store(bi)
	app.Logger.Debug("blockInfo", "height", bi.Number, "hash", gethcmn.Hash(bi.Hash).Hex())
	return bi
}

func (app *App) LoadBlockInfo() *types.BlockInfo {
	return app.blockInfo.Load().(*types.BlockInfo)
}

func (app *App) GetRunTxContext() *types.Context {
	c := types.NewContext(nil, nil)
	r := rabbit.NewRabbitStore(app.Trunk)
	c = c.WithRbt(&r)
	c = c.WithDb(app.HistoryStore)
	c.SetShaGateForkBlock(param.ShaGateForkBlock)
	c.SetXHedgeForkBlock(param.XHedgeForkBlock)
	c.SetCurrentHeight(app.currHeight)
	return c
}

func (app *App) GetRpcContext() *types.Context {
	c := types.NewContext(nil, nil)
	r := rabbit.NewReadOnlyRabbitStore(app.Root)
	c = c.WithRbt(&r)
	c = c.WithDb(app.HistoryStore)
	c.SetShaGateForkBlock(param.ShaGateForkBlock)
	c.SetXHedgeForkBlock(param.XHedgeForkBlock)
	c.SetCurrentHeight(app.currHeight)
	return c
}

func (app *App) GetRpcContextAtHeight(height int64) *types.Context {
	if !app.Config.AppConfig.ArchiveMode || height < 0 {
		return app.GetRpcContext()
	}

	c := types.NewContext(nil, nil)
	r := rabbit.NewReadOnlyRabbitStoreAtHeight(app.Root, uint64(height))
	c = c.WithRbt(&r)
	c = c.WithDb(app.HistoryStore)
	c.SetShaGateForkBlock(param.ShaGateForkBlock)
	c.SetXHedgeForkBlock(param.XHedgeForkBlock)
	c.SetCurrentHeight(height)
	return c
}

func (app *App) GetHistoryOnlyContext() *types.Context {
	c := types.NewContext(nil, nil)
	c = c.WithDb(app.HistoryStore)
	c.SetShaGateForkBlock(param.ShaGateForkBlock)
	c.SetXHedgeForkBlock(param.XHedgeForkBlock)
	c.SetCurrentHeight(app.currHeight)
	return c
}

func (app *App) RunTxForRpc(gethTx *gethtypes.Transaction, sender gethcmn.Address, estimateGas bool, height int64) (*ebp.TxRunner, int64) {
	txToRun := &types.TxToRun{}
	txToRun.FromGethTx(gethTx, sender, uint64(app.currHeight))
	ctx := app.GetRpcContextAtHeight(height)
	defer ctx.Close(false)
	runner := ebp.NewTxRunner(ctx, txToRun)
	bi := app.LoadBlockInfo()
	if height > 0 {
		blk, err := ctx.GetBlockByHeight(uint64(height))
		if err != nil {
			return nil, 0
		}
		bi = &types.BlockInfo{
			Coinbase:  blk.Miner,
			Number:    blk.Number,
			Timestamp: blk.Timestamp,
			ChainId:   app.ChainId.Bytes32(),
			Hash:      blk.Hash,
		}
	}
	estimateResult := ebp.RunTxForRpc(bi, estimateGas, runner)
	return runner, estimateResult
}

// RunTxForSbchRpc is like RunTxForRpc, with two differences:
// 1. estimateGas is always false
// 2. Run under context of block#height-1
func (app *App) RunTxForSbchRpc(gethTx *gethtypes.Transaction, sender gethcmn.Address, height int64) (*ebp.TxRunner, int64) {
	if height < 1 {
		return app.RunTxForRpc(gethTx, sender, false, height)
	}

	txToRun := &types.TxToRun{}
	txToRun.FromGethTx(gethTx, sender, uint64(app.currHeight))
	ctx := app.GetRpcContextAtHeight(height - 1)
	defer ctx.Close(false)
	runner := ebp.NewTxRunner(ctx, txToRun)
	blk, err := ctx.GetBlockByHeight(uint64(height))
	if err != nil {
		return nil, 0
	}
	bi := &types.BlockInfo{
		Coinbase:  blk.Miner,
		Number:    blk.Number,
		Timestamp: blk.Timestamp,
		ChainId:   app.ChainId.Bytes32(),
		Hash:      blk.Hash,
	}
	estimateResult := ebp.RunTxForRpc(bi, false, runner)
	return runner, estimateResult
}

func (app *App) GetLatestBlockNum() int64 {
	return app.currHeight
}

func (app *App) ChainID() *uint256.Int {
	return app.ChainId
}

func (app *App) GetValidatorsInfo() ValidatorsInfo {
	ctx := app.GetRpcContext()
	defer ctx.Close(false)
	return app.getValidatorsInfoFromCtx(ctx)
}

func (app *App) getValidatorsInfoFromCtx(ctx *types.Context) ValidatorsInfo {
	stakingInfo := staking.LoadStakingInfo(ctx)
	currValidators := stakingtypes.GetActiveValidators(stakingInfo.Validators, staking.MinimumStakingAmount)
	minGasPrice := staking.LoadMinGasPrice(ctx, false)
	lastMinGasPrice := staking.LoadMinGasPrice(ctx, true)
	return newValidatorsInfo(currValidators, stakingInfo, minGasPrice, lastMinGasPrice)
}

func (app *App) IsArchiveMode() bool {
	return app.Config.AppConfig.ArchiveMode
}
