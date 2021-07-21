package api

import (
	"github.com/ethereum/go-ethereum/common/hexutil"

	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
)

var _ PublicTxPoolAPI = (*txPoolAPI)(nil)

type PublicTxPoolAPI interface {
	Content() map[string]map[string]map[string]*rpctypes.Transaction
	Status() map[string]hexutil.Uint
	Inspect() map[string]map[string]map[string]string
}

type txPoolAPI struct {
}

func (txPoolAPI) Content() map[string]map[string]map[string]*rpctypes.Transaction {
	content := map[string]map[string]map[string]*rpctypes.Transaction{
		"pending": make(map[string]map[string]*rpctypes.Transaction),
		"queued":  make(map[string]map[string]*rpctypes.Transaction),
	}
	return content
}

func (txPoolAPI) Status() map[string]hexutil.Uint {
	pending, queue := 0, 0
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(pending),
		"queued":  hexutil.Uint(queue),
	}
}

func (txPoolAPI) Inspect() map[string]map[string]map[string]string {
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string),
		"queued":  make(map[string]map[string]string),
	}
	return content
}
