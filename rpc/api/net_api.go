package api

import (
	"fmt"
)

var _ PublicNetAPI = (*netAPI)(nil)

type PublicNetAPI interface {
	Version() string
	Listening() bool
	PeerCount() int
}

type netAPI struct {
	networkID uint64
}

func newNetAPI(networkID uint64) PublicNetAPI {
	return netAPI{
		networkID: networkID,
	}
}

// https://eth.wiki/json-rpc/API#net_version
func (n netAPI) Version() string {
	return fmt.Sprintf("%d", n.networkID)
}

func (n netAPI) Listening() bool {
	panic("implement me")
}

func (n netAPI) PeerCount() int {
	panic("implement me")
}
