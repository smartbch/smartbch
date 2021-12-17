package app

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
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
	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/moeingevm/types"

	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/seps"
	"github.com/smartbch/smartbch/staking"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
	"github.com/smartbch/smartbch/watcher"
)

var _ abcitypes.Application = (*App)(nil)

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

type App struct {
	mtx sync.Mutex

	//config
	config  *param.ChainConfig
	chainId *uint256.Int

	//store
	mads         *moeingads.MoeingADS
	root         *store.RootStore
	historyStore modbtypes.DB

	//refresh with block
	currHeight      int64
	trunk           *store.TrunkStore
	checkTrunk      *store.TrunkStore
	block           *types.Block
	blockInfo       atomic.Value // to store *types.BlockInfo
	slashValidators [][20]byte   // recorded in BeginBlock, used in Commit
	lastVoters      [][]byte     // recorded in BeginBlock, used in Commit
	lastProposer    [20]byte     // recorded in app.block in BeginBlock, copied here in refresh
	lastGasUsed     uint64       // recorded in last block's postCommit, used in current block's refresh
	lastGasRefund   uint256.Int  // recorded in last block's postCommit, used in current block's refresh
	lastGasFee      uint256.Int  // recorded in last block's postCommit, used in current block's refresh
	lastMinGasPrice uint64       // recorded in refresh, used in next block's CheckTx and Commit
	txid2sigMap     map[[32]byte][65]byte

	// feeds
	chainFeed event.Feed // For pub&sub new blocks
	logsFeed  event.Feed // For pub&sub new logs
	scope     event.SubscriptionScope

	//engine
	txEngine     ebp.TxExecutor
	reorderSeed  int64                   // recorded in BeginBlock, used in Commit
	touchedAddrs map[gethcmn.Address]int // recorded in Commint, used in next block's CheckTx

	//watcher
	watcher   *watcher.Watcher
	epochList []*stakingtypes.Epoch // caches the epochs collected by the watcher

	//util
	signer gethtypes.Signer
	logger log.Logger

	//genesis data
	currValidators  []*stakingtypes.Validator // it is needed to compute validatorUpdate
	validatorUpdate []*stakingtypes.Validator // tendermint wants to know validators whose voting power change

	//signature cache, cache ecrecovery's resulting sender addresses, to speed up checktx
	sigCache map[gethcmn.Hash]SenderAndHeight

	// the senders who send native tokens through sep206 in the last block
	sep206SenderSet map[gethcmn.Address]struct{} // recorded in refresh, used in CheckTx
	// it shows how many tx remains in the mempool after committing a new block
	recheckCounter int
}

// The value entry of signature cache. The Height helps in evicting old entries.
type SenderAndHeight struct {
	Sender gethcmn.Address
	Height int64
}

func NewApp(config *param.ChainConfig, chainId *uint256.Int, genesisWatcherHeight int64, logger log.Logger, forTest bool) *App {
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
	app.root, app.mads = createRootStore(config)
	app.historyStore = createHistoryStore(config, app.logger.With("module", "modb"))
	app.trunk = app.root.GetTrunkStore(config.AppConfig.TrunkCacheSize).(*store.TrunkStore)
	app.checkTrunk = app.root.GetReadOnlyTrunkStore(config.AppConfig.TrunkCacheSize).(*store.TrunkStore)

	/*------set engine------*/
	app.txEngine = ebp.NewEbpTxExec(
		param.EbpExeRoundCount,
		param.EbpRunnerNumber,
		param.EbpParallelNum,
		5000 /*not consensus relevant*/, app.signer)
	//ebp.AdjustGasUsed = false

	/*------set system contract------*/
	ctx := app.GetRunTxContext()
	//init PredefinedSystemContractExecutors before tx execute
	ebp.PredefinedSystemContractExecutor = staking.NewStakingContractExecutor(app.logger.With("module", "staking"))
	ebp.PredefinedSystemContractExecutor.Init(ctx)

	// We assign empty maps to them just to avoid accessing nil-maps.
	// Commit will assign meaningful contents to them
	app.txid2sigMap = make(map[[32]byte][65]byte)
	app.touchedAddrs = make(map[gethcmn.Address]int)
	app.sep206SenderSet = make(map[gethcmn.Address]struct{})

	/*------set refresh field------*/
	prevBlk := ctx.GetCurrBlockBasicInfo()
	if prevBlk != nil {
		app.block = prevBlk
		app.currHeight = app.block.Number
		app.lastProposer = app.block.Miner
	} else {
		app.block = &types.Block{}
	}

	app.root.SetHeight(app.currHeight)
	if app.currHeight != 0 {
		app.restartPostCommit()
	} else {
		app.txEngine.SetContext(app.GetRunTxContext())
	}

	/*------set stakingInfo------*/
	stakingInfo := staking.LoadStakingInfo(ctx)
	app.currValidators = stakingInfo.GetActiveValidators(staking.MinimumStakingAmount)
	app.validatorUpdate = stakingInfo.ValidatorsUpdate
	for _, val := range app.currValidators {
		app.logger.Debug(fmt.Sprintf("Load validator in NewApp: address(%s), pubkey(%s), votingPower(%d)",
			gethcmn.Address(val.Address).String(), ed25519.PubKey(val.Pubkey[:]).String(), val.VotingPower))
	}
	if stakingInfo.CurrEpochNum == 0 && stakingInfo.GenesisMainnetBlockHeight == 0 {
		stakingInfo.GenesisMainnetBlockHeight = genesisWatcherHeight
		staking.SaveStakingInfo(ctx, stakingInfo) // only executed at genesis
	}

	/*------set watcher------*/
	watcherLogger := app.logger.With("module", "watcher")
	client := watcher.NewParallelRpcClient(config.AppConfig.MainnetRPCUrl, config.AppConfig.MainnetRPCUsername, config.AppConfig.MainnetRPCPassword, watcherLogger)
	lastEpochEndHeight := stakingInfo.GenesisMainnetBlockHeight + param.StakingNumBlocksInEpoch*stakingInfo.CurrEpochNum
	app.watcher = watcher.NewWatcher(watcherLogger, lastEpochEndHeight, client, config.AppConfig.SmartBchRPCUrl, stakingInfo.CurrEpochNum, config.AppConfig.Speedup)
	app.logger.Debug(fmt.Sprintf("New watcher: mainnet url(%s), epochNum(%d), lastEpochEndHeight(%d), speedUp(%v)\n",
		config.AppConfig.MainnetRPCUrl, stakingInfo.CurrEpochNum, lastEpochEndHeight, config.AppConfig.Speedup))
	app.watcher.CheckSanity(forTest)
	catchupChan := make(chan bool, 1)
	go app.watcher.Run(catchupChan)
	<-catchupChan

	app.lastMinGasPrice = staking.LoadMinGasPrice(ctx, true)
	ctx.Close(true)
	return app
}

func createRootStore(config *param.ChainConfig) (*store.RootStore, *moeingads.MoeingADS) {
	first := [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
	last := [8]byte{255, 255, 255, 255, 255, 255, 255, 255}
	mads, err := moeingads.NewMoeingADS(config.AppConfig.AppDataPath, false, [][]byte{first[:], last[:]})
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

func (app *App) restartPostCommit() {
	app.txEngine.SetContext(app.GetRunTxContext())
	app.mtx.Lock()
	app.postCommit(app.syncBlockInfo())
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
	if _, ok := app.touchedAddrs[sender]; ok {
		// if the sender is touched, it is most likely to have an uncertain nonce, so we reject it
		return abcitypes.ResponseCheckTx{Code: HasPendingTx, Info: "still has pending transaction"}
	}
	if req.Type == abcitypes.CheckTxType_Recheck {
		// During rechecking, if the sender has not been touched or lose balance, the tx can pass
		if _, ok := app.sep206SenderSet[sender]; !ok {
			return abcitypes.ResponseCheckTx{
				Code:      abcitypes.CodeTypeOK,
				GasWanted: int64(tx.Gas()),
			}
		}
	}
	return app.checkTxWithContext(tx, sender)
}

func (app *App) checkTxWithContext(tx *gethtypes.Transaction, sender gethcmn.Address) abcitypes.ResponseCheckTx {
	ctx := app.GetCheckTxContext()
	dirty := false
	defer func(dirtyPtr *bool) {
		ctx.Close(*dirtyPtr)
	}(&dirty)

	intrGas, err := gethcore.IntrinsicGas(tx.Data(), nil, tx.To() == nil, true, true)
	if err != nil || tx.Gas() < intrGas {
		return abcitypes.ResponseCheckTx{Code: GasLimitTooSmall, Info: "gas limit too small"}
	}
	if tx.Gas() > param.MaxTxGasLimit {
		return abcitypes.ResponseCheckTx{Code: GasLimitInvalid, Info: "invalid gas limit"}
	}
	acc, err := ctx.CheckNonce(sender, tx.Nonce())
	if err != nil {
		return abcitypes.ResponseCheckTx{Code: AccountNonceMismatch, Info: "bad nonce: " + err.Error()}
	}
	gasPrice, _ := uint256.FromBig(tx.GasPrice())
	if gasPrice.Cmp(uint256.NewInt(app.lastMinGasPrice)) < 0 {
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
	genesisValidators := genesisData.stakingValidators()

	if len(genesisValidators) == 0 {
		panic("no genesis validator in genesis.json")
	}

	//store all genesis validators even if it is inactive
	ctx := app.GetRunTxContext()
	staking.AddGenesisValidatorsInStakingInfo(ctx, genesisValidators)
	ctx.Close(true)

	app.currValidators = stakingtypes.GetActiveValidators(genesisValidators, staking.MinimumStakingAmount)
	valSet := make([]abcitypes.ValidatorUpdate, len(app.currValidators))
	for i, v := range app.currValidators {
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
	//app.randomPanic(5000, 7919)
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
	// collect slash info, only double sign
	var addr [20]byte
	for _, val := range req.ByzantineValidators {
		//not check time, always slash
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
		app.txid2sigMap[tx.Hash()] = encodeVRS(tx)
	}
	return abcitypes.ResponseDeliverTx{Code: abcitypes.CodeTypeOK}
}

func encodeVRS(tx *gethtypes.Transaction) [65]byte {
	v, r, s := tx.RawSignatureValues()
	r256, _ := uint256.FromBig(r)
	s256, _ := uint256.FromBig(s)

	bs := [65]byte{}
	bs[0] = byte(v.Uint64())
	copy(bs[1:33], r256.PaddedBytes(32))
	copy(bs[33:65], s256.PaddedBytes(32))
	return bs
}
func DecodeVRS(bs [65]byte) (v, r, s *big.Int) {
	v = big.NewInt(int64(bs[0]))
	r = big.NewInt(0).SetBytes(bs[1:33])
	s = big.NewInt(0).SetBytes(bs[33:65])
	return
}

func (app *App) EndBlock(req abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	// app.validatorUpdate is recorded in Commit and used in the next block's EndBlock
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

	ctx := app.GetRunTxContext()
	//distribute previous block gas fee
	var blockReward = app.lastGasFee
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

	app.updateValidatorsAndStakingInfo(ctx, &blockReward)
	ctx.Close(true)

	app.touchedAddrs = app.txEngine.Prepare(app.reorderSeed, 0, param.MaxTxGasLimit)
	app.refresh()
	bi := app.syncBlockInfo()
	if bi == nil {
		app.logger.Debug("nil blockInfo after sync")
	} else {
		app.logger.Debug(fmt.Sprintf("blockInfo: hash:%s, height:%d", gethcmn.Hash(bi.Hash).Hex(), bi.Number))
	}
	go app.postCommit(bi)
	res := abcitypes.ResponseCommit{
		Data: append([]byte{}, app.block.StateRoot[:]...),
	}
	// prune tendermint history block and state every ChangeRetainEveryN blocks
	if app.config.AppConfig.RetainBlocks > 0 && app.currHeight >= app.config.AppConfig.RetainBlocks && (app.currHeight%app.config.AppConfig.ChangeRetainEveryN == 0) {
		res.RetainHeight = app.currHeight - app.config.AppConfig.RetainBlocks + 1
	}
	return res
}

func (app *App) updateValidatorsAndStakingInfo(ctx *types.Context, blockReward *uint256.Int) {
	newValidators := staking.SlashAndReward(ctx, app.slashValidators, app.block.Miner, app.lastProposer, app.lastVoters, blockReward)
	app.slashValidators = app.slashValidators[:0]

	select {
	case epoch := <-app.watcher.EpochChan:
		app.epochList = append(app.epochList, epoch)
		app.logger.Debug(fmt.Sprintf("Get new epoch, epochNum(%d), startHeight(%d), epochListLens(%d)",
			epoch.Number, epoch.StartHeight, len(app.epochList)))
	default:
	}
	if len(app.epochList) != 0 {
		//epoch switch delay time should bigger than 10 mainnet block interval as of block finalization need
		if app.block.Timestamp > app.epochList[0].EndTime+param.StakingEpochSwitchDelay {
			app.logger.Debug(fmt.Sprintf("Switch epoch at block(%d), eppchNum(%d)",
				app.block.Number, app.epochList[0].Number))
			newValidators = staking.SwitchEpoch(ctx, app.epochList[0], app.logger,
				param.StakingMinVotingPercentPerEpoch, param.StakingMinVotingPubKeysPercentPerEpoch)
			app.epochList = app.epochList[1:]
		}
	}

	app.validatorUpdate = staking.GetUpdateValidatorSet(app.currValidators, newValidators)
	for _, v := range app.validatorUpdate {
		app.logger.Debug(fmt.Sprintf("Updated validator in commit: address(%s), pubkey(%s), voting power: %d",
			gethcmn.Address(v.Address).String(), ed25519.PubKey(v.Pubkey[:]), v.VotingPower))
	}
	newInfo := staking.LoadStakingInfo(ctx)
	newInfo.ValidatorsUpdate = app.validatorUpdate
	staking.SaveStakingInfo(ctx, newInfo)
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
	return bi
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

func (app *App) refresh() {
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
	ctx.Close(true)

	lastCacheSize := app.trunk.CacheSize() // predict the next truck's cache size with the last one
	app.trunk.Close(true)                  //write cached KVs back to app.root
	if prevBlkInfo != nil && prevBlkInfo.Number%app.config.AppConfig.PruneEveryN == 0 && prevBlkInfo.Number > app.config.AppConfig.NumKeptBlocks {
		app.mads.PruneBeforeHeight(prevBlkInfo.Number - app.config.AppConfig.NumKeptBlocks)
	}

	appHash := app.root.GetRootHash()
	copy(app.block.StateRoot[:], appHash)

	//jump block which prev height = 0
	if prevBlkInfo != nil {
		var wg sync.WaitGroup
		app.sep206SenderSet = app.getSep206SenderSet(&wg)
		//use current block commit app hash as prev history block stateRoot
		prevBlkInfo.StateRoot = app.block.StateRoot
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
		if app.config.AppConfig.NumKeptBlocksInMoDB > 0 && app.currHeight > app.config.AppConfig.NumKeptBlocksInMoDB {
			app.historyStore.AddBlock(&prevBlk4MoDB, app.currHeight-app.config.AppConfig.NumKeptBlocksInMoDB, app.txid2sigMap)
		} else {
			app.historyStore.AddBlock(&prevBlk4MoDB, -1, app.txid2sigMap) // do not prune moeingdb
		}
		app.txid2sigMap = make(map[[32]byte][65]byte)
		app.publishNewBlock(&prevBlk4MoDB)
		wg.Wait() // wait for getSep206SenderSet to finish its job
	}
	//make new
	app.recheckCounter = 0 // reset counter before counting the remained TXs which need rechecking
	app.lastProposer = app.block.Miner
	app.lastVoters = app.lastVoters[:0]
	app.root.SetHeight(app.currHeight)
	app.trunk = app.root.GetTrunkStore(lastCacheSize).(*store.TrunkStore)
	app.checkTrunk = app.root.GetReadOnlyTrunkStore(app.config.AppConfig.TrunkCacheSize).(*store.TrunkStore)
	app.txEngine.SetContext(app.GetRunTxContext())
}

// Iterate over the txEngine.CommittedTxs, and find the senders who send native tokens to others
// through sep206 in the last block
func (app *App) getSep206SenderSet(wg *sync.WaitGroup) map[gethcmn.Address]struct{} {
	res := make(map[gethcmn.Address]struct{})
	wg.Add(1)
	go func() {
		for _, tx := range app.txEngine.CommittedTxs() {
			for _, log := range tx.Logs {
				if log.Address == seps.SEP206Addr && len(log.Topics) == 3 &&
					log.Topics[0] == modbtypes.TransferEvent {
					var addr gethcmn.Address
					copy(addr[:], log.Topics[1][12:]) // Topics[1] is the from-address
					res[addr] = struct{}{}
				}
			}
		}
		wg.Done()
	}()
	return res
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
	c := types.NewContext(uint64(app.currHeight), nil, nil)
	r := rabbit.NewReadOnlyRabbitStore(app.root)
	c = c.WithRbt(&r)
	c = c.WithDb(app.historyStore)
	return c
}
func (app *App) GetRunTxContext() *types.Context {
	c := types.NewContext(uint64(app.currHeight), nil, nil)
	r := rabbit.NewRabbitStore(app.trunk)
	c = c.WithRbt(&r)
	c = c.WithDb(app.historyStore)
	return c
}
func (app *App) GetHistoryOnlyContext() *types.Context {
	c := types.NewContext(uint64(app.currHeight), nil, nil)
	c = c.WithDb(app.historyStore)
	return c
}
func (app *App) GetCheckTxContext() *types.Context {
	c := types.NewContext(uint64(app.currHeight), nil, nil)
	r := rabbit.NewRabbitStore(app.checkTrunk)
	c = c.WithRbt(&r)
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
	minGasPrice := staking.LoadMinGasPrice(ctx, false)
	lastMinGasPrice := staking.LoadMinGasPrice(ctx, true)
	return newValidatorsInfo(app.currValidators, stakingInfo, minGasPrice, lastMinGasPrice)
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
