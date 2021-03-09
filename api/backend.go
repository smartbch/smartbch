package api

import (
	"context"
	"errors"
	"math/big"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/node"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/moeing-chain/MoeingEVM/ebp"
	"github.com/moeing-chain/MoeingEVM/types"
	"github.com/moeing-chain/moeing-chain/app"
	"github.com/moeing-chain/moeing-chain/param"
)

var _ BackendService = &moeingAPIBackend{}

const (
	// Ethereum Wire Protocol
	// https://github.com/ethereum/devp2p/blob/master/caps/eth.md
	protocolVersion = 63
)

type moeingAPIBackend struct {
	//extRPCEnabled bool
	node *node.Node
	app  *app.App
	//gpo *gasprice.Oracle

	chainSideFeed event.Feed
	chainHeadFeed event.Feed
	blockProcFeed event.Feed
	txFeed        event.Feed
	logsFeed      event.Feed
	rmLogsFeed    event.Feed
	//pendingLogsFeed event.Feed
}

func NewBackend(node *node.Node, app *app.App) BackendService {
	return &moeingAPIBackend{
		node: node,
		app:  app,
	}
}

//func (backend *moeingAPIBackend) GetLogs(blockHash common.Hash) (logs [][]types.Log, err error) {
//	ctx := backend.app.GetContext(app.RpcMode)
//	defer ctx.Close(false)
//
//	block, err := ctx.GetBlockByHash(blockHash)
//	if err == nil && block != nil {
//		for _, txHash := range block.Transactions {
//			tx, err := ctx.GetTxByHash(txHash)
//			if err == nil && tx != nil {
//				logs = append(logs, tx.Logs)
//			}
//		}
//	}
//	return
//}

//func (m moeingAPIBackend) GetReceipts(hash common.Hash) (*types.Transaction, error) {
//	tx, _, _, _, err := m.GetTransaction(hash)
//	return tx, err
//}

func (backend *moeingAPIBackend) ChainId() *big.Int {
	return backend.app.ChainID().ToBig()
}

func (backend *moeingAPIBackend) GetStorageAt(address common.Address, key string, blockNumber uint64) []byte {
	ctx := backend.app.GetContext(app.RpcMode)
	defer ctx.Close(false)

	acc := ctx.GetAccount(address)
	if acc == nil {
		return nil
	}
	if blockNumber == 0 {
		return ctx.GetStorageAt(acc.Sequence(), key)
	}
	return nil
}

func (backend *moeingAPIBackend) GetCode(contract common.Address) (bytecode []byte, codeHash []byte) {
	ctx := backend.app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	info := ctx.GetCode(contract)
	if info != nil {
		bytecode = info.BytecodeSlice()
		codeHash = info.CodeHashSlice()
	}
	return
}

func (backend *moeingAPIBackend) GetBalance(owner common.Address, height int64) (*big.Int, error) {
	ctx := backend.app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	b, err := ctx.GetBalance(owner, height)
	if err != nil {
		return nil, err
	}
	return b.ToBig(), nil
}

func (backend *moeingAPIBackend) GetNonce(address common.Address) (uint64, error) {
	ctx := backend.app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	if acc := ctx.GetAccount(address); acc != nil {
		return acc.Nonce(), nil
	}

	return 0, types.ErrAccNotFound
}

func (backend *moeingAPIBackend) GetTransaction(txHash common.Hash) (tx *types.Transaction, blockHash common.Hash, blockNumber uint64, blockIndex uint64, err error) {
	ctx := backend.app.GetContext(app.RpcMode)
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

func (backend *moeingAPIBackend) BlockByHash(hash common.Hash) (*types.Block, error) {
	ctx := backend.app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	block, err := ctx.GetBlockByHash(hash)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (backend *moeingAPIBackend) BlockByNumber(number int64) (*types.Block, error) {
	ctx := backend.app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	return ctx.GetBlockByHeight(uint64(number))
}

func (backend *moeingAPIBackend) ProtocolVersion() int {
	return protocolVersion
}

func (backend *moeingAPIBackend) CurrentBlock() *types.Block {
	return backend.app.CurrentBlock()
}

func (backend *moeingAPIBackend) ChainConfig() *param.ChainConfig {
	return backend.app.Config
}

func (backend *moeingAPIBackend) SendTx(signedTx *types.Transaction) error {
	panic("implement me")
}

func (backend *moeingAPIBackend) SendRawTx(signedTx []byte) (common.Hash, error) {
	return backend.broadcastTxSync(signedTx)
}

func (backend *moeingAPIBackend) broadcastTxSync(tx tmtypes.Tx) (common.Hash, error) {
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

func (backend *moeingAPIBackend) Call(tx *gethtypes.Transaction, sender common.Address) (statusCode int, statusStr string, retData []byte) {
	runner, _ := backend.app.RunTxForRpc(tx, sender, false)
	return runner.Status, ebp.StatusToStr(runner.Status), runner.OutData
}

func (backend *moeingAPIBackend) EstimateGas(tx *gethtypes.Transaction, sender common.Address) (statusCode int, statusStr string, gas int64) {
	runner, gas := backend.app.RunTxForRpc(tx, sender, true)
	return runner.Status, ebp.StatusToStr(runner.Status), gas
}

func (backend *moeingAPIBackend) QueryLogs(addresses []common.Address, topics [][]common.Hash, startHeight, endHeight uint32) ([]types.Log, error) {
	ctx := backend.app.GetContext(app.RpcMode)
	defer ctx.Close(false)

	return ctx.QueryLogs(addresses, topics, startHeight, endHeight)
}

func (backend *moeingAPIBackend) QueryTxBySrc(addr common.Address, startHeight, endHeight uint32) (tx []*types.Transaction, err error) {
	ctx := backend.app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	return ctx.QueryTxBySrc(addr, startHeight, endHeight)
}

func (backend *moeingAPIBackend) QueryTxByDst(addr common.Address, startHeight, endHeight uint32) (tx []*types.Transaction, err error) {
	ctx := backend.app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	return ctx.QueryTxByDst(addr, startHeight, endHeight)
}

func (backend *moeingAPIBackend) QueryTxByAddr(addr common.Address, startHeight, endHeight uint32) (tx []*types.Transaction, err error) {
	ctx := backend.app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	return ctx.QueryTxByAddr(addr, startHeight, endHeight)
}

func (backend *moeingAPIBackend) MoeQueryLogs(addr common.Address, topics []common.Hash, startHeight, endHeight uint32) ([]types.Log, error) {
	ctx := backend.app.GetContext(app.RpcMode)
	defer ctx.Close(false)

	return ctx.BasicQueryLogs(addr, topics, startHeight, endHeight)
}

func (backend *moeingAPIBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	if blockNr == rpc.LatestBlockNumber {
		blockNr = rpc.BlockNumber(backend.app.GetLatestBlockNum())
	}

	appCtx := backend.app.GetContext(app.RpcMode)
	defer appCtx.Close(false)
	block, err := appCtx.GetBlockByHeight(uint64(blockNr))
	if err != nil {
		return nil, nil
	}
	return &types.Header{
		Number:    uint64(block.Number),
		BlockHash: block.Hash,
		Bloom:     block.LogsBloom,
		// TODO: fill more fields
	}, nil
}
func (backend *moeingAPIBackend) HeaderByHash(ctx context.Context, blockHash common.Hash) (*types.Header, error) {
	appCtx := backend.app.GetContext(app.RpcMode)
	defer appCtx.Close(false)
	block, err := appCtx.GetBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	return &types.Header{
		Number:    uint64(block.Number),
		BlockHash: block.Hash,
		// TODO: fill more fields
	}, nil
}
func (backend *moeingAPIBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (gethtypes.Receipts, error) {
	appCtx := backend.app.GetContext(app.RpcMode)
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

//func toGethReceipt(tx *types.Transaction) *gethtypes.Receipt {
//	return &gethtypes.Receipt{
//		// TODO
//	}
//}

func (backend *moeingAPIBackend) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*gethtypes.Log, error) {
	appCtx := backend.app.GetContext(app.RpcMode)
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

func (backend *moeingAPIBackend) SubscribeChainEvent(ch chan<- types.ChainEvent) event.Subscription {
	return backend.app.SubscribeChainEvent(ch)
}
func (backend *moeingAPIBackend) SubscribeLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription {
	return backend.app.SubscribeLogsEvent(ch)
}
func (backend *moeingAPIBackend) SubscribeNewTxsEvent(ch chan<- gethcore.NewTxsEvent) event.Subscription {
	return backend.txFeed.Subscribe(ch)
}
func (backend *moeingAPIBackend) SubscribeRemovedLogsEvent(ch chan<- gethcore.RemovedLogsEvent) event.Subscription {
	return backend.rmLogsFeed.Subscribe(ch)
}

//func (b2 *moeingAPIBackend) SubscribePendingLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription {
//	return b2.pendingLogsFeed.Subscribe(ch)
//}

func (backend *moeingAPIBackend) BloomStatus() (uint64, uint64) {
	return 4096, 0 // TODO: this is temporary implementation
}
func (backend *moeingAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	panic("implement me")
}
