package api

import (
	"encoding/json"

	"github.com/smartbch/smartbch/api"
)

var _ TendermintAPI = (*tmAPI)(nil)

type TendermintAPI interface {
	NodeInfo() json.RawMessage
	ValidatorsInfo() json.RawMessage
}

type tmAPI struct {
	backend api.BackendService
}

func newTendermintAPI(backend api.BackendService) TendermintAPI {
	return &tmAPI{backend}
}

func (tm *tmAPI) NodeInfo() json.RawMessage {
	nodeInfo := tm.backend.NodeInfo()
	bytes, _ := json.Marshal(nodeInfo)
	return bytes
}

func (tm *tmAPI) ValidatorsInfo() json.RawMessage {
	info := tm.backend.ValidatorsInfo()
	bytes, _ := json.Marshal(info)
	return bytes
}
