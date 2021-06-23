package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"math/big"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/node"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"

	"github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/staking"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

var _ BackendService = &apiBackend{}

const (
	// Ethereum Wire Protocol
	// https://github.com/ethereum/devp2p/blob/master/caps/eth.md
	protocolVersion = 63
)

type apiBackend struct {
	//extRPCEnabled bool
	node *node.Node
	app  *app.App
	//gpo *gasprice.Oracle

	//chainSideFeed event.Feed
	//chainHeadFeed event.Feed
	//blockProcFeed event.Feed
	txFeed event.Feed
	//logsFeed   event.Feed
	rmLogsFeed event.Feed
	//pendingLogsFeed event.Feed
}

func NewBackend(node *node.Node, app *app.App) BackendService {
	return &apiBackend{
		node: node,
		app:  app,
	}
}

func (backend *apiBackend) ChainId() *big.Int {
	return backend.app.ChainID().ToBig()
}

func (backend *apiBackend) GetStorageAt(address common.Address, key string, blockNumber int64) []byte {
	if blockNumber != int64(rpc.LatestBlockNumber) {
		// TODO: not supported yet
		return nil
	}

	ctx := backend.app.GetRpcContext()
	defer ctx.Close(false)

	acc := ctx.GetAccount(address)
	if acc == nil {
		return nil
	}
	return ctx.GetStorageAt(acc.Sequence(), key)
}

func (backend *apiBackend) GetCode(contract common.Address, blockNumber int64) (bytecode []byte, codeHash []byte) {
	if blockNumber != int64(rpc.LatestBlockNumber) {
		// TODO: not supported yet
		return
	}

	ctx := backend.app.GetRpcContext()
	defer ctx.Close(false)

	info := ctx.GetCode(contract)
	if info != nil {
		bytecode = info.BytecodeSlice()
		codeHash = info.CodeHashSlice()
	}
	return
}

func (backend *apiBackend) GetBalance(owner common.Address, height int64) (*big.Int, error) {
	ctx := backend.app.GetRpcContext()
	defer ctx.Close(false)
	b, err := ctx.GetBalance(owner, height)
	if err != nil {
		return nil, err
	}
	return b.ToBig(), nil
}

func (backend *apiBackend) GetNonce(address common.Address) (uint64, error) {
	ctx := backend.app.GetRpcContext()
	defer ctx.Close(false)
	if acc := ctx.GetAccount(address); acc != nil {
		return acc.Nonce(), nil
	}

	return 0, types.ErrAccNotFound
}

func (backend *apiBackend) GetTransaction(txHash common.Hash) (tx *types.Transaction, blockHash common.Hash, blockNumber uint64, blockIndex uint64, err error) {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)

	if tx, err = ctx.GetTxByHash(txHash); err != nil {
		return
	}
	if tx != nil {
		blockHash = tx.BlockHash
		blockNumber = uint64(tx.BlockNumber)
		blockIndex = uint64(tx.TransactionIndex)
	} else {
		err = errors.New("tx with specific hash not exist")
	}
	return
}

func (backend *apiBackend) BlockByHash(hash common.Hash) (*types.Block, error) {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)
	block, err := ctx.GetBlockByHash(hash)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (backend *apiBackend) BlockByNumber(number int64) (*types.Block, error) {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)
	return ctx.GetBlockByHeight(uint64(number))
}

func (backend *apiBackend) ProtocolVersion() int {
	return protocolVersion
}

func (backend *apiBackend) LatestHeight() int64 {
	appCtx := backend.app.GetHistoryOnlyContext()
	return appCtx.GetLatestHeight()
}

func (backend *apiBackend) CurrentBlock() (*types.Block, error) {
	appCtx := backend.app.GetHistoryOnlyContext()
	block, err := appCtx.GetBlockByHeight(uint64(appCtx.GetLatestHeight()))
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (backend *apiBackend) SendRawTx(signedTx []byte) (common.Hash, error) {
	return backend.broadcastTxSync(signedTx)
}

func (backend *apiBackend) broadcastTxSync(tx tmtypes.Tx) (common.Hash, error) {
	resCh := make(chan *abci.Response, 1)
	err := backend.node.Mempool().CheckTx(tx, func(res *abci.Response) {
		resCh <- res
	}, mempool.TxInfo{})
	if err != nil {
		return common.Hash{}, err
	}
	res := <-resCh
	r := res.GetCheckTx()
	if r.Code != abci.CodeTypeOK {
		return common.Hash{}, errors.New(r.String())
	}
	return common.BytesToHash(tx.Hash()), nil
}

func (backend *apiBackend) Call(tx *gethtypes.Transaction, sender common.Address) (statusCode int, retData []byte) {
	runner, _ := backend.app.RunTxForRpc(tx, sender, false)
	return runner.Status, runner.OutData
}

func (backend *apiBackend) EstimateGas(tx *gethtypes.Transaction, sender common.Address) (statusCode int, retData []byte, gas int64) {
	runner, gas := backend.app.RunTxForRpc(tx, sender, true)
	return runner.Status, runner.OutData, gas
}

func (backend *apiBackend) QueryLogs(addresses []common.Address, topics [][]common.Hash, startHeight, endHeight uint32, filter types.FilterFunc) ([]types.Log, error) {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)

	return ctx.QueryLogs(addresses, topics, startHeight, endHeight, filter)
}

func (backend *apiBackend) QueryTxBySrc(addr common.Address, startHeight, endHeight, limit uint32) (tx []*types.Transaction, err error) {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)
	return ctx.QueryTxBySrc(addr, startHeight, endHeight, limit)
}

func (backend *apiBackend) QueryTxByDst(addr common.Address, startHeight, endHeight, limit uint32) (tx []*types.Transaction, err error) {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)
	return ctx.QueryTxByDst(addr, startHeight, endHeight, limit)
}

func (backend *apiBackend) QueryTxByAddr(addr common.Address, startHeight, endHeight, limit uint32) (tx []*types.Transaction, err error) {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)
	return ctx.QueryTxByAddr(addr, startHeight, endHeight, limit)
}

func (backend *apiBackend) SbchQueryLogs(addr common.Address, topics []common.Hash, startHeight, endHeight, limit uint32) ([]types.Log, error) {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)

	return ctx.BasicQueryLogs(addr, topics, startHeight, endHeight, limit)
}

func (backend *apiBackend) GetTxListByHeight(height uint32) (tx []*types.Transaction, err error) {
	return backend.GetTxListByHeightWithRange(height, 0, math.MaxInt32)
}
func (backend *apiBackend) GetTxListByHeightWithRange(height uint32, start, end int) (tx []*types.Transaction, err error) {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)

	return ctx.GetTxListByHeightWithRange(height, start, end)
}

func (backend *apiBackend) GetToAddressCount(addr common.Address) int64 {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)

	return ctx.GetToAddressCount(addr)
}
func (backend *apiBackend) GetFromAddressCount(addr common.Address) int64 {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)

	return ctx.GetFromAddressCount(addr)
}
func (backend *apiBackend) GetSep20ToAddressCount(contract common.Address, addr common.Address) int64 {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)

	return ctx.GetSep20ToAddressCount(contract, addr)
}
func (backend *apiBackend) GetSep20FromAddressCount(contract common.Address, addr common.Address) int64 {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)

	return ctx.GetSep20FromAddressCount(contract, addr)
}

//[start, end)
func (backend *apiBackend) GetEpochs(start, end uint64) ([]*stakingtypes.Epoch, error) {
	ctx := backend.app.GetHistoryOnlyContext()
	defer ctx.Close(false)

	_, info := staking.LoadStakingAcc(ctx)
	length := uint64(len(info.Epochs))
	if end > length {
		end = length
	}
	if start >= end {
		return nil, errors.New("invalid start")
	}
	return info.Epochs[start:end], nil
}

func (backend *apiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	if blockNr == rpc.LatestBlockNumber {
		blockNr = rpc.BlockNumber(backend.app.GetLatestBlockNum())
	}

	appCtx := backend.app.GetHistoryOnlyContext()
	defer appCtx.Close(false)
	block, err := appCtx.GetBlockByHeight(uint64(blockNr))
	if err != nil {
		return nil, nil
	}
	return &types.Header{
		Number:    uint64(block.Number),
		BlockHash: block.Hash,
		Bloom:     block.LogsBloom,
	}, nil
}
func (backend *apiBackend) HeaderByHash(ctx context.Context, blockHash common.Hash) (*types.Header, error) {
	appCtx := backend.app.GetHistoryOnlyContext()
	defer appCtx.Close(false)
	block, err := appCtx.GetBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	return &types.Header{
		Number:    uint64(block.Number),
		BlockHash: block.Hash,
	}, nil
}
func (backend *apiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (gethtypes.Receipts, error) {
	appCtx := backend.app.GetHistoryOnlyContext()
	defer appCtx.Close(false)

	receipts := make([]*gethtypes.Receipt, 0, 8)

	// TODO: query receipts
	//block, err := appCtx.GetBlockByHash(blockHash)
	//if err == nil && block != nil {
	//	for _, txHash := range block.Transactions {
	//		tx, err := appCtx.GetTxByHash(txHash)
	//		if err == nil && tx != nil {
	//			receipts = append(receipts, toGethReceipt(tx))
	//		}
	//	}
	//}
	return receipts, nil
}

func (backend *apiBackend) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*gethtypes.Log, error) {
	appCtx := backend.app.GetHistoryOnlyContext()
	defer appCtx.Close(false)

	logs := make([][]*gethtypes.Log, 0, 8)

	block, err := appCtx.GetBlockByHash(blockHash)
	if err == nil && block != nil {
		for _, txHash := range block.Transactions {
			tx, err := appCtx.GetTxByHash(txHash)
			if err == nil && tx != nil {
				txLogs := types.ToGethLogs(tx.Logs)
				// fix log.TxHash
				for _, txLog := range txLogs {
					txLog.TxHash = tx.Hash
				}
				logs = append(logs, txLogs)
			}
		}
	}

	return logs, nil
}

func (backend *apiBackend) SubscribeChainEvent(ch chan<- types.ChainEvent) event.Subscription {
	return backend.app.SubscribeChainEvent(ch)
}
func (backend *apiBackend) SubscribeLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription {
	return backend.app.SubscribeLogsEvent(ch)
}
func (backend *apiBackend) SubscribeNewTxsEvent(ch chan<- gethcore.NewTxsEvent) event.Subscription {
	return backend.txFeed.Subscribe(ch)
}
func (backend *apiBackend) SubscribeRemovedLogsEvent(ch chan<- gethcore.RemovedLogsEvent) event.Subscription {
	return backend.rmLogsFeed.Subscribe(ch)
}

//func (b2 *apiBackend) SubscribePendingLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription {
//	return b2.pendingLogsFeed.Subscribe(ch)
//}

func (backend *apiBackend) BloomStatus() (uint64, uint64) {
	return 4096, 0 // TODO: this is temporary implementation
}
func (backend *apiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	panic("implement me")
}

/*-----------------------tendermint info----------------------------*/
type Info struct {
	IsValidator     bool            `json:"is_validator"`
	ValidatorIndex  int64           `json:"validator_index"`
	Height          int64           `json:"height"`
	Seed            string          `json:"seed"`
	ConsensusPubKey crypto.PubKey   `json:"consensus_pub_key"`
	AppState        json.RawMessage `json:"app_state"`
}

func (backend *apiBackend) NodeInfo() Info {
	i := Info{}
	i.Height = backend.node.BlockStore().Height()
	address, _ := backend.node.NodeInfo().NetAddress()
	if address != nil {
		i.Seed = address.String()
	}
	i.ConsensusPubKey, _ = backend.node.PrivValidator().GetPubKey()
	i.AppState = backend.node.GenesisDoc().AppState
	genesisData := app.GenesisData{}
	err := json.Unmarshal(i.AppState, &genesisData)
	if err == nil {
		for k, v := range genesisData.Validators {
			if bytes.Equal(v.Pubkey[:], i.ConsensusPubKey.Bytes()) {
				i.IsValidator = true
				i.ValidatorIndex = int64(k)
			}
		}
	}
	return i
}

type ValidatorsInfo struct {
	// StakingInfo
	GenesisMainnetBlockHeight int64            `json:"genesisMainnetBlockHeight"`
	CurrEpochNum              int64            `json:"currEpochNum"`
	Validators                []*app.Validator `json:"validators"`
	ValidatorsUpdate          []*app.Validator `json:"validatorsUpdate"`
	PendingRewards            []*PendingReward `json:"pendingRewards"`

	// App
	CurrValidators []*app.Validator `json:"currValidators"`
}

type PendingReward struct {
	Address  common.Address `json:"address"`
	EpochNum int64          `json:"epochNum"`
	Amount   string         `json:"amount"`
}

func (backend *apiBackend) ValidatorsInfo() ValidatorsInfo {
	ctx := backend.app.GetRpcContext()
	defer ctx.Close(false)

	_, stakingInfo := staking.LoadStakingAcc(ctx)
	currValidators := backend.app.CurrValidators()
	info := ValidatorsInfo{
		GenesisMainnetBlockHeight: stakingInfo.GenesisMainnetBlockHeight,
		CurrEpochNum:              stakingInfo.CurrEpochNum,
		Validators:                app.FromStakingValidators(stakingInfo.Validators),
		ValidatorsUpdate:          app.FromStakingValidators(stakingInfo.ValidatorsUpdate),
		CurrValidators:            app.FromStakingValidators(currValidators),
	}

	info.PendingRewards = make([]*PendingReward, len(stakingInfo.PendingRewards))
	for i, pr := range stakingInfo.PendingRewards {
		info.PendingRewards[i] = &PendingReward{
			Address:  pr.Address,
			EpochNum: pr.EpochNum,
			Amount:   uint256.NewInt().SetBytes(pr.Amount[:]).String(),
		}
	}

	return info
}
