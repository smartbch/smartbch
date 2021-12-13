package api

import (
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/tendermint/tendermint/libs/log"

	sbchapi "github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/rpc/api/filters"
)

const (
	namespaceEth    = "eth"
	namespaceNet    = "net"
	namespaceWeb3   = "web3"
	namespaceTxPool = "txpool"
	namespaceSBCH   = "sbch"
	namespaceTM     = "tm"
	namespaceDebug  = "debug"
	//namespaceEVM    = "evm"
	//namespacePersonal = "personal"

	apiVersion = "1.0"
)

// GetAPIs returns the list of all APIs from the Ethereum namespaces
func GetAPIs(backend sbchapi.BackendService,
	logger log.Logger, testKeys []string) []rpc.API {

	logger = logger.With("module", "json-rpc")
	_ethAPI := newEthAPI(backend, testKeys, logger)
	_netAPI := newNetAPI(backend.ChainId().Uint64(), logger)
	_filterAPI := filters.NewAPI(backend, logger)
	_web3API := newWeb3API(logger)
	_txPoolAPI := newTxPoolAPI(logger)
	_sbchAPI := newSbchAPI(backend, logger)
	_tmAPI := newTendermintAPI(backend, logger)
	_debugAPI := newDebugAPI()
	//_evmAPI := newEvmAPI(backend)

	return []rpc.API{
		{
			Namespace: namespaceEth,
			Version:   apiVersion,
			Service:   _ethAPI,
			Public:    true,
		},
		{
			Namespace: namespaceEth,
			Version:   apiVersion,
			Service:   _filterAPI,
			Public:    true,
		},
		{
			Namespace: namespaceWeb3,
			Version:   apiVersion,
			Service:   _web3API,
			Public:    true,
		},
		{
			Namespace: namespaceNet,
			Version:   apiVersion,
			Service:   _netAPI,
			Public:    true,
		},
		{
			Namespace: namespaceTxPool,
			Version:   apiVersion,
			Service:   _txPoolAPI,
			Public:    true,
		},
		{
			Namespace: namespaceSBCH,
			Version:   apiVersion,
			Service:   _sbchAPI,
			Public:    true,
		},
		{
			Namespace: namespaceTM,
			Version:   apiVersion,
			Service:   _tmAPI,
			Public:    true,
		},
		{
			Namespace: namespaceDebug,
			Version:   apiVersion,
			Service:   _debugAPI,
			Public:    true,
		},
	}
}
