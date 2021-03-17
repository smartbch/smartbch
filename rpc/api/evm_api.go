package api

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	sbchapi "github.com/smartbch/smartbch/api"
)

// https://github.com/trufflesuite/ganache-cli#custom-methods
type EvmAPI interface {
	Mine(ts hexutil.Uint64) error
}

func newEvmAPI(backend sbchapi.BackendService) EvmAPI {
	return evmAPI{backend}
}

type evmAPI struct {
	backend sbchapi.BackendService
}

func (api evmAPI) Mine(ts hexutil.Uint64) error {
	for i := 0; i < 100; i++ {
		h := api.backend.LatestHeight()
		b, err := api.backend.BlockByNumber(h)
		if err != nil {
			return err
		}
		if b.Timestamp >= int64(ts) {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return errors.New("wait too long")
}
