package api

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"

	motypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/app"
	cctypes "github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/staking/types"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
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

	// General Ethereum API
	//Downloader() *downloader.Downloader
	ProtocolVersion() int
	//SuggestPrice(ctx context.Context) (*big.Int, error)
	//ChainDb() Database
	//AccountManager() *accounts.Manager
	//ExtRPCEnabled() bool
	//RPCGasCap() uint64    // global gas cap for eth_call over rpc: DoS protection
	//RPCTxFeeCap() float64 // global tx fee cap for all transaction related APIs

	// Blockchain API
	ChainId() *big.Int
	//SetHead(number uint64)
	//HeaderByNumber(ctx context.Context, number int64) (*types.Header, error)
	//HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	//HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error)
	//CurrentHeader() *types.Header
	LatestHeight() int64
	CurrentBlock() (*motypes.Block, error)
	BlockByNumber(number int64) (*motypes.Block, error)
	BlockByHash(hash common.Hash) (*motypes.Block, error)
	//BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error)
	//StateAndHeaderByNumber(ctx context.Context, number int64) (*state.StateDB, error)
	//StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
	//GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) /*All receipt fields is in types.Transaction, use getTransaction() instead*/
	//GetTd(ctx context.Context, hash common.Hash) *big.Int
	//GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header) (*vm.EVM, func() error, error)

	// Transaction pool API
	SendRawTx(signedTx []byte) (common.Hash, error)
	GetTransaction(txHash common.Hash) (tx *motypes.Transaction, sig [65]byte, err error)
	//GetPoolTransactions() (types.Transactions, error)
	//GetPoolTransaction(txHash common.Hash) *types.Transaction
	//GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error)
	//Stats() (pending int, queued int)
	//TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions)

	// Filter API
	//BloomStatus() (uint64, uint64)
	//GetLogs(blockHash common.Hash) ([][]types.Log, error)
	//ServiceFilter(ctx context.Context, session *bloombits.MatcherSession)
	//SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription
	//SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription
	//SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription
	//SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription
	//SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription
	//SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
	//SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription

	//Engine() consensus.Engine

	//Below is added in moeing chain only
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
	GetEpochs(start, end uint64) ([]*types.Epoch, error)
	GetEpochList(from string) ([]*types.Epoch, error)
	GetCurrEpoch() *types.Epoch
	GetCCEpochs(start, end uint64) ([]*cctypes.CCEpoch, error)
	GetSeq(address common.Address) uint64
	GetPosVotes() map[[32]byte]*big.Int
	GetSyncBlock(height int64) (blk []byte, err error)
	GetRpcMaxLogResults() int

	//tendermint info
	NodeInfo() Info
	ValidatorsInfo(height int64) app.ValidatorsInfo
	ValidatorOnlineInfos() (int64, stakingtypes.ValidatorOnlineInfos)
	ValidatorWatchInfos() stakingtypes.ValidatorWatchInfos

	IsArchiveMode() bool

	GetRpcPrivateKey() *ecdsa.PrivateKey
	SetRpcPrivateKey(key *ecdsa.PrivateKey) bool
	WaitRpcKeySet()
}
