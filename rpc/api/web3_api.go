package api

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

var _ PublicWeb3API = (*web3API)(nil)

type PublicWeb3API interface {
	ClientVersion() string
	Sha3(input hexutil.Bytes) hexutil.Bytes
}

type web3API struct {
}

func (w web3API) ClientVersion() string {
	// TODO: this is temporary implementation
	return "smartBCH v0.2.0"
}

func (w web3API) Sha3(input hexutil.Bytes) hexutil.Bytes {
	return crypto.Keccak256(input)
}
