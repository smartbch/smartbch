package api

import (
	"context"
	"github.com/smartbch/smartbch/app"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"

	motypes "github.com/smartbch/moeingevm/types"
)

type CallDetail struct {
	Status                 int
	GasUsed                uint64
	OutData                []byte
	Logs                   []motypes.EvmLog
	CreatedContractAddress common.Address
	InternalTxCalls        []motypes.InternalTxCall
	InternalTxReturns      []motypes.InternalTxReturn
	RwLists                *motypes.ReadWriteLists
}

type FilterService interface {
	HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*motypes.Header, error)
	HeaderByHash(ctx context.Context, blockHash common.Hash) (*motypes.Header, error)
	GetReceipts(ctx context.Context, blockNum uint64) (gethtypes.Receipts, error)
	GetLogs(ctx context.Context, blockHash common.Hash) ([][]*gethtypes.Log, error)

	SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription
	SubscribeChainEvent(ch chan<- motypes.ChainEvent) event.Subscription
	SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription
	SubscribeLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription
	//SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription

	BloomStatus() (uint64, uint64)
	ServiceFilter(ctx context.Context, session *bloombits.MatcherSession)
}

type BackendService interface {
	FilterService
	ProtocolVersion() int
	ChainId() *big.Int
	LatestHeight() int64
	CurrentBlock() (*motypes.Block, error)
	BlockByNumber(number int64) (*motypes.Block, error)
	BlockByHash(hash common.Hash) (*motypes.Block, error)
	SendRawTx(signedTx []byte) (common.Hash, error)
	GetTransaction(txHash common.Hash) (tx *motypes.Transaction, sig [65]byte, err error)
	GetNonce(address common.Address, height int64) (uint64, error)
	GetBalance(address common.Address, height int64) (*big.Int, error)
	GetCode(contract common.Address, height int64) (bytecode []byte, codeHash []byte)
	GetStorageAt(address common.Address, key string, height int64) []byte
	Call(tx *gethtypes.Transaction, from common.Address, height int64) (statusCode int, retData []byte)
	CallForSbch(tx *gethtypes.Transaction, sender common.Address, height int64) *CallDetail
	EstimateGas(tx *gethtypes.Transaction, from common.Address, height int64) (statusCode int, retData []byte, gas int64)
	QueryLogs(addresses []common.Address, topics [][]common.Hash, startHeight, endHeight uint32, filter motypes.FilterFunc) ([]motypes.Log, error)
	QueryTxBySrc(address common.Address, startHeight, endHeight, limit uint32) (tx []*motypes.Transaction, sigs [][65]byte, err error)
	QueryTxByDst(address common.Address, startHeight, endHeight, limit uint32) (tx []*motypes.Transaction, sigs [][65]byte, err error)
	QueryTxByAddr(address common.Address, startHeight, endHeight, limit uint32) (tx []*motypes.Transaction, sigs [][65]byte, err error)
	SbchQueryLogs(addr common.Address, topics []common.Hash, startHeight, endHeight, limit uint32) ([]motypes.Log, error)
	GetTxListByHeight(height uint32) (tx []*motypes.Transaction, sigs [][65]byte, err error)
	GetTxListByHeightWithRange(height uint32, start, end int) (tx []*motypes.Transaction, sigs [][65]byte, err error)
	GetFromAddressCount(addr common.Address) int64
	GetToAddressCount(addr common.Address) int64
	GetSep20ToAddressCount(contract common.Address, addr common.Address) int64
	GetSep20FromAddressCount(contract common.Address, addr common.Address) int64
	GetSeq(address common.Address) uint64
	ValidatorsInfo() app.ValidatorsInfo

	IsArchiveMode() bool
}
