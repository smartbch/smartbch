package api

import (
	"fmt"
	"runtime"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartbch/smartbch/app"
)

var _ PublicWeb3API = (*web3API)(nil)

type PublicWeb3API interface {
	ClientVersion() string
	Sha3(input hexutil.Bytes) hexutil.Bytes
}

type web3API struct {
}

func (w web3API) ClientVersion() string {
	// like Geth/v1.10.2-unstable/darwin-amd64/go1.16.3
	return fmt.Sprintf("%s/%s/%s/%s",
		app.ClientID, app.GitTag, runtime.GOOS, runtime.Version())
}

func (w web3API) Sha3(input hexutil.Bytes) hexutil.Bytes {
	return crypto.Keccak256(input)
}
