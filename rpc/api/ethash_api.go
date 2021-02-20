package api

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
)

type EthashAPI interface {
	GetWork() ([4]string, error)
	Hashrate() hexutil.Uint64
	Mining() bool
	SubmitWork(nonce gethtypes.BlockNonce, hash, digest common.Hash) bool
}
