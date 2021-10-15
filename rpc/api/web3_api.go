package api

import (
	"fmt"
	"runtime"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/app"
)

var _ PublicWeb3API = (*web3API)(nil)

type PublicWeb3API interface {
	ClientVersion() string
	Sha3(input hexutil.Bytes) hexutil.Bytes
}

type web3API struct {
	logger log.Logger
}

func newWeb3API(logger log.Logger) PublicWeb3API {
	return web3API{logger: logger}
}

// https://eth.wiki/json-rpc/API#web3_clientversion
func (w web3API) ClientVersion() string {
	w.logger.Debug("web3_clientVersion")
	// like Geth/v1.10.2-unstable/darwin-amd64/go1.16.3
	return fmt.Sprintf("%s/%s/%s/%s",
		app.ClientID, app.GitTag, runtime.GOOS, runtime.Version())
}

// https://eth.wiki/json-rpc/API#web3_sha3
func (w web3API) Sha3(input hexutil.Bytes) hexutil.Bytes {
	w.logger.Debug("web3_sha3")
	return crypto.Keccak256(input)
}
