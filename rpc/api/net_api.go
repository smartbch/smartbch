package api

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"
)

var _ PublicNetAPI = (*netAPI)(nil)

type PublicNetAPI interface {
	Version() string
	Listening() bool
	PeerCount() int
}

type netAPI struct {
	networkID uint64
	logger    log.Logger
}

func newNetAPI(networkID uint64, logger log.Logger) PublicNetAPI {
	return netAPI{
		networkID: networkID,
		logger:    logger,
	}
}

// https://eth.wiki/json-rpc/API#net_version
func (n netAPI) Version() string {
	n.logger.Debug("net_version")
	return fmt.Sprintf("%d", n.networkID)
}

// https://eth.wiki/json-rpc/API#net_listening
func (n netAPI) Listening() bool {
	n.logger.Debug("net_listening")
	return true
}

// https://eth.wiki/json-rpc/API#net_peerCount
func (n netAPI) PeerCount() int {
	n.logger.Debug("net_peerCount")
	return 0
}
