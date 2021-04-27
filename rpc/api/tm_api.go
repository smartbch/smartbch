package api

import (
	"encoding/json"

	"github.com/smartbch/smartbch/api"
)

var _ TendermintAPI = (*tmAPI)(nil)

type TendermintAPI interface {
	GetNodeInfo() json.RawMessage // TODO: remove this method
	NodeInfo() json.RawMessage
}

type tmAPI struct {
	backend api.BackendService
}

func newTendermintAPI(backend api.BackendService) TendermintAPI {
	return &tmAPI{backend}
}

func (tm *tmAPI) NodeInfo() json.RawMessage {
	return tm.GetNodeInfo()
}

func (tm *tmAPI) GetNodeInfo() json.RawMessage {
	nodeInfo := tm.backend.NodeInfo()
	bytes, _ := json.Marshal(nodeInfo)
	return bytes
}
