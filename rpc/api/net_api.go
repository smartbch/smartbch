package api

var _ PublicNetAPI = (*netAPI)(nil)

type PublicNetAPI interface {
	Version() string
	Listening() bool
	PeerCount() int
}

type netAPI struct {
}

// https://eth.wiki/json-rpc/API#net_version
func (n netAPI) Version() string {
	return "1" // TODO: 1 is Ethereum Mainnet
}

func (n netAPI) Listening() bool {
	panic("implement me")
}

func (n netAPI) PeerCount() int {
	panic("implement me")
}
