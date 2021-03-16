package api

import (
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/tendermint/tendermint/libs/log"

	moeingapi "github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/rpc/api/filters"
)

const (
	namespaceEth      = "eth"
	namespaceNet      = "net"
	namespaceWeb3     = "web3"
	namespacePersonal = "personal"
	namespaceMoe      = "moe"

	apiVersion = "1.0"
)

// GetAPIs returns the list of all APIs from the Ethereum namespaces
func GetAPIs(backend moeingapi.BackendService,
	logger log.Logger, testKeys []string) []rpc.API {

	_ethAPI := newEthAPI(backend, testKeys, logger)
	filterAPI := filters.NewAPI(backend)

	_moeAPI := newMoeAPI(backend)

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
			Service:   filterAPI,
			Public:    true,
		},
		{
			Namespace: namespaceWeb3,
			Version:   apiVersion,
			Service:   web3API{},
			Public:    true,
		},
		{
			Namespace: namespaceNet,
			Version:   apiVersion,
			Service:   netAPI{},
			Public:    true,
		},
		{
			Namespace: namespaceMoe,
			Version:   apiVersion,
			Service:   _moeAPI,
			Public:    true,
		},
		// TODO: more apis
	}
}
