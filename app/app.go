package app

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"sort"
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
)

type ContextMode uint8

const (
	CheckTxMode     ContextMode = iota
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
	GasLimitInvalid      uint32 = 106
	InvalidMinGasPrice   uint32 = 107
	HasPendingTx         uint32 = 108
	MempoolBusy          uint32 = 109

	PruneEveryN = 10

	BlockMaxBytes = 24 * 1024 * 1024 // 24MB
	BlockMaxGas   = 900_000_000_000

	DefaultTrunkCacheSize = 200
)

type App struct {
	mtx sync.Mutex

	//config
	chainId      *uint256.Int
	retainBlocks int64

	//store
	mads          *moeingads.MoeingADS
	root          *store.RootStore
	numKeptBlocks int64
	historyStore  modbtypes.DB

	//refresh with block
	currHeight      int64
	trunk           *store.TrunkStore
	checkTrunk      *store.TrunkStore
	block           *types.Block
	blockInfo       atomic.Value // to store *types.BlockInfo
	slashValidators [][20]byte
	lastVoters      [][]byte
	lastProposer    [20]byte
	lastGasUsed     uint64
	lastGasRefund   uint256.Int
	lastGasFee      uint256.Int
	lastMinGasPrice uint64

	// feeds
	chainFeed event.Feed
	logsFeed  event.Feed
	scope     event.SubscriptionScope

	//engine
	txEngine     ebp.TxExecutor
	reorderSeed  int64
	touchedAddrs map[gethcmn.Address]int

	//watcher
	watcher   *staking.Watcher
	epochList []*stakingtypes.Epoch

	//util
	signer gethtypes.Signer
	logger log.Logger

	//genesis data
	currValidators  []*stakingtypes.Validator
	validatorUpdate []*stakingtypes.Validator

	//signature cache, cache ecrecovery's resulting sender addresses, to speed up checktx
	sigCache     map[gethcmn.Hash]SenderAndHeight
	sigCacheSize int

	// the senders who send native tokens through sep206 in the last block
	sep206SenderSet map[gethcmn.Address]struct{}
	// it shows how many tx remains in the mempool after committing a new block
	recheckCounter int
	// if recheckCounter is larger than recheckThreshold, mempool is in a traffic-jam status
	// and we'd better refuse further transactions
	recheckThreshold int
}

// The value entry of signature cache. The Height helps in evicting old entries.
type SenderAndHeight struct {
	Sender gethcmn.Address
	Height int64
}

func NewApp(config *param.ChainConfig, chainId *uint256.Int, genesisWatcherHeight int64, logger log.Logger) *App {
	app := &App{}

	app.recheckThreshold = config.RecheckThreshold

	/*------signature cache------*/
	app.sigCacheSize = config.SigCacheSize
	app.sigCache = make(map[gethcmn.Hash]SenderAndHeight, app.sigCacheSize)

	/*------set config------*/
	app.retainBlocks = config.RetainBlocks
	app.chainId = chainId

	/*------set store------*/
	app.numKeptBlocks = int64(config.NumKeptBlocks)
	app.root, app.mads = createRootStore(config)
	app.historyStore = createHistoryStore(config)
	app.trunk = app.root.GetTrunkStore(DefaultTrunkCacheSize).(*store.TrunkStore)
	app.checkTrunk = app.root.GetReadOnlyTrunkStore(DefaultTrunkCacheSize).(*store.TrunkStore)

	/*------set util------*/
	app.signer = gethtypes.NewEIP155Signer(app.chainId.ToBig())
	app.logger = logger.With("module", "app")

	/*------set engine------*/
	app.txEngine = ebp.NewEbpTxExec(200 /*exeRoundCount*/, 256 /*runnerNumber*/, 32, /*parallelNum*/
		5000 /*defaultTxListCap*/, app.signer)

	/*------set system contract------*/
	ctx := app.GetRunTxContext()
	//init PredefinedSystemContractExecutors before tx execute
	ebp.PredefinedSystemContractExecutor = &staking.StakingContractExecutor{}
	ebp.PredefinedSystemContractExecutor.Init(ctx)

	// We make these maps not for really usage, just to avoid accessing nil-maps
	app.touchedAddrs = make(map[gethcmn.Address]int)
	app.sep206SenderSet = make(map[gethcmn.Address]struct{})

	/*------set refresh field------*/
	prevBlk := ctx.GetCurrBlockBasicInfo()
	app.block = &types.Block{}
	if prevBlk != nil {
		app.block.Number = prevBlk.Number
		app.currHeight = app.block.Number
	}

	app.root.SetHeight(app.currHeight + 1)
	if app.currHeight != 0 {
		app.reload()
	} else {
		app.txEngine.SetContext(app.GetRunTxContext())
	}

	/*------set stakingInfo------*/
	acc, stakingInfo := staking.LoadStakingAcc(ctx)
	app.currValidators = stakingInfo.GetActiveValidators(staking.MinimumStakingAmount)
	app.validatorUpdate = stakingInfo.ValidatorsUpdate
	for _, val := range app.currValidators {
		fmt.Printf("validator:%v\n", val.Address)
	}
	if stakingInfo.CurrEpochNum == 0 {
		stakingInfo.GenesisMainnetBlockHeight = genesisWatcherHeight
		staking.SaveStakingInfo(ctx, acc, stakingInfo)
	}

	/*------set watcher------*/
	client := staking.NewParallelRpcClient(config.MainnetRPCUrl, config.MainnetRPCUserName, config.MainnetRPCPassword)
	lastWatch2016xHeight := stakingInfo.GenesisMainnetBlockHeight + staking.NumBlocksInEpoch*stakingInfo.CurrEpochNum
	fmt.Printf("current epoch:%d,lastWatch2016xHeight:%d in NewApp\n", stakingInfo.CurrEpochNum, lastWatch2016xHeight)
	app.watcher = staking.NewWatcher(lastWatch2016xHeight, client)
	catchupChan := make(chan bool, 1)
	go app.watcher.Run(catchupChan)
	<-catchupChan

	app.lastMinGasPrice = staking.LoadMinGasPrice(ctx, true)
	ctx.Close(true)
	return app
}

func createRootStore(config *param.ChainConfig) (*store.RootStore, *moeingads.MoeingADS) {
	first := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	last := []byte{255, 255, 255, 255, 255, 255, 255, 255}
	mads, err := moeingads.NewMoeingADS(config.AppDataPath, false, [][]byte{first, last})
	if err != nil {
		panic(err)
	}
	return store.NewRootStore(mads, nil), mads
}

func createHistoryStore(config *param.ChainConfig) (historyStore modbtypes.DB) {
	modbDir := config.ModbDataPath
	if config.UseLiteDB {
		historyStore = modb.NewLiteDB(modbDir)
	} else {
		if _, err := os.Stat(modbDir); os.IsNotExist(err) {
			_ = os.MkdirAll(path.Join(modbDir, "data"), 0700)
			historyStore = modb.CreateEmptyMoDB(modbDir, [8]byte{1, 2, 3, 4, 5, 6, 7, 8})
		} else {
			historyStore = modb.NewMoDB(modbDir)
		}
		historyStore.SetMaxEntryCount(config.RpcEthGetLogsMaxResults)
	}
	return
}

func (app *App) reload() {
	app.txEngine.SetContext(app.GetRunTxContext())
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
	if req.Type == abcitypes.CheckTxType_Recheck {
		app.recheckCounter++ // calculate how many TXs remain in the mempool after a new block
	} else if app.recheckCounter > app.recheckThreshold {
		// Refuse to accept new TXs to drain the remain TXs
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
		if len(app.sigCache) > app.sigCacheSize { //select one old entry to evict
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
		app.sigCache[txid] = SenderAndHeight{sender, app.currHeight} // add to cache
	}
	if _, ok := app.touchedAddrs[sender]; ok {
		// if the sender is touched, it is most likely to have an uncertain nonce, so we reject it
		return abcitypes.ResponseCheckTx{Code: HasPendingTx, Info: "still has pending transaction"}
	}
	if req.Type == abcitypes.CheckTxType_Recheck {
		// During rechecking, if the sender has not not been touched or lose balance, the tx can pass
		if _, ok := app.sep206SenderSet[sender]; !ok {
			return abcitypes.ResponseCheckTx{
				Code:      abcitypes.CodeTypeOK,
				GasWanted: int64(tx.Gas()),
			}
		}
	}
	return app.checkTx(tx, sender)
}

func (app *App) checkTx(tx *gethtypes.Transaction, sender gethcmn.Address) abcitypes.ResponseCheckTx {
	ctx := app.GetCheckTxContext()
	dirty := false
	defer func(dirtyPtr *bool) {
		ctx.Close(*dirtyPtr)
	}(&dirty)

	//todo: replace with engine param
	if tx.Gas() > ebp.MaxTxGasLimit {
		return abcitypes.ResponseCheckTx{Code: GasLimitInvalid, Info: "invalid gas limit"}
	}
	acc, err := ctx.CheckNonce(sender, tx.Nonce())
	if err != nil {
		return abcitypes.ResponseCheckTx{Code: AccountNonceMismatch, Info: "bad nonce: " + err.Error()}
	}
	gasPrice, _ := uint256.FromBig(tx.GasPrice())
	if gasPrice.Cmp(uint256.NewInt().SetUint64(app.lastMinGasPrice)) < 0 {
		return abcitypes.ResponseCheckTx{Code: InvalidMinGasPrice, Info: "gas price too small"}
	}
	err = ctx.DeductTxFee(sender, acc, tx.Gas(), gasPrice)
	if err != nil {
		return abcitypes.ResponseCheckTx{Code: CannotPayGasFee, Info: "failed to deduct tx fee"}
	}
	dirty = true
	app.logger.Debug("leave check tx!")
	return abcitypes.ResponseCheckTx{
		Code:      abcitypes.CodeTypeOK,
		GasWanted: int64(tx.Gas()),
	}
}

//TODO: if the last height is not 0, we must run app.txEngine.Execute(&bi) here!!
func (app *App) InitChain(req abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	app.logger.Debug("enter init chain!, id=", req.ChainId)
	//fmt.Printf("InitChain!!!!!\n")

	if len(req.AppStateBytes) == 0 {
		panic("no AppStateBytes")
	}

	//fmt.Printf("appstate:%s\n", req.AppStateBytes)
	genesisData := GenesisData{}
	err := json.Unmarshal(req.AppStateBytes, &genesisData)
	if err != nil {
		panic(err)
	}

	app.createGenesisAccs(genesisData.Alloc)
	genesisValidators := genesisData.stakingValidators()

	if len(genesisValidators) == 0 {
		panic("no genesis validator in genesis.json")
	}

	//store all genesis validators even if it is inactive
	ctx := app.GetRunTxContext()
	staking.InitStakingInfoFromGenesis(ctx, genesisValidators)
	ctx.Close(true)

	var activeValidator []*stakingtypes.Validator
	for _, v := range genesisValidators {
		if uint256.NewInt().SetBytes(v.StakedCoins[:]).Cmp(staking.MinimumStakingAmount) >= 0 && !v.IsRetiring && v.VotingPower > 0 {
			activeValidator = append(activeValidator, v)
		}
	}
	app.currValidators = activeValidator
	valSet := make([]abcitypes.ValidatorUpdate, len(app.currValidators))
	for i, v := range app.currValidators {
		p, _ := cryptoenc.PubKeyToProto(ed25519.PubKey(v.Pubkey[:]))
		valSet[i] = abcitypes.ValidatorUpdate{
			PubKey: p,
			Power:  v.VotingPower,
		}
		fmt.Printf("inichain validator:%s\n", p.String())
	}

	params := &abcitypes.ConsensusParams{
		Block: &abcitypes.BlockParams{
			MaxBytes: BlockMaxBytes,
			MaxGas:   BlockMaxGas,
		},
	}
	return abcitypes.ResponseInitChain{
		ConsensusParams: params,
		Validators:      valSet,
	}
}

func (app *App) createGenesisAccs(alloc gethcore.GenesisAlloc) {
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
	//fmt.Printf("BeginBlock!!!!!!!!!!!!!!!\n")
	//app.randomPanic(5000, 7919)
	app.logger.Debug("enter begin block!")
	app.block = &types.Block{
		Number:    req.Header.Height,
		Timestamp: req.Header.Time.Unix(),
		Size:      int64(req.Size()),
	}
	copy(app.lastProposer[:], req.Header.ProposerAddress)
	fmt.Printf("PROPOSER: %s\n", gethcmn.Address(app.lastProposer).String())
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
	copy(app.block.Miner[:], req.Header.ProposerAddress)
	copy(app.block.Hash[:], req.Hash) // Just use tendermint's block hash
	copy(app.block.StateRoot[:], req.Header.AppHash)
	//TODO: slash req.ByzantineValidators
	app.currHeight = req.Header.Height
	// collect slash info, only double sign
	var addr [20]byte
	for _, val := range req.ByzantineValidators {
		//not check time, always slash
		if val.Type == abcitypes.EvidenceType_DUPLICATE_VOTE {
			copy(addr[:], val.Validator.Address)
			app.slashValidators = append(app.slashValidators, addr)
		}
	}
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
	//fmt.Printf("EndBlock!!!!!!!!!!!!!!!\n")
	app.logger.Debug("enter end block!")
	valSet := make([]abcitypes.ValidatorUpdate, len(app.validatorUpdate))
	for i, v := range app.validatorUpdate {
		p, _ := cryptoenc.PubKeyToProto(ed25519.PubKey(v.Pubkey[:]))
		valSet[i] = abcitypes.ValidatorUpdate{
			PubKey: p,
			Power:  v.VotingPower,
		}
		fmt.Printf("endblock validator pubkey=%s voting powe=%d\n\n", hex.EncodeToString(v.Pubkey[:]), v.VotingPower)
	}
	app.logger.Debug("leave end block!")
	return abcitypes.ResponseEndBlock{
		ValidatorUpdates: valSet,
	}
}

func (app *App) Commit() abcitypes.ResponseCommit {
	app.logger.Debug("enter commit!", "txs", app.txEngine.CollectTxsCount())
	app.mtx.Lock()

	ctx := app.GetRunTxContext()
	//distribute previous block gas gee
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
	if sysB.Cmp(&app.lastGasFee) < 0 {
		panic("system balance not enough!")
	}
	if app.txEngine.StandbyQLen() == 0 {
		// distribute extra balance to validators
		blockReward = *sysB
	}
	if !blockReward.IsZero() {
		err := ebp.SubSystemAccBalance(ctx, &blockReward)
		if err != nil {
			//todo: be careful
			panic(err)
		}
	}

	newValidators := staking.SlashAndReward(ctx, app.slashValidators, app.lastProposer, app.lastVoters, &blockReward)
	app.slashValidators = app.slashValidators[:0]

	select {
	case epoch := <-app.watcher.EpochChan:
		app.epochList = append(app.epochList, epoch)
		fmt.Printf("get new epoch in commit, its startHeight is:%d, len of epochList:%d\n", epoch.StartHeight, len(app.epochList))
	default:
		//fmt.Println("no new epoch")
	}
	if len(app.epochList) != 0 {
		//if app.block.Timestamp > app.epochList[0].EndTime+100*10*60 /*100 * 10min*/ {
		//epoch switch delay time should bigger than 10 mainnet block interval as of block finalization need
		if app.block.Timestamp > app.epochList[0].EndTime+3*10+10 /*100 second*/ {
			fmt.Printf("switch epoch, height:%d,blockTime:%d,endTime:%d\n", app.block.Number, app.block.Timestamp, app.epochList[0].EndTime)
			newValidators = staking.SwitchEpoch(ctx, app.epochList[0])
			app.epochList = app.epochList[1:]
		}
	}
	app.validatorUpdate = GetUpdateValidatorSet(app.currValidators, newValidators)
	for _, v := range app.validatorUpdate {
		fmt.Printf("validator update in commit: %s, voting power: %d\n", gethcmn.Address(v.Address).String(), v.VotingPower)
	}
	acc, newInfo := staking.LoadStakingAcc(ctx)
	newInfo.ValidatorsUpdate = app.validatorUpdate
	staking.SaveStakingInfo(ctx, acc, newInfo)
	ctx.Close(true)

	app.currValidators = newValidators

	app.touchedAddrs = app.txEngine.Prepare(app.reorderSeed, app.lastMinGasPrice)
	app.refresh()
	bi := app.syncBlockInfo()
	go app.postCommit(bi)
	app.logger.Debug("leave commit!")
	res := abcitypes.ResponseCommit{
		Data: append([]byte{}, app.block.StateRoot[:]...),
	}
	// prune tendermint history block and state every 100 blocks, maybe param it
	if app.retainBlocks > 0 && app.currHeight >= app.retainBlocks && (app.currHeight%100 == 0) {
		res.RetainHeight = app.currHeight - app.retainBlocks + 1
	}
	return res
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

func (app *App) GetLastGasUsed() uint64 {
	return app.lastGasUsed
}

func (app *App) refresh() {
	//close old
	app.checkTrunk.Close(false)

	ctx := app.GetRunTxContext()
	prevBlkInfo := ctx.GetCurrBlockBasicInfo()
	ctx.SetCurrBlockBasicInfo(app.block)
	//refresh lastMinGasPrice
	mGP := staking.LoadMinGasPrice(ctx, false)
	staking.SaveMinGasPrice(ctx, mGP, true)
	app.lastMinGasPrice = mGP
	ctx.Close(true)
	lastCacheSize := app.trunk.CacheSize() // predict the next truck's cache size with the last one
	app.trunk.Close(true)
	if prevBlkInfo != nil && prevBlkInfo.Number%PruneEveryN == 0 && prevBlkInfo.Number > app.numKeptBlocks {
		app.mads.PruneBeforeHeight(prevBlkInfo.Number - app.numKeptBlocks)
	}

	appHash := app.root.GetRootHash()
	copy(app.block.StateRoot[:], appHash)

	//jump block which prev height = 0
	if prevBlkInfo != nil {
		var wg *sync.WaitGroup
		app.sep206SenderSet, wg = app.getSep206SenderSet()
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
		for i, tx := range app.txEngine.CommittedTxs() {
			t := modbtypes.Tx{}
			copy(t.HashId[:], tx.Hash[:])
			copy(t.SrcAddr[:], tx.From[:])
			copy(t.DstAddr[:], tx.To[:])
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
		wg.Wait() // wait for getSep206SenderSet to finish its job
	}
	//make new
	app.recheckCounter = 0 // reset counter before counting the remained TXs which need rechecking
	app.lastProposer = app.block.Miner
	app.lastVoters = app.lastVoters[:0]
	app.root.SetHeight(app.currHeight + 1)
	app.trunk = app.root.GetTrunkStore(lastCacheSize).(*store.TrunkStore)
	app.checkTrunk = app.root.GetReadOnlyTrunkStore(DefaultTrunkCacheSize).(*store.TrunkStore)
	app.txEngine.SetContext(app.GetRunTxContext())
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

func (app *App) GetRpcContext() *types.Context {
	return app.GetContext(RpcMode)
}
func (app *App) GetRunTxContext() *types.Context {
	return app.GetContext(RunTxMode)
}
func (app *App) GetHistoryOnlyContext() *types.Context {
	return app.GetContext(HistoryOnlyMode)
}
func (app *App) GetCheckTxContext() *types.Context {
	return app.GetContext(CheckTxMode)
}

func (app *App) GetContext(mode ContextMode) *types.Context {
	c := types.NewContext(uint64(app.currHeight), nil, nil)
	if mode == CheckTxMode {
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
	ctx := app.GetRpcContext()
	defer ctx.Close(false)
	runner := &ebp.TxRunner{
		Ctx: ctx,
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

//nolint
func (app *App) WaitLock() {
	app.mtx.Lock()
	app.mtx.Unlock()
}

func (app *App) HistoryStore() modbtypes.DB {
	return app.historyStore
}

func (app *App) BlockNum() int64 {
	return app.block.Number
}

func (app *App) CurrValidators() []*stakingtypes.Validator {
	return app.currValidators
}
func (app *App) ValidatorUpdate() []*stakingtypes.Validator {
	return app.validatorUpdate
}

func (app *App) EpochChan() chan *stakingtypes.Epoch {
	return app.watcher.EpochChan
}

func (app *App) AddBlockFotTest(mdbBlock *modbtypes.Block) {
	app.historyStore.AddBlock(mdbBlock, -1)
	app.historyStore.AddBlock(nil, -1) // To Flush
	app.publishNewBlock(mdbBlock)
}

//nolint
// for ((i=10; i<80000; i+=50)); do RANDPANICHEIGHT=$i ./smartbchd start; done | tee a.log
func (app *App) randomPanic(baseNumber, primeNumber int64) {
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

var sep206Addr = gethcmn.HexToAddress("0x0000000000000000000000000000000000002711")

// Iterate over the txEngine.CommittedTxs, and find the senders who send native tokens to others
// through sep206 in the last block
func (app *App) getSep206SenderSet() (map[gethcmn.Address]struct{}, *sync.WaitGroup) {
	res := make(map[gethcmn.Address]struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for _, tx := range app.txEngine.CommittedTxs() {
			for _, log := range tx.Logs {
				if log.Address == sep206Addr && len(log.Topics) == 2 &&
					log.Topics[0] == modbtypes.TransferEvent {
					var addr gethcmn.Address
					copy(addr[:], log.Topics[1][12:]) // Topics[1] is the from-address
					res[addr] = struct{}{}
				}
			}
		}
		wg.Done()
	}()
	return res, &wg
}

func GetUpdateValidatorSet(currentValidators, newValidators []*stakingtypes.Validator) []*stakingtypes.Validator {
	if newValidators == nil {
		return nil
	}
	var currentSet = make(map[gethcmn.Address]bool)
	var newSet = make(map[gethcmn.Address]*stakingtypes.Validator)
	var updatedList = make([]*stakingtypes.Validator, 0, len(currentValidators))
	for _, v := range currentValidators {
		currentSet[v.Address] = true
	}
	for _, v := range newValidators {
		newSet[v.Address] = v
	}
	for _, v := range currentValidators {
		if newSet[v.Address] == nil {
			removedV := *v
			removedV.VotingPower = 0
			updatedList = append(updatedList, &removedV)
		} else if v.VotingPower != newSet[v.Address].VotingPower {
			updatedV := *newSet[v.Address]
			updatedList = append(updatedList, &updatedV)
			delete(newSet, v.Address)
		} else {
			delete(newSet, v.Address)
		}
	}
	for _, v := range newSet {
		addedV := *v
		updatedList = append(updatedList, &addedV)
	}
	sort.Slice(updatedList, func(i, j int) bool {
		return gethcmn.Address(updatedList[i].Address).String() < gethcmn.Address(updatedList[j].Address).String()
	})
	return updatedList
}
