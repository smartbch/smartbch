package app

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"sync/atomic"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/libs/log"

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
	//config
	config  *param.ChainConfig
	chainId *uint256.Int

	//store
	mads         *moeingads.MoeingADS
	root         *store.RootStore
	historyStore modbtypes.DB

	currHeight int64
	// 'block' contains some meta information of a block. It is collected during BeginBlock&DeliverTx,
	// and save to world state in Commit.
	block *types.Block
	// Some fields of 'block' are copied to 'blockInfo' in Commit. It will be later used by RpcContext
	// Thus, eth_call can view the new block's height a little earlier than eth_blockNumber
	blockInfo atomic.Value // to store *types.BlockInfo
	// of current block. It needs to be reloaded in NewApp
	// to be reloaded in NewApp
	rpcClient *RpcClient
	//util
	logger log.Logger
}

// The value entry of signature cache. The Height helps in evicting old entries.
type SenderAndHeight struct {
	Sender gethcmn.Address
	Height int64
}

func NewApp(config *param.ChainConfig, logger log.Logger) *App {
	app := &App{}

	/*------set config------*/
	app.config = config
	app.chainId = uint256.NewInt(param.ChainID)

	/*------set util------*/
	app.logger = logger.With("module", "app")

	/*------set store------*/
	app.root, app.mads = createRootStore(config)
	app.historyStore = createHistoryStore(config, app.logger.With("module", "modb"))

	app.rpcClient = NewRpcClient(config.AppConfig.SmartBchRPCUrl, "", "", "application/json", app.logger.With("module", "client"))
	go app.run(0)
	return app
}

func createRootStore(config *param.ChainConfig) (*store.RootStore, *moeingads.MoeingADS) {
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

func createHistoryStore(config *param.ChainConfig, logger log.Logger) (historyStore modbtypes.DB) {
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

func (app *App) run(storeHeight int64) {
	//1. fetch blocks until catch up leader.
	latestHeight := app.catchupLeader(storeHeight)
	// run 2 times to catch blocks mint amount 1st catchupLeader running.
	latestHeight = app.catchupLeader(latestHeight)
	//2. keep sync with leader.
	for {
		app.updateState(latestHeight + 1)
		//3. sleep and continue
		time.Sleep(100 * time.Millisecond)
	}
}

func (app *App) updateState(height int64) {
	blk := app.rpcClient.getSyncBlock(uint64(height))
	if blk == nil {
		time.Sleep(100 * time.Millisecond)
		return
	}
	//2. save data to local db
	store.SyncUpdateTo(blk.UpdateOfADS, app.root)
	app.historyStore.AddBlock(&blk.Block, -1, blk.Txid2sigMap)
	//3. refresh app fields
	app.currHeight = blk.Height
	_, err := app.block.UnmarshalMsg(blk.BlockInfo)
	if err != nil {
		panic(err)
	}
	app.syncBlockInfo()
	return
}

func (app *App) catchupLeader(storeHeight int64) int64 {
	latestHeight := app.GetLatestBlockNum()
	if latestHeight == -1 {
		panic(app.rpcClient.err.Error())
	}
	for h := storeHeight + 1; h <= latestHeight; h++ {
		app.updateState(h)
	}
	return latestHeight
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
	app.logger.Debug(fmt.Sprintf("blockInfo: [height:%d, hash:%s]", bi.Number, gethcmn.Hash(bi.Hash).Hex()))
	return bi
}

func (app *App) LoadBlockInfo() *types.BlockInfo {
	return app.blockInfo.Load().(*types.BlockInfo)
}

//func (app *App) refresh() (appHash []byte) {
//	//close old
//	app.checkTrunk.Close(false)
//	lastCacheSize := app.trunk.CacheSize() // predict the next truck's cache size with the last one
//	//updateOfADS := app.trunk.GetCacheContent()
//	app.trunk.Close(true) //write cached KVs back to app.root
//
//	//make new
//	app.root.SetHeight(app.currHeight)
//	app.trunk = app.root.GetTrunkStore(lastCacheSize).(*store.TrunkStore)
//	app.checkTrunk = app.root.GetReadOnlyTrunkStore(app.config.AppConfig.TrunkCacheSize).(*store.TrunkStore)
//	return
//}

func (app *App) GetRpcContext() *types.Context {
	c := types.NewContext(nil, nil)
	r := rabbit.NewReadOnlyRabbitStore(app.root)
	c = c.WithRbt(&r)
	c = c.WithDb(app.historyStore)
	c.SetShaGateForkBlock(param.ShaGateForkBlock)
	c.SetXHedgeForkBlock(param.XHedgeForkBlock)
	c.SetCurrentHeight(app.currHeight)
	return c
}

func (app *App) GetRpcContextAtHeight(height int64) *types.Context {
	if !app.config.AppConfig.ArchiveMode || height < 0 {
		return app.GetRpcContext()
	}

	c := types.NewContext(nil, nil)
	r := rabbit.NewReadOnlyRabbitStoreAtHeight(app.root, uint64(height))
	c = c.WithRbt(&r)
	c = c.WithDb(app.historyStore)
	c.SetShaGateForkBlock(param.ShaGateForkBlock)
	c.SetXHedgeForkBlock(param.XHedgeForkBlock)
	c.SetCurrentHeight(height)
	return c
}

func (app *App) GetHistoryOnlyContext() *types.Context {
	c := types.NewContext(nil, nil)
	c = c.WithDb(app.historyStore)
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
			ChainId:   app.chainId.Bytes32(),
			Hash:      blk.Hash,
		}
	}
	estimateResult := ebp.RunTxForRpc(bi, estimateGas, runner)
	return runner, estimateResult
}

// RunTxForSbchRpc is like RunTxForRpc, with two differences:
// 1. estimateGas is always false
// 2. run under context of block#height-1
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
		ChainId:   app.chainId.Bytes32(),
		Hash:      blk.Hash,
	}
	estimateResult := ebp.RunTxForRpc(bi, false, runner)
	return runner, estimateResult
}

func (app *App) GetLatestBlockNum() int64 {
	return app.currHeight
}

func (app *App) ChainID() *uint256.Int {
	return app.chainId
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
	return app.config.AppConfig.ArchiveMode
}
