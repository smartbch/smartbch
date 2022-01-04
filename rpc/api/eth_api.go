package api

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"sort"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/tendermint/tendermint/libs/log"
	tmrpc "github.com/tendermint/tendermint/rpc/core"

	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/moeingevm/types"
	sbchapi "github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/internal/ethutils"
	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
	"github.com/smartbch/smartbch/staking"
)

const (
	// DefaultGasPrice is default gas price for evm transactions
	DefaultGasPrice = 20000000000
	// DefaultRPCGasLimit is default gas limit for RPC call operations
	DefaultRPCGasLimit = 10000000
)

// smartBCH genesis height is 1, so we need this to make it compatible with Ethereum
var fakeBlock0 = &types.Block{
	Timestamp: 1627574400, // 2021-07-30
	ParentHash: [32]byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	},
}

var _ PublicEthAPI = (*ethAPI)(nil)

type PublicEthAPI interface {
	Accounts() ([]common.Address, error)
	BlockNumber() (hexutil.Uint64, error)
	Call(args rpctypes.CallArgs, blockNr gethrpc.BlockNumber) (hexutil.Bytes, error)
	ChainId() hexutil.Uint64
	Coinbase() (common.Address, error)
	EstimateGas(args rpctypes.CallArgs, blockNr *gethrpc.BlockNumber) (hexutil.Uint64, error)
	GasPrice() *hexutil.Big
	GetBalance(addr common.Address, blockNum gethrpc.BlockNumber) (*hexutil.Big, error)
	GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error)
	GetBlockByNumber(blockNum gethrpc.BlockNumber, fullTx bool) (map[string]interface{}, error)
	GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint
	GetBlockTransactionCountByNumber(blockNum gethrpc.BlockNumber) *hexutil.Uint
	GetCode(addr common.Address, blockNum gethrpc.BlockNumber) (hexutil.Bytes, error)
	GetStorageAt(addr common.Address, key string, blockNum gethrpc.BlockNumber) (hexutil.Bytes, error)
	GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.Transaction, error)
	GetTransactionByBlockNumberAndIndex(blockNum gethrpc.BlockNumber, idx hexutil.Uint) (*rpctypes.Transaction, error)
	GetTransactionByHash(hash common.Hash) (*rpctypes.Transaction, error)
	GetTransactionCount(addr common.Address, blockNum gethrpc.BlockNumber) (*hexutil.Uint64, error)
	GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error)
	GetUncleByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) map[string]interface{}
	GetUncleByBlockNumberAndIndex(number hexutil.Uint, idx hexutil.Uint) map[string]interface{}
	GetUncleCountByBlockHash(_ common.Hash) hexutil.Uint
	GetUncleCountByBlockNumber(_ gethrpc.BlockNumber) hexutil.Uint
	ProtocolVersion() hexutil.Uint
	SendRawTransaction(data hexutil.Bytes) (common.Hash, error) // ?
	SendTransaction(args rpctypes.SendTxArgs) (common.Hash, error)
	Syncing() (interface{}, error)
}

type ethAPI struct {
	backend  sbchapi.BackendService
	accounts map[common.Address]*ecdsa.PrivateKey // only for test
	logger   log.Logger
	numCall  uint64
}

func newEthAPI(backend sbchapi.BackendService, testKeys []string, logger log.Logger) *ethAPI {
	return &ethAPI{
		backend:  backend,
		accounts: loadTestAccounts(testKeys, logger),
		logger:   logger,
	}
}

func loadTestAccounts(testKeys []string, logger log.Logger) map[common.Address]*ecdsa.PrivateKey {
	accs := make(map[common.Address]*ecdsa.PrivateKey, len(testKeys))
	for _, testKey := range testKeys {
		if key, _, err := ethutils.HexToPrivKey(testKey); err == nil {
			addr := crypto.PubkeyToAddress(key.PublicKey)
			accs[addr] = key
		} else {
			logger.Error("failed to load private key:", testKey, err.Error())
		}
	}
	return accs
}

func (api *ethAPI) Accounts() ([]common.Address, error) {
	api.logger.Debug("eth_accounts")
	addrs := make([]common.Address, 0, len(api.accounts))
	for addr := range api.accounts {
		addrs = append(addrs, addr)
	}

	sort.Slice(addrs, func(i, j int) bool {
		for k := 0; k < common.AddressLength; k++ {
			if addrs[i][k] < addrs[j][k] {
				return true
			} else if addrs[i][k] > addrs[j][k] {
				return false
			}
		}
		return false
	})
	return addrs, nil
}

// https://eth.wiki/json-rpc/API#eth_blockNumber
func (api *ethAPI) BlockNumber() (hexutil.Uint64, error) {
	api.logger.Debug("eth_blockNumber")
	return hexutil.Uint64(api.backend.LatestHeight()), nil
}

// https://eips.ethereum.org/EIPS/eip-695
func (api *ethAPI) ChainId() hexutil.Uint64 {
	api.logger.Debug("eth_chainId")
	chainID := api.backend.ChainId()
	return hexutil.Uint64(chainID.Uint64())
}

// https://eth.wiki/json-rpc/API#eth_coinbase
func (api *ethAPI) Coinbase() (common.Address, error) {
	api.logger.Debug("eth_coinbase")
	// TODO: this is temporary implementation
	return common.Address{}, nil
}

// https://eth.wiki/json-rpc/API#eth_gasPrice
func (api *ethAPI) GasPrice() *hexutil.Big {
	api.logger.Debug("eth_gasPrice")
	val, err := api.GetStorageAt(staking.StakingContractAddress, staking.SlotMinGasPriceHex, -1)
	if err != nil {
		return (*hexutil.Big)(big.NewInt(0))
	}
	return (*hexutil.Big)(big.NewInt(0).SetBytes(val))
}

// https://eth.wiki/json-rpc/API#eth_getBalance
func (api *ethAPI) GetBalance(addr common.Address, blockNum gethrpc.BlockNumber) (*hexutil.Big, error) {
	api.logger.Debug("eth_getBalance")
	// ignore blockNumber temporary
	b, err := api.backend.GetBalance(addr)
	if err != nil {
		if err == types.ErrAccNotFound {
			return (*hexutil.Big)(big.NewInt(0)), nil
		}
		return nil, err
	}
	return (*hexutil.Big)(b), err
}

// https://eth.wiki/json-rpc/API#eth_getCode
func (api *ethAPI) GetCode(addr common.Address, blockNum gethrpc.BlockNumber) (hexutil.Bytes, error) {
	api.logger.Debug("eth_getCode")
	// ignore blockNumber temporary
	code, _ := api.backend.GetCode(addr)
	return code, nil
}

// https://eth.wiki/json-rpc/API#eth_getStorageAt
func (api *ethAPI) GetStorageAt(addr common.Address, key string, blockNum gethrpc.BlockNumber) (hexutil.Bytes, error) {
	api.logger.Debug("eth_getStorageAt")
	// ignore blockNumber temporary
	hash := common.HexToHash(key)
	key = string(hash[:])
	return api.backend.GetStorageAt(addr, key), nil
}

// https://eth.wiki/json-rpc/API#eth_getBlockByHash
func (api *ethAPI) GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	api.logger.Debug("eth_getBlockByHash")
	var zeroHash common.Hash
	var txs []*types.Transaction
	var sigs [][65]byte
	if hash == zeroHash {
		return blockToRpcResp(fakeBlock0, txs, sigs), nil
	}
	block, err := api.backend.BlockByHash(hash)
	if err != nil {
		if err == types.ErrBlockNotFound {
			return nil, nil
		}
		return nil, err
	}

	if fullTx {
		txs, err = api.backend.GetTxListByHeight(uint32(block.Number))
		if err != nil {
			return nil, err
		}

		sigs = api.backend.GetSigs(txs)
	}

	return blockToRpcResp(block, txs, sigs), nil
}

// https://eth.wiki/json-rpc/API#eth_getBlockByNumber
func (api *ethAPI) GetBlockByNumber(blockNum gethrpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	api.logger.Debug("eth_getBlockByNumber")
	block, err := api.getBlockByNum(blockNum)
	if err != nil {
		if err == types.ErrBlockNotFound {
			return nil, nil
		}
		return nil, err
	}

	var txs []*types.Transaction
	var sigs [][65]byte
	if fullTx {
		txs, err = api.backend.GetTxListByHeight(uint32(block.Number))
		if err != nil {
			return nil, err
		}

		sigs = api.backend.GetSigs(txs)
	}
	return blockToRpcResp(block, txs, sigs), nil
}

// https://eth.wiki/json-rpc/API#eth_getBlockTransactionCountByHash
func (api *ethAPI) GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint {
	api.logger.Debug("eth_getBlockTransactionCountByHash")
	block, err := api.backend.BlockByHash(hash)
	if err != nil {
		return nil
	}
	n := hexutil.Uint(len(block.Transactions))
	return &n
}

// https://eth.wiki/json-rpc/API#eth_getBlockTransactionCountByNumber
func (api *ethAPI) GetBlockTransactionCountByNumber(blockNum gethrpc.BlockNumber) *hexutil.Uint {
	api.logger.Debug("eth_getBlockTransactionCountByNumber")
	block, err := api.getBlockByNum(blockNum)
	if err != nil {
		return nil
	}
	n := hexutil.Uint(len(block.Transactions))
	return &n
}

// https://eth.wiki/json-rpc/API#eth_getTransactionByBlockHashAndIndex
func (api *ethAPI) GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.Transaction, error) {
	api.logger.Debug("eth_getTransactionByBlockHashAndIndex")
	block, err := api.backend.BlockByHash(hash)
	if err != nil {
		if err == types.ErrBlockNotFound {
			return nil, nil
		}
		return nil, err
	}
	return api.getTxByIdx(block, idx)
}

// https://eth.wiki/json-rpc/API#eth_getTransactionByBlockNumberAndIndex
func (api *ethAPI) GetTransactionByBlockNumberAndIndex(blockNum gethrpc.BlockNumber, idx hexutil.Uint) (*rpctypes.Transaction, error) {
	api.logger.Debug("eth_getTransactionByBlockNumberAndIndex")
	block, err := api.getBlockByNum(blockNum)
	if err != nil {
		if err == types.ErrBlockNotFound {
			return nil, nil
		}
		return nil, err
	}
	return api.getTxByIdx(block, idx)
}

// https://eth.wiki/json-rpc/API#eth_getTransactionByHash
func (api *ethAPI) GetTransactionByHash(hash common.Hash) (*rpctypes.Transaction, error) {
	api.logger.Debug("eth_getTransactionByHash")
	tx, sig, err := api.backend.GetTransaction(hash)
	if err != nil {
		return nil, nil
	}
	return txToRpcResp(tx, sig), nil
}

// https://eth.wiki/json-rpc/API#eth_getTransactionCount
func (api *ethAPI) GetTransactionCount(addr common.Address, blockNum gethrpc.BlockNumber) (*hexutil.Uint64, error) {
	api.logger.Debug("eth_getTransactionCount")
	// ignore blockNumber temporary
	nonce, _ := api.backend.GetNonce(addr)
	nonceU64 := hexutil.Uint64(nonce)
	return &nonceU64, nil
}

func (api *ethAPI) getBlockByNum(blockNum gethrpc.BlockNumber) (*types.Block, error) {
	height := blockNum.Int64()
	if height < 0 {
		// get latest block height
		return api.backend.CurrentBlock()
	}
	if height == 0 {
		return fakeBlock0, nil
	}
	return api.backend.BlockByNumber(height)
}

func (api *ethAPI) getTxByIdx(block *types.Block, idx hexutil.Uint) (*rpctypes.Transaction, error) {
	if uint64(idx) >= uint64(len(block.Transactions)) {
		// return if index out of bounds
		return nil, nil
	}

	txHash := block.Transactions[idx]
	tx, sig, err := api.backend.GetTransaction(txHash)
	if err != nil {
		return nil, err
	}

	return txToRpcResp(tx, sig), nil
}

// https://eth.wiki/json-rpc/API#eth_getTransactionReceipt
func (api *ethAPI) GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error) {
	api.logger.Debug("eth_getTransactionReceipt")
	tx, _, err := api.backend.GetTransaction(hash)
	if err != nil {
		// the transaction is not yet available
		return nil, nil
	}
	return txToReceiptRpcResp(tx), nil
}

// https://eth.wiki/json-rpc/API#eth_getUncleByBlockHashAndIndex
func (api *ethAPI) GetUncleByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) map[string]interface{} {
	api.logger.Debug("eth_getUncleByBlockHashAndIndex")
	// not supported
	return nil
}

// https://eth.wiki/json-rpc/API#eth_getUncleByBlockNumberAndIndex
func (api *ethAPI) GetUncleByBlockNumberAndIndex(number hexutil.Uint, idx hexutil.Uint) map[string]interface{} {
	api.logger.Debug("eth_getUncleByBlockNumberAndIndex")
	// not supported
	return nil
}

// https://eth.wiki/json-rpc/API#eth_getUncleCountByBlockHash
func (api *ethAPI) GetUncleCountByBlockHash(_ common.Hash) hexutil.Uint {
	api.logger.Debug("eth_getUncleCountByBlockHash")
	// not supported
	return 0
}

// https://eth.wiki/json-rpc/API#eth_getUncleCountByBlockNumber
func (api *ethAPI) GetUncleCountByBlockNumber(_ gethrpc.BlockNumber) hexutil.Uint {
	api.logger.Debug("eth_getUncleCountByBlockNumber")
	// not supported
	return 0
}

// https://eth.wiki/json-rpc/API#eth_protocolVersion
func (api *ethAPI) ProtocolVersion() hexutil.Uint {
	api.logger.Debug("eth_protocolVersion")
	return hexutil.Uint(api.backend.ProtocolVersion())
}

// https://eth.wiki/json-rpc/API#eth_sendRawTransaction
func (api *ethAPI) SendRawTransaction(data hexutil.Bytes) (common.Hash, error) {
	api.logger.Debug("eth_sendRawTransaction")
	tx, err := ethutils.DecodeTx(data)
	if err != nil {
		return common.Hash{}, err
	}

	tmTxHash, err := api.backend.SendRawTx(data)
	if err != nil {
		return tmTxHash, err
	}

	return tx.Hash(), nil
}

// https://eth.wiki/json-rpc/API#eth_sendTransaction
func (api *ethAPI) SendTransaction(args rpctypes.SendTxArgs) (common.Hash, error) {
	api.logger.Debug("eth_sendTransaction")
	privKey, found := api.accounts[args.From]
	if !found {
		return common.Hash{}, errors.New("unknown account: " + args.From.Hex())
	}

	if args.Nonce == nil {
		if nonce, err := api.backend.GetNonce(args.From); err == nil {
			args.Nonce = (*hexutil.Uint64)(&nonce)
		}
	}

	tx, err := createGethTxFromSendTxArgs(args)
	if err != nil {
		return common.Hash{}, err
	}

	chainID := api.backend.ChainId()
	tx, err = ethutils.SignTx(tx, chainID, privKey)
	if err != nil {
		return common.Hash{}, err
	}

	txBytes, err := ethutils.EncodeTx(tx)
	if err != nil {
		return common.Hash{}, err
	}

	tmTxHash, err := api.backend.SendRawTx(txBytes)
	if err != nil {
		return tmTxHash, err
	}

	txHash := tx.Hash()
	return txHash, err
}

// https://eth.wiki/json-rpc/API#eth_syncing
func (api *ethAPI) Syncing() (interface{}, error) {
	api.logger.Debug("eth_syncing")
	status, err := tmrpc.Status(nil)
	if err != nil {
		return false, err
	}
	if !status.SyncInfo.CatchingUp {
		return false, nil
	}

	return map[string]interface{}{
		// "startingBlock": nil, // NA
		"currentBlock": hexutil.Uint64(status.SyncInfo.LatestBlockHeight),
		// "highestBlock":  nil, // NA
		// "pulledStates":  nil, // NA
		// "knownStates":   nil, // NA
	}, nil
}

// https://eth.wiki/json-rpc/API#eth_call
func (api *ethAPI) Call(args rpctypes.CallArgs, blockNr gethrpc.BlockNumber) (hexutil.Bytes, error) {
	atomic.AddUint64(&api.numCall, 1)
	api.logger.Debug("eth_call", "from", addrToStr(args.From), "to", addrToStr(args.To))
	// ignore blockNumber temporary
	tx, from, err := api.createGethTxFromCallArgs(args)
	if err != nil {
		return hexutil.Bytes{}, err
	}

	statusCode, retData := api.backend.Call(tx, from)
	if !ebp.StatusIsFailure(statusCode) {
		return retData, nil
	}

	return nil, toCallErr(statusCode, retData)
}

func addrToStr(addr *common.Address) string {
	if addr != nil {
		return addr.Hex()
	}
	return "0x"
}

// https://eth.wiki/json-rpc/API#eth_estimateGas
func (api *ethAPI) EstimateGas(args rpctypes.CallArgs, blockNr *gethrpc.BlockNumber) (hexutil.Uint64, error) {
	api.logger.Debug("eth_estimateGas")
	tx, from, err := api.createGethTxFromCallArgs(args)
	if err != nil {
		return 0, err
	}

	statusCode, retData, gas := api.backend.EstimateGas(tx, from)
	if !ebp.StatusIsFailure(statusCode) {
		return hexutil.Uint64(gas), nil
	}

	return 0, toCallErr(statusCode, retData)
}

func (api *ethAPI) createGethTxFromCallArgs(args rpctypes.CallArgs,
) (*gethtypes.Transaction, common.Address, error) {

	var from, to common.Address
	if args.From != nil {
		from = *args.From
	}
	if args.To != nil {
		to = *args.To
	}

	var val *big.Int
	if args.Value != nil {
		val = args.Value.ToInt()
	} else {
		val = big.NewInt(0)
	}

	var gasLimit uint64 = DefaultRPCGasLimit
	if args.Gas != nil {
		gasLimit = uint64(*args.Gas)
	}

	var gasPrice *big.Int
	if args.GasPrice != nil {
		gasPrice = args.GasPrice.ToInt()
	} else {
		gasPrice = big.NewInt(DefaultGasPrice)
	}

	var data []byte
	if args.Data != nil {
		data = *args.Data
	}

	tx := ethutils.NewTx(0, &to, val, gasLimit, gasPrice, data)
	return tx, from, nil
}
