package app

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/holiman/uint256"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	cryptoenc "github.com/tendermint/tendermint/crypto/encoding"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingads"
	"github.com/smartbch/moeingads/store"
	"github.com/smartbch/moeingads/store/rabbit"
	"github.com/smartbch/moeingdb/modb"
	"github.com/smartbch/moeingdb/syncdb"
	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/moeingevm/types"

	"github.com/smartbch/smartbch/crosschain"
	cctypes "github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/staking"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
	"github.com/smartbch/smartbch/watcher"
)

var (
	_ abcitypes.Application = (*App)(nil)
	_ IApp                  = (*App)(nil)
)

const (
	CannotDecodeTx       uint32 = 101
	CannotRecoverSender  uint32 = 102
	SenderNotFound       uint32 = 103
	AccountNonceMismatch uint32 = 104
	CannotPayGasFee      uint32 = 105
	GasLimitInvalid      uint32 = 106
	InvalidMinGasPrice   uint32 = 107
	HasPendingTx         uint32 = 108
	MempoolBusy          uint32 = 109
	GasLimitTooSmall     uint32 = 110
)

var (
	errNoSyncDB    = errors.New("syncdb is not open")
	errNoSyncBlock = errors.New("syncdb block is not ready")
)

type IApp interface {
	ChainID() *uint256.Int
	GetRunTxContext() *types.Context
	GetRpcContext() *types.Context
	GetRpcContextAtHeight(height int64) *types.Context
	GetHistoryOnlyContext() *types.Context
	RunTxForRpc(gethTx *gethtypes.Transaction, sender gethcmn.Address, estimateGas bool, height int64) (*ebp.TxRunner, int64)
	RunTxForSbchRpc(gethTx *gethtypes.Transaction, sender gethcmn.Address, height int64) (*ebp.TxRunner, int64)
	GetCurrEpoch() *stakingtypes.Epoch
	GetWatcherEpochList() []*stakingtypes.Epoch
	GetAppEpochList() []*stakingtypes.Epoch
	GetLatestBlockNum() int64
	SubscribeChainEvent(ch chan<- types.ChainEvent) event.Subscription
	SubscribeLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription
	LoadBlockInfo() *types.BlockInfo
	GetValidatorsInfo() ValidatorsInfo
	IsArchiveMode() bool
	GetBlockForSync(height int64) (blk []byte, err error)
	GetRedeemingUtxoIds() [][36]byte
	GetLostAndFoundUtxoIds() [][36]byte
	GetRedeemableUtxoIdsByCovenantAddr(addr [20]byte) [][36]byte
	GetWatcherHeight() int64
	InjectHandleUtxosFault()
	InjectRedeemFault()
	InjectTransferByBurnFault()
}

type App struct {
	mtx sync.Mutex

	//config
	config  *param.ChainConfig
	chainId *uint256.Int

	//store
	mads         *moeingads.MoeingADS
	root         *store.RootStore
	historyStore modbtypes.DB
	syncDB       *syncdb.SyncDB

	currHeight int64
	trunk      *store.TrunkStore
	checkTrunk *store.TrunkStore
	// 'block' contains some meta information of a block. It is collected during BeginBlock&DeliverTx,
	// and save to world state in Commit.
	block *types.Block
	// Some fields of 'block' are copied to 'blockInfo' in Commit. It will be later used by RpcContext
	// Thus, eth_call can view the new block's height a little earlier than eth_blockNumber
	blockInfo       atomic.Value // to store *types.BlockInfo
	slashValidators [][20]byte   // updated in BeginBlock, used in Commit
	lastVoters      [][]byte     // updated in BeginBlock, used in Commit
	lastProposer    [20]byte     // updated in refresh of last block, used in updateValidatorsAndStakingInfo
	// of current block. It needs to be reloaded in NewApp
	lastGasUsed     uint64      // updated in last block's postCommit, used in current block's refresh
	lastGasRefund   uint256.Int // updated in last block's postCommit, used in current block's refresh
	lastGasFee      uint256.Int // updated in last block's postCommit, used in current block's refresh
	lastMinGasPrice uint64      // updated in refresh, used in next block's CheckTx and Commit. It needs
	// to be reloaded in NewApp
	txid2sigMap map[[32]byte][65]byte //updated in DeliverTx, flushed in refresh

	// feeds
	chainFeed event.Feed // For pub&sub new blocks
	logsFeed  event.Feed // For pub&sub new logs
	scope     event.SubscriptionScope

	//engine
	txEngine    ebp.TxExecutor
	reorderSeed int64        // recorded in BeginBlock, used in Commit
	frontier    ebp.Frontier // recorded in Commit, used in next block's CheckTx

	//watcher
	watcher             *watcher.Watcher
	epochList           []*stakingtypes.Epoch      // caches the epochs collected by the watcher
	monitorVoteInfoList []*cctypes.MonitorVoteInfo // caches the monitor vote infos collected by the watcher

	//util
	signer gethtypes.Signer
	logger log.Logger

	//for amber
	currValidators []*stakingtypes.Validator

	// tendermint wants to know validators whose voting power change
	// it is loaded from ctx in Commit and used in EndBlock
	validatorUpdate []*stakingtypes.Validator

	//signature cache, cache ecrecovery's resulting sender addresses, to speed up checktx
	sigCache map[gethcmn.Hash]SenderAndHeight

	// it shows how many tx remains in the mempool after committing a new block
	recheckCounter int

	// todo: for injection fault test
	CcContractExecutor *crosschain.CcContractExecutor
}

// The value entry of signature cache. The Height helps in evicting old entries.
type SenderAndHeight struct {
	Sender gethcmn.Address
	Height int64
}

func NewApp(config *param.ChainConfig, chainId *uint256.Int, genesisWatcherHeight, genesisCCHeight int64, logger log.Logger, skipSanityCheck bool) *App {
	app := &App{}
	/*------set config------*/
	app.config = config
	app.chainId = chainId
	/*------signature cache------*/
	app.sigCache = make(map[gethcmn.Hash]SenderAndHeight, config.AppConfig.SigCacheSize)
	/*------set util------*/
	app.signer = gethtypes.NewEIP155Signer(app.chainId.ToBig())
	app.logger = logger.With("module", "app")
	/*------set store------*/
	app.root, app.mads = CreateRootStore(config.AppConfig.AppDataPath, config.AppConfig.ArchiveMode)
	app.historyStore = CreateHistoryStore(config.AppConfig.ModbDataPath, config.AppConfig.UseLiteDB, config.AppConfig.RpcEthGetLogsMaxResults,
		app.logger.With("module", "modb"))
	if config.AppConfig.WithSyncDB {
		app.syncDB = syncdb.NewSyncDB(config.AppConfig.SyncdbDataPath)
	}
	app.trunk = app.root.GetTrunkStore(config.AppConfig.TrunkCacheSize).(*store.TrunkStore)
	app.checkTrunk = app.root.GetReadOnlyTrunkStore(config.AppConfig.TrunkCacheSize).(*store.TrunkStore)
	/*------set engine------*/
	app.txEngine = ebp.NewEbpTxExec(
		param.EbpExeRoundCount,
		param.EbpRunnerNumber,
		param.EbpParallelNum,
		5000, /*not consensus relevant*/
		app.signer,
		app.logger.With("module", "engine"))
	//ebp.AdjustGasUsed = false

	// must refresh ctx.Height when app.currHeight set later
	ctx := app.GetRunTxContext()
	/*------set system contract------*/
	ebp.RegisterPredefinedContract(ctx, staking.StakingContractAddress, staking.NewStakingContractExecutor(app.logger.With("module", "staking")))

	// We assign empty maps to them just to avoid accessing nil-maps.
	// Commit will assign meaningful contents to them
	app.txid2sigMap = make(map[[32]byte][65]byte)
	app.frontier = ebp.GetEmptyFrontier()

	/*------set refresh field------*/
	prevBlk := ctx.GetCurrBlockBasicInfo()
	if prevBlk != nil {
		app.block = prevBlk //will be overwritten in BeginBlock soon
		app.currHeight = app.block.Number
		app.lastProposer = app.block.Miner
	} else {
		app.block = &types.Block{}
	}
	app.root.SetHeight(app.currHeight)
	ctx.SetCurrentHeight(app.currHeight)
	app.txEngine.SetContext(app.GetRunTxContext())
	/*------set stakingInfo------*/
	stakingInfo := staking.LoadStakingInfo(ctx)
	currValidators := stakingtypes.GetActiveValidators(stakingInfo.Validators, staking.MinimumStakingAmount)
	app.validatorUpdate = stakingInfo.ValidatorsUpdate
	for _, val := range currValidators {
		app.logger.Debug(fmt.Sprintf("Load validator in NewApp: address(%s), pubkey(%s), votingPower(%d)",
			gethcmn.Address(val.Address).String(), ed25519.PubKey(val.Pubkey[:]).String(), val.VotingPower))
	}
	if stakingInfo.CurrEpochNum == 0 && stakingInfo.GenesisMainnetBlockHeight == 0 {
		stakingInfo.GenesisMainnetBlockHeight = genesisWatcherHeight
		staking.SaveStakingInfo(ctx, stakingInfo) // only executed at genesis
	}
	/*------set cc------*/
	ccExecutor := crosschain.NewCcContractExecutor(app.logger.With("module", "crosschain"), crosschain.VoteContract{})
	if ctx.IsShaGateFork() {
		ebp.RegisterPredefinedContract(ctx, crosschain.CCContractAddress, ccExecutor)
		app.CcContractExecutor = ccExecutor
	}
	/*------set watcher------*/
	lastEpochEndHeight := stakingInfo.GenesisMainnetBlockHeight + param.StakingNumBlocksInEpoch*stakingInfo.CurrEpochNum
	app.watcher = watcher.NewWatcher(app.logger.With("module", "watcher"), app.historyStore, lastEpochEndHeight, stakingInfo.CurrEpochNum, app.config)
	app.logger.Debug(fmt.Sprintf("New watcher: mainnet url(%s), epochNum(%d), lastEpochEndHeight:(%d), speedUp(%v)\n",
		config.AppConfig.MainnetRPCUrl, stakingInfo.CurrEpochNum, lastEpochEndHeight, config.AppConfig.Speedup))
	app.watcher.SetCCExecutor(ccExecutor)
	app.watcher.CheckSanity(skipSanityCheck)
	app.watcher.SetContextGetter(app)
	go app.watcher.Run()
	if ctx.IsShaGateFork() {
		crosschain.WaitUTXOCollectDone(ctx, app.watcher.CcContractExecutor.UTXOInitCollectDoneChan)
	}
	app.watcher.WaitCatchup()
	app.lastMinGasPrice = staking.LoadMinGasPrice(ctx, true)
	if app.currHeight != 0 { // restart postCommit
		app.mtx.Lock()
		app.postCommit(app.syncBlockInfo())
	}
	ctx.Close(true)
	return app
}

func CreateRootStore(dataPath string, isArchiveMode bool) (*store.RootStore, *moeingads.MoeingADS) {
	first := [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
	last := [8]byte{255, 255, 255, 255, 255, 255, 255, 255}
	mads, err := moeingads.NewMoeingADS(dataPath, isArchiveMode,
		[][]byte{first[:], last[:]})
	if err != nil {
		panic(err)
	}
	root := store.NewRootStore(mads, func(k []byte) bool {
		return len(k) >= 1 && k[0] > (128+64) //only cache the standby queue
	})
	return root, mads
}

func CreateHistoryStore(dataPath string, useLiteDB bool, maxLogResults int, logger log.Logger) (historyStore modbtypes.DB) {
	modbDir := dataPath
	if useLiteDB {
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
		historyStore.SetMaxEntryCount(maxLogResults)
	}
	return
}

func (app *App) Info(req abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{
		LastBlockHeight:  app.block.Number,
		LastBlockAppHash: app.root.GetRootHash(),
	}
}

func (app *App) SetOption(option abcitypes.RequestSetOption) abcitypes.ResponseSetOption {
	return abcitypes.ResponseSetOption{} // take it as a nop
}

func (app *App) Query(req abcitypes.RequestQuery) abcitypes.ResponseQuery {
	return abcitypes.ResponseQuery{Code: abcitypes.CodeTypeOK} // take it as a nop
}

func (app *App) sigCacheAdd(txid gethcmn.Hash, value SenderAndHeight) {
	if len(app.sigCache) > app.config.AppConfig.SigCacheSize { //select one old entry to evict
		delKey, minHeight, count := gethcmn.Hash{}, int64(math.MaxInt64), 6 /*iterate 6 steps*/
		for key, value := range app.sigCache {                              //pseudo-random iterate
			if minHeight > value.Height { //select the oldest entry within a short iteration
				minHeight, delKey = value.Height, key
			}
			if count--; count == 0 {
				break
			}
		}
		delete(app.sigCache, delKey)
	}
	app.sigCache[txid] = value
}

func (app *App) CheckTx(req abcitypes.RequestCheckTx) abcitypes.ResponseCheckTx {
	app.logger.Debug("enter check tx!")
	if req.Type == abcitypes.CheckTxType_Recheck {
		app.recheckCounter++ // calculate how many TXs remain in the mempool after a new block
	} else if app.recheckCounter > app.config.AppConfig.RecheckThreshold {
		// Refuse to accept new TXs on P2P to drain the remain TXs in mempool
		return abcitypes.ResponseCheckTx{Code: MempoolBusy, Info: "mempool is too busy"}
	}
	tx := &gethtypes.Transaction{}
	err := tx.DecodeRLP(rlp.NewStream(bytes.NewReader(req.Tx), 0))
	if err != nil {
		return abcitypes.ResponseCheckTx{Code: CannotDecodeTx}
	}
	txid := tx.Hash()
	var sender gethcmn.Address
	senderAndHeight, ok := app.sigCache[txid]
	if ok { // cache hit
		sender = senderAndHeight.Sender
	} else { // cache miss
		sender, err = app.signer.Sender(tx)
		if err != nil {
			return abcitypes.ResponseCheckTx{Code: CannotRecoverSender, Info: "invalid sender: " + err.Error()}
		}
		app.sigCacheAdd(txid, SenderAndHeight{sender, app.currHeight})
	}
	return app.checkTxWithContext(tx, sender, req.Type)
}

func (app *App) checkTxWithContext(tx *gethtypes.Transaction, sender gethcmn.Address, txType abcitypes.CheckTxType) abcitypes.ResponseCheckTx {
	ctx := app.GetCheckTxContext()
	defer ctx.Close(false)
	if ok, res := checkGasLimit(tx); !ok {
		return res
	}
	acc := ctx.GetAccount(sender)
	if acc == nil {
		return abcitypes.ResponseCheckTx{Code: SenderNotFound, Info: types.ErrAccountNotExist.Error()}
	}
	targetNonce, exist := app.frontier.GetLatestNonce(sender)
	if !exist {
		app.frontier.SetLatestBalance(sender, acc.Balance().Clone())
		targetNonce = acc.Nonce()
	}
	app.logger.Debug("checkTxWithContext",
		"isReCheckTx", txType == abcitypes.CheckTxType_Recheck,
		"tx.nonce", tx.Nonce(),
		"account.nonce", acc.Nonce(),
		"targetNonce", targetNonce)

	if tx.Nonce() > targetNonce {
		return abcitypes.ResponseCheckTx{Code: AccountNonceMismatch, Info: "bad nonce: " + types.ErrNonceTooLarge.Error()}
	} else if tx.Nonce() < targetNonce {
		return abcitypes.ResponseCheckTx{Code: AccountNonceMismatch, Info: "bad nonce: " + types.ErrNonceTooSmall.Error()}
	}
	gasPrice, _ := uint256.FromBig(tx.GasPrice())
	gasFee := uint256.NewInt(0).Mul(gasPrice, uint256.NewInt(tx.Gas()))
	if gasPrice.Cmp(uint256.NewInt(app.lastMinGasPrice)) < 0 {
		return abcitypes.ResponseCheckTx{Code: InvalidMinGasPrice, Info: "gas price too small"}
	}
	balance, ok := app.frontier.GetLatestBalance(sender)
	if !ok || balance.Cmp(gasFee) < 0 {
		return abcitypes.ResponseCheckTx{Code: CannotPayGasFee, Info: "failed to deduct tx fee"}
	}
	totalGasLimit, _ := app.frontier.GetLatestTotalGas(sender)
	if exist { // We do not count in the gas of the first tx found during CheckTx
		totalGasLimit += tx.Gas()
	}
	if totalGasLimit > app.config.AppConfig.FrontierGasLimit {
		return abcitypes.ResponseCheckTx{Code: GasLimitInvalid, Info: "send transaction too frequent"}
	}
	app.frontier.SetLatestTotalGas(sender, totalGasLimit)
	//update frontier
	app.frontier.SetLatestNonce(sender, tx.Nonce()+1)
	balance.Sub(balance, gasFee)
	value, _ := uint256.FromBig(tx.Value())
	if balance.Cmp(value) < 0 {
		app.frontier.SetLatestBalance(sender, uint256.NewInt(0))
	} else {
		balance = balance.Sub(balance, value)
		app.frontier.SetLatestBalance(sender, balance)
	}
	app.logger.Debug("checkTxWithContext:", "value", value.String(), "balance", balance.String())
	app.logger.Debug("leave check tx!")
	return abcitypes.ResponseCheckTx{
		Code:      abcitypes.CodeTypeOK,
		GasWanted: int64(tx.Gas()),
	}
}

func checkGasLimit(tx *gethtypes.Transaction) (ok bool, res abcitypes.ResponseCheckTx) {
	intrinsicGas, err2 := gethcore.IntrinsicGas(tx.Data(), nil, tx.To() == nil, true, true)
	if err2 != nil || tx.Gas() < intrinsicGas {
		return false, abcitypes.ResponseCheckTx{Code: GasLimitTooSmall, Info: "gas limit too small"}
	}
	if tx.Gas() > param.MaxTxGasLimit {
		return false, abcitypes.ResponseCheckTx{Code: GasLimitInvalid, Info: "invalid gas limit"}
	}
	return true, abcitypes.ResponseCheckTx{}
}

func (app *App) InitChain(req abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	app.logger.Debug("InitChain, id=", req.ChainId)
	if len(req.AppStateBytes) == 0 {
		panic("no AppStateBytes")
	}
	genesisData := GenesisData{}
	err := json.Unmarshal(req.AppStateBytes, &genesisData)
	if err != nil {
		panic(err)
	}
	app.createGenesisAccounts(genesisData.Alloc)
	genesisValidators := genesisData.StakingValidators()
	if len(genesisValidators) == 0 {
		panic("no genesis validator in genesis.json")
	}
	//store all genesis validators even if it is inactive
	ctx := app.GetRunTxContext()
	staking.AddGenesisValidatorsIntoStakingInfo(ctx, genesisValidators)
	ctx.Close(true)
	currValidators := stakingtypes.GetActiveValidators(genesisValidators, staking.MinimumStakingAmount)
	valSet := make([]abcitypes.ValidatorUpdate, len(currValidators))
	for i, v := range currValidators {
		p, _ := cryptoenc.PubKeyToProto(ed25519.PubKey(v.Pubkey[:]))
		valSet[i] = abcitypes.ValidatorUpdate{
			PubKey: p,
			Power:  v.VotingPower,
		}
		app.logger.Debug(fmt.Sprintf("Active genesis validator: address(%s), pubkey(%s), votingPower(%d)\n",
			gethcmn.Address(v.Address).String(), p.String(), v.VotingPower))
	}
	params := &abcitypes.ConsensusParams{
		Block: &abcitypes.BlockParams{
			MaxBytes: param.BlockMaxBytes,
			MaxGas:   param.BlockMaxGas,
		},
	}
	return abcitypes.ResponseInitChain{
		ConsensusParams: params,
		Validators:      valSet,
	}
}

func (app *App) createGenesisAccounts(alloc gethcore.GenesisAlloc) {
	if len(alloc) == 0 {
		return
	}
	rbt := rabbit.NewRabbitStore(app.trunk)
	app.logger.Info("air drop", "accounts", len(alloc))
	for addr, acc := range alloc {
		amt, _ := uint256.FromBig(acc.Balance)
		k := types.GetAccountKey(addr)
		v := types.ZeroAccountInfo()
		v.UpdateBalance(amt)
		rbt.Set(k, v.Bytes())
		//app.logger.Info("Air drop " + amt.String() + " to " + addr.Hex())
	}
	rbt.Close()
	rbt.WriteBack()
}

func (app *App) BeginBlock(req abcitypes.RequestBeginBlock) abcitypes.ResponseBeginBlock {
	for app.block.Timestamp > app.watcher.GetCurrMainnetBlockTimestamp()+12*3600 {
		app.logger.Debug("waiting BCH node catchup...", "smartBCH block timestamp", app.block.Timestamp, "BCH block timestamp", app.watcher.GetCurrMainnetBlockTimestamp())
		time.Sleep(30 * time.Second)
	}
	app.block = &types.Block{
		Number:    req.Header.Height,
		Timestamp: req.Header.Time.Unix(),
		Size:      int64(req.Size()),
	}
	copy(app.block.Miner[:], req.Header.ProposerAddress)
	app.logger.Debug(fmt.Sprintf("current proposer %s, last proposer: %s",
		gethcmn.Address(app.block.Miner).String(), gethcmn.Address(app.lastProposer).String()))
	for _, v := range req.LastCommitInfo.GetVotes() {
		if v.SignedLastBlock {
			app.lastVoters = append(app.lastVoters, v.Validator.Address) //this is validator consensus address
		}
	}
	copy(app.block.ParentHash[:], req.Header.LastBlockId.Hash)
	copy(app.block.TransactionsRoot[:], req.Header.DataHash) //TODO changed to committed tx hash
	app.reorderSeed = 0
	if len(req.Header.DataHash) >= 8 {
		app.reorderSeed = int64(binary.LittleEndian.Uint64(req.Header.DataHash[0:8]))
	}
	copy(app.block.Hash[:], req.Hash) // Just use tendermint's block hash
	copy(app.block.StateRoot[:], req.Header.AppHash)
	app.currHeight = req.Header.Height
	// collect slash info, currently only double signing is slashed
	var addr [20]byte
	for _, val := range req.ByzantineValidators {
		//we always slash, without checking the time of bad behavior
		if val.Type == abcitypes.EvidenceType_DUPLICATE_VOTE {
			copy(addr[:], val.Validator.Address)
			app.slashValidators = append(app.slashValidators, addr)
		}
	}
	return abcitypes.ResponseBeginBlock{}
}

func (app *App) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	app.block.Size += int64(req.Size())
	tx, err := ethutils.DecodeTx(req.Tx)
	if err == nil {
		app.txEngine.CollectTx(tx)
		app.txid2sigMap[tx.Hash()] = ethutils.EncodeVRS(tx)
	}
	return abcitypes.ResponseDeliverTx{Code: abcitypes.CodeTypeOK}
}

func (app *App) EndBlock(req abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	valSet := make([]abcitypes.ValidatorUpdate, len(app.validatorUpdate))
	for i, v := range app.validatorUpdate {
		p, _ := cryptoenc.PubKeyToProto(ed25519.PubKey(v.Pubkey[:]))
		valSet[i] = abcitypes.ValidatorUpdate{
			PubKey: p,
			Power:  v.VotingPower,
		}
		app.logger.Debug(fmt.Sprintf("Validator updated in EndBlock: pubkey(%s) votingPower(%d)",
			hex.EncodeToString(v.Pubkey[:]), v.VotingPower))
	}
	return abcitypes.ResponseEndBlock{
		ValidatorUpdates: valSet,
	}
}

func (app *App) Commit() abcitypes.ResponseCommit {
	app.logger.Debug("Enter commit!", "collected txs", app.txEngine.CollectedTxsCount())
	app.mtx.Lock()
	app.updateValidatorsAndStakingInfo()
	app.frontier = app.txEngine.Prepare(app.reorderSeed, 0, param.MaxTxGasLimit)
	appHash := app.refresh()
	go app.postCommit(app.syncBlockInfo())
	return app.buildCommitResponse(appHash)
}

func (app *App) getBlockRewardAndUpdateSysAcc(ctx *types.Context) *uint256.Int {
	if !app.lastGasRefund.IsZero() {
		err := ebp.SubSystemAccBalance(ctx, &app.lastGasRefund)
		if err != nil {
			panic(err)
		}
	}
	//invariant check for fund safe
	sysB := ebp.GetSystemBalance(ctx)
	if sysB.Cmp(&app.lastGasFee) < 0 {
		panic("system balance not enough!")
	}
	//if there has tx in standbyQ, it means there has some gasFee prepay to system account in app.prepare should not distribute in this block.
	//if standbyQ is empty, we distribute all system account balance as reward in that block,
	//which have all bch sent to system account through external transfer in blocks standbyQ not empty, this is not fair but simple, and this is rarely the case now.
	var blockReward = app.lastGasFee //distribute previous block gas fee
	if app.txEngine.StandbyQLen() == 0 {
		// distribute extra balance to validators
		blockReward = *sysB
	}
	if !blockReward.IsZero() {
		//block reward is subtracted from systemAcc here, and will be added to stakingAcc in SlashAndReward.
		err := ebp.SubSystemAccBalance(ctx, &blockReward)
		if err != nil {
			//todo: be careful
			panic(err)
		}
	}
	return &blockReward
}

func (app *App) buildCommitResponse(appHash []byte) abcitypes.ResponseCommit {
	res := abcitypes.ResponseCommit{
		Data: appHash,
	}
	// prune tendermint history block and state every ChangeRetainEveryN blocks
	if app.config.AppConfig.RetainBlocks > 0 && app.currHeight >= app.config.AppConfig.RetainBlocks && (app.currHeight%app.config.AppConfig.ChangeRetainEveryN == 0) {
		res.RetainHeight = app.currHeight - app.config.AppConfig.RetainBlocks + 1
	}
	return res
}

func (app *App) updateValidatorsAndStakingInfo() {
	ctx := app.GetRunTxContext()
	defer ctx.Close(true) // context must be written back such that txEngine can read it in 'Prepare'
	blkBalance := ebp.GetBlackHoleBalance(ctx)
	fmt.Printf("blackhole balance:%d\n", blkBalance)
	currValidators, newValidators, currEpochNum := staking.SlashAndReward(ctx, app.slashValidators, app.block.Miner,
		app.lastProposer, app.lastVoters, app.getBlockRewardAndUpdateSysAcc(ctx))
	app.slashValidators = app.slashValidators[:0]

	if param.IsAmber && ctx.IsXHedgeFork() {
		//make fake epoch after xHedgeFork, change amber to pure pos
		if (app.currHeight%param.AmberBlocksInEpochAfterXHedgeFork == 0) && (app.currHeight > (ctx.XHedgeForkBlock + param.AmberBlocksInEpochAfterXHedgeFork/2)) {
			e := &stakingtypes.Epoch{}
			app.epochList = append(app.epochList, e)
			app.logger.Debug("Get new fake epoch")
			select {
			case <-app.watcher.EpochChan:
				app.logger.Debug("ignore epoch from watcher after xHedgeFork")
			default:
			}
		}
	} else {
		select {
		case epoch := <-app.watcher.EpochChan:
			app.epochList = append(app.epochList, epoch)
			app.logger.Debug(fmt.Sprintf("Get new epoch, epochNum(%d), startHeight(%d), epochListLens(%d)",
				epoch.Number, epoch.StartHeight, len(app.epochList)))
		default:
		}
		if ctx.IsShaGateFork() {
			select {
			case voteInfo := <-app.watcher.MonitorVoteChan:
				app.monitorVoteInfoList = append(app.monitorVoteInfoList, voteInfo)
				app.logger.Debug(fmt.Sprintf("Get new monitor vote info, infoNum(%d), startHeight(%d), infoListLens(%d)",
					voteInfo.Number, voteInfo.StartHeight, len(app.monitorVoteInfoList)))
			default:
			}
		}
	}

	if len(app.epochList) != 0 {
		//epoch switch delay time should bigger than 10 mainnet block interval as of block finalization need
		epochSwitchDelay := param.StakingEpochSwitchDelay
		// this 20 is hardcode to fix the 20220520 bch node not upgrade error. don't modify it ever.
		if currEpochNum == 20 {
			// make epoch switch delay in epoch 20th 50% longer.
			epochSwitchDelay = param.StakingEpochSwitchDelay * 10
		}
		if app.block.Timestamp > app.epochList[0].EndTime+epochSwitchDelay {
			app.logger.Debug(fmt.Sprintf("Switch epoch at block(%d), eppchNum(%d)",
				app.block.Number, app.epochList[0].Number))
			var posVotes map[[32]byte]int64
			var xHedgeSequence = param.XHedgeContractSequence
			if ctx.IsXHedgeFork() {
				//deploy xHedge contract before fork
				posVotes = staking.GetAndClearPosVotes(ctx, xHedgeSequence)
			}
			newEpoch := app.epochList[0]
			newValidators = staking.SwitchEpoch(ctx, newEpoch, posVotes, app.logger)
			app.epochList = app.epochList[1:] // possible memory leak here, but the length would not be very large
			if ctx.IsXHedgeFork() {
				staking.CreateInitVotes(ctx, xHedgeSequence, newValidators)
			}
			if ctx.IsShaGateFork() {
				if len(app.monitorVoteInfoList) != 0 {
					info := app.monitorVoteInfoList[0]
					info.Number = newEpoch.Number
					app.logger.Debug("save new monitor info")
					crosschain.SaveMonitorVoteInfo(ctx, *info)
					app.monitorVoteInfoList = app.monitorVoteInfoList[1:]
				}
			}
		}
	}

	// hardcode for sync block meet appHash error on 4435201 in amber.
	if param.IsAmber && app.currHeight == 4435201 {
		app.validatorUpdate = nil
	} else if param.IsAmber {
		app.validatorUpdate = stakingtypes.GetUpdateValidatorSet(app.currValidators, newValidators)
	} else {
		app.validatorUpdate = stakingtypes.GetUpdateValidatorSet(currValidators, newValidators)
	}
	for _, v := range app.validatorUpdate {
		app.logger.Debug(fmt.Sprintf("Updated validator in commit: address(%s), pubkey(%s), voting power: %d",
			gethcmn.Address(v.Address).String(), ed25519.PubKey(v.Pubkey[:]), v.VotingPower))
	}
	newInfo := staking.LoadStakingInfo(ctx)
	newInfo.ValidatorsUpdate = app.validatorUpdate
	staking.SaveStakingInfo(ctx, newInfo)
	//only amber need this
	app.currValidators = newValidators
	//log all validators info when validator set update
	if len(app.validatorUpdate) != 0 {
		validatorsInfo := app.getValidatorsInfoFromCtx(ctx)
		bz, _ := json.Marshal(validatorsInfo)
		app.logger.Debug(fmt.Sprintf("ValidatorsInfo:%s", string(bz)))
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
	app.logger.Debug(fmt.Sprintf("blockInfo: [height:%d, hash:%s]", bi.Number, gethcmn.Hash(bi.Hash).Hex()))
	return bi
}

func (app *App) LoadBlockInfo() *types.BlockInfo {
	return app.blockInfo.Load().(*types.BlockInfo)
}

func (app *App) postCommit(bi *types.BlockInfo) {
	defer app.mtx.Unlock()
	if bi != nil {
		if bi.Number > 1 {
			hash := app.historyStore.GetBlockHashByHeight(bi.Number - 1)
			if hash == [32]byte{} {
				app.logger.Debug(fmt.Sprintf("postcommit blockInfo: height:%d, blockHash:%s", bi.Number-1, gethcmn.Hash(hash).Hex()))
			}
		}
	}
	app.txEngine.Execute(bi)
	app.lastGasUsed, app.lastGasRefund, app.lastGasFee = app.txEngine.GasUsedInfo()
}

func (app *App) refresh() (appHash []byte) {
	//close old
	app.checkTrunk.Close(false)
	ctx := app.GetRunTxContext()
	prevBlkInfo := ctx.GetCurrBlockBasicInfo()
	if prevBlkInfo == nil {
		app.logger.Debug(fmt.Sprintf("prevBlkInfo is nil in height:%d", app.block.Number))
	} else {
		app.logger.Debug(fmt.Sprintf("prevBlkInfo: blockHash:%s, number:%d", gethcmn.Hash(prevBlkInfo.Hash).Hex(), prevBlkInfo.Number))
	}
	ctx.SetCurrBlockBasicInfo(app.block)
	//refresh lastMinGasPrice
	mGP := staking.LoadMinGasPrice(ctx, false) // load current block's gas price
	staking.SaveMinGasPrice(ctx, mGP, true)    // save it as last block's gas price
	app.lastMinGasPrice = mGP
	if ctx.IsShaGateFork() {
		ccExecutor := ebp.PredefinedContractManager[crosschain.CCContractAddress]
		if ccExecutor == nil {
			if app.watcher.CcContractExecutor != nil {
				ebp.RegisterPredefinedContract(ctx, crosschain.CCContractAddress, app.watcher.CcContractExecutor)
				// todo: for injection fault test
				app.CcContractExecutor = app.watcher.CcContractExecutor
			} else {
				executor := crosschain.NewCcContractExecutor(app.logger.With("module", "crosschain"), crosschain.VoteContract{})
				app.watcher.SetCCExecutor(executor)
				ebp.RegisterPredefinedContract(ctx, crosschain.CCContractAddress, executor)
				// todo: for injection fault test
				app.CcContractExecutor = executor
			}
		}
	}
	ctx.Close(true)
	lastCacheSize := app.trunk.CacheSize() // predict the next truck's cache size with the last one
	updateOfADS := app.trunk.GetCacheContent()
	app.trunk.Close(true) //write cached KVs back to app.root
	if !app.config.AppConfig.ArchiveMode && prevBlkInfo != nil &&
		prevBlkInfo.Number%app.config.AppConfig.PruneEveryN == 0 &&
		prevBlkInfo.Number > app.config.AppConfig.NumKeptBlocks {
		app.mads.PruneBeforeHeight(prevBlkInfo.Number - app.config.AppConfig.NumKeptBlocks)
	}
	appHash = append([]byte{}, app.root.GetRootHash()...)
	//jump block which prev height = 0
	if prevBlkInfo != nil {
		//use current block commit app hash as prev history block stateRoot
		copy(prevBlkInfo.StateRoot[:], appHash)
		prevBlkInfo.GasUsed = app.lastGasUsed
		prevBlk4MoDB := modbtypes.Block{
			Height: prevBlkInfo.Number,
		}
		prevBlkInfo.Transactions = app.txEngine.CommittedTxIds()
		blkInfo, err := prevBlkInfo.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}
		copy(prevBlk4MoDB.BlockHash[:], prevBlkInfo.Hash[:])
		prevBlk4MoDB.BlockInfo = blkInfo
		prevBlk4MoDB.TxList = app.txEngine.CommittedTxsForMoDB()
		//if ctx.IsShaGateFork() {
		app.historyStore.SetOpListsForCcUtxo(crosschain.CollectOpList(&prevBlk4MoDB))
		//}
		if app.config.AppConfig.NumKeptBlocksInMoDB > 0 && app.currHeight > app.config.AppConfig.NumKeptBlocksInMoDB {
			app.historyStore.AddBlock(&prevBlk4MoDB, app.currHeight-app.config.AppConfig.NumKeptBlocksInMoDB, app.txid2sigMap)
		} else {
			app.historyStore.AddBlock(&prevBlk4MoDB, -1, app.txid2sigMap) // do not prune moeingdb
		}
		if app.syncDB != nil {
			app.syncDB.AddBlock(prevBlk4MoDB.Height, &prevBlk4MoDB, app.txid2sigMap, updateOfADS)
		}
		app.txid2sigMap = make(map[[32]byte][65]byte) // clear its content after flushing into historyStore
		app.publishNewBlock(&prevBlk4MoDB)
	}
	//make new
	app.recheckCounter = 0 // reset counter before counting the remained TXs which need rechecking
	app.lastProposer = app.block.Miner
	app.lastVoters = app.lastVoters[:0]
	app.root.SetHeight(app.currHeight)
	app.trunk = app.root.GetTrunkStore(lastCacheSize).(*store.TrunkStore)
	app.checkTrunk = app.root.GetReadOnlyTrunkStore(app.config.AppConfig.TrunkCacheSize).(*store.TrunkStore)
	app.txEngine.SetContext(app.GetRunTxContext())
	return
}

func (app *App) publishNewBlock(mdbBlock *modbtypes.Block) {
	if mdbBlock == nil {
		return
	}
	chainEvent := types.BlockToChainEvent(mdbBlock)
	app.chainFeed.Send(chainEvent)
	if len(chainEvent.Logs) > 0 {
		app.logsFeed.Send(chainEvent.Logs)
	}
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
func (app *App) GetRunTxContext() *types.Context {
	c := types.NewContext(nil, nil)
	r := rabbit.NewRabbitStore(app.trunk)
	c = c.WithRbt(&r)
	c = c.WithDb(app.historyStore)
	c.SetShaGateForkBlock(param.ShaGateForkBlock)
	c.SetXHedgeForkBlock(param.XHedgeForkBlock)
	c.SetCurrentHeight(app.currHeight)
	c.SetType(types.RunTxType)
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
func (app *App) GetCheckTxContext() *types.Context {
	c := types.NewContext(nil, nil)
	r := rabbit.NewRabbitStore(app.checkTrunk)
	c = c.WithRbt(&r)
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
	bi := app.blockInfo.Load().(*types.BlockInfo)
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

// SubscribeChainEvent registers a subscription of ChainEvent.
func (app *App) SubscribeChainEvent(ch chan<- types.ChainEvent) event.Subscription {
	return app.scope.Track(app.chainFeed.Subscribe(ch))
}

// SubscribeLogsEvent registers a subscription of []*types.Log.
func (app *App) SubscribeLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription {
	return app.scope.Track(app.logsFeed.Subscribe(ch))
}

func (app *App) GetLastGasUsed() uint64 {
	return app.lastGasUsed
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
	return NewValidatorsInfo(currValidators, stakingInfo, minGasPrice, lastMinGasPrice)
}

func (app *App) IsArchiveMode() bool {
	return app.config.AppConfig.ArchiveMode
}

func (app *App) GetCurrEpoch() *stakingtypes.Epoch {
	return app.watcher.GetCurrEpoch()
}

func (app *App) GetAppEpochList() []*stakingtypes.Epoch {
	return stakingtypes.CopyEpochs(app.epochList)
}
func (app *App) GetWatcherEpochList() []*stakingtypes.Epoch {
	return app.watcher.GetEpochList()
}

func (app *App) GetBlockForSync(height int64) (blk []byte, err error) {
	if app.syncDB == nil {
		return nil, errNoSyncDB
	}
	blk = app.syncDB.Get(height)
	if blk == nil {
		err = errNoSyncBlock
	}
	return
}

func (app *App) GetLostAndFoundUtxoIds() [][36]byte {
	return app.historyStore.GetLostAndFoundUtxoIds()
}

func (app *App) GetRedeemingUtxoIds() [][36]byte {
	return app.historyStore.GetRedeemingUtxoIds()
}

func (app *App) GetRedeemableUtxoIdsByCovenantAddr(addr [20]byte) [][36]byte {
	return app.historyStore.GetRedeemableUtxoIdsByCovenantAddr(addr)
}

func (app *App) GetWatcherHeight() int64 {
	return app.watcher.GetLatestFinalizedHeight()
}

func (app *App) InjectHandleUtxosFault() {
	app.CcContractExecutor.HandleUtxosInject = true
	fmt.Println("app.CcContractExecutor.HandleUtxosInject = true")
}

func (app *App) InjectRedeemFault() {
	app.CcContractExecutor.RedeemInject = true
	fmt.Println("app.CcContractExecutor.RedeemInject = true")
}

func (app *App) InjectTransferByBurnFault() {
	app.CcContractExecutor.TransferByBurnInject = true
	fmt.Println("app.CcContractExecutor.TransferByBurnInject = true")
}

//nolint
// for ((i=10; i<80000; i+=50)); do RANDPANICHEIGHT=$i ./smartbchd start; done | tee a.log
func (app *App) randomPanic(baseNumber, primeNumber int64) { // breaks normal function, only used in test
	heightStr := os.Getenv("RANDPANICHEIGHT")
	if len(heightStr) == 0 {
		return
	}
	h, err := strconv.Atoi(heightStr)
	if err != nil {
		panic(err)
	}
	if app.currHeight < int64(h) {
		return
	}
	go func(sleepMilliseconds int64) {
		time.Sleep(time.Duration(sleepMilliseconds * int64(time.Millisecond)))
		s := fmt.Sprintf("random panic after %d millisecond", sleepMilliseconds)
		fmt.Println(s)
		panic(s)
	}(baseNumber + time.Now().UnixNano()%primeNumber)
}
