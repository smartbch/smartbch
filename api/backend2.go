package api

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/moeing-chain/moeing-chain/app"
	"github.com/moeing-chain/MoeingEVM/types"
)

// TODO: merge with Backend
type GethBackend struct {
	app *app.App

	chainSideFeed event.Feed
	chainHeadFeed event.Feed
	blockProcFeed event.Feed
	txFeed        event.Feed
	logsFeed      event.Feed
	rmLogsFeed    event.Feed
	//pendingLogsFeed event.Feed
}

func NewGethBackend(app *app.App) *GethBackend {
	return &GethBackend{app: app}
}

func (b2 *GethBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	if blockNr == rpc.LatestBlockNumber {
		blockNr = rpc.BlockNumber(b2.app.GetLatestBlockNum())
	}

	appCtx := b2.app.GetContext(app.RpcMode)
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
func (b2 *GethBackend) HeaderByHash(ctx context.Context, blockHash common.Hash) (*types.Header, error) {
	appCtx := b2.app.GetContext(app.RpcMode)
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
func (b2 *GethBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (gethtypes.Receipts, error) {
	appCtx := b2.app.GetContext(app.RpcMode)
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

func (b2 *GethBackend) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*gethtypes.Log, error) {
	appCtx := b2.app.GetContext(app.RpcMode)
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

func (b2 *GethBackend) SubscribeChainEvent(ch chan<- types.ChainEvent) event.Subscription {
	return b2.app.SubscribeChainEvent(ch)
}
func (b2 *GethBackend) SubscribeLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription {
	return b2.app.SubscribeLogsEvent(ch)
}
func (b2 *GethBackend) SubscribeNewTxsEvent(ch chan<- gethcore.NewTxsEvent) event.Subscription {
	return b2.txFeed.Subscribe(ch)
}
func (b2 *GethBackend) SubscribeRemovedLogsEvent(ch chan<- gethcore.RemovedLogsEvent) event.Subscription {
	return b2.rmLogsFeed.Subscribe(ch)
}

//func (b2 *GethBackend) SubscribePendingLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription {
//	return b2.pendingLogsFeed.Subscribe(ch)
//}

func (b2 *GethBackend) BloomStatus() (uint64, uint64) {
	return 4096, 0 // TODO: this is temporary implementation
}
func (b2 *GethBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	panic("implement me")
}

func blockToGethHeader(block *types.Block) *gethtypes.Header {
	return &gethtypes.Header{
		ParentHash: block.ParentHash,
		Root:       block.StateRoot,
		TxHash:     block.TransactionsRoot,
		Number:     big.NewInt(block.Number),
		GasUsed:    block.GasUsed,
		// TODO: other fields
	}
}
