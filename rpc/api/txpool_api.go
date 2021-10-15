package api

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/tendermint/tendermint/libs/log"

	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
)

var _ PublicTxPoolAPI = (*txPoolAPI)(nil)

type PublicTxPoolAPI interface {
	Content() map[string]map[string]map[string]*rpctypes.Transaction
	Status() map[string]hexutil.Uint
	Inspect() map[string]map[string]map[string]string
}

type txPoolAPI struct {
	logger log.Logger
}

func newTxPoolAPI(logger log.Logger) PublicTxPoolAPI {
	return txPoolAPI{logger: logger}
}

func (api txPoolAPI) Content() map[string]map[string]map[string]*rpctypes.Transaction {
	api.logger.Debug("txpool_content")
	content := map[string]map[string]map[string]*rpctypes.Transaction{
		"pending": make(map[string]map[string]*rpctypes.Transaction),
		"queued":  make(map[string]map[string]*rpctypes.Transaction),
	}
	return content
}

func (api txPoolAPI) Status() map[string]hexutil.Uint {
	api.logger.Debug("txpool_status")
	pending, queue := 0, 0
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(pending),
		"queued":  hexutil.Uint(queue),
	}
}

func (api txPoolAPI) Inspect() map[string]map[string]map[string]string {
	api.logger.Debug("txpool_inspect")
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string),
		"queued":  make(map[string]map[string]string),
	}
	return content
}
