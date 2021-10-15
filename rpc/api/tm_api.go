package api

import (
	"encoding/json"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/api"
)

var _ TendermintAPI = (*tmAPI)(nil)

type TendermintAPI interface {
	NodeInfo() json.RawMessage
	ValidatorsInfo() json.RawMessage
}

type tmAPI struct {
	backend api.BackendService
	logger  log.Logger
}

func newTendermintAPI(backend api.BackendService, logger log.Logger) TendermintAPI {
	return &tmAPI{
		backend: backend,
		logger:  logger,
	}
}

func (tm *tmAPI) NodeInfo() json.RawMessage {
	tm.logger.Debug("tm_nodeInfo")
	nodeInfo := tm.backend.NodeInfo()
	bytes, _ := json.Marshal(nodeInfo)
	return bytes
}

func (tm *tmAPI) ValidatorsInfo() json.RawMessage {
	tm.logger.Debug("tm_validatorsInfo")
	info := tm.backend.ValidatorsInfo()
	bytes, _ := json.Marshal(info)
	return bytes
}
