package api

import (
	"encoding/json"

	"github.com/smartbch/smartbch/api"
)

var _ TendermintAPI = (*tmAPI)(nil)

type TendermintAPI interface {
	NodeInfo() json.RawMessage
}

type tmAPI struct {
	backend api.BackendService
}

func newTendermintAPI(backend api.BackendService) TendermintAPI {
	return &tmAPI{backend}
}

func (tm *tmAPI) NodeInfo() json.RawMessage {
	nodeInfo := tm.backend.NodeInfo()
	return marshalNodeInfo(nodeInfo)
}
