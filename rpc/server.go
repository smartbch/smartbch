package rpc

import (
	"net"

	tmlog "github.com/tendermint/tendermint/libs/log"
	tmservice "github.com/tendermint/tendermint/libs/service"
	tmrpcserver "github.com/tendermint/tendermint/rpc/jsonrpc/server"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/moeing-chain/moeing-chain/api"
	rpcapi "github.com/moeing-chain/moeing-chain/rpc/api"
)

var _ tmservice.Service = (*Server)(nil)

// serve JSON-RPC over HTTP & WebSocket
type Server struct {
	tmservice.BaseService

	rpcAddr string // listen address of rest-server
	wsAddr  string // listen address of ws server
	logger  tmlog.Logger
	backend api.BackendService

	httpServer   *gethrpc.Server
	httpListener net.Listener
	wsServer     *gethrpc.Server
	wsListener   net.Listener

	testKeys []string
}

func NewServer(rpcAddr string, wsAddr string,
	backend api.BackendService,
	logger tmlog.Logger, testKeys []string) tmservice.Service {

	impl := &Server{
		rpcAddr:  rpcAddr,
		wsAddr:   wsAddr,
		backend:  backend,
		logger:   logger,
		testKeys: testKeys,
	}
	return tmservice.NewBaseService(logger, "", impl)
}

func (server *Server) OnStart() error {
	apis := rpcapi.GetAPIs(server.backend,
		server.logger, server.testKeys)
	if err := server.startHTTP(apis); err != nil {
		return err
	}
	return server.startWS(apis)
}

func (server *Server) startHTTP(apis []gethrpc.API) (err error) {
	server.httpServer = gethrpc.NewServer()
	if err = registerApis(server.httpServer, apis); err != nil {
		return err
	}

	server.httpListener, err = tmrpcserver.Listen(
		server.rpcAddr, tmrpcserver.DefaultConfig())
	if err != nil {
		return err
	}

	go tmrpcserver.Serve(server.httpListener, server.httpServer, server.logger,
		tmrpcserver.DefaultConfig()) // TODO: get config from config file
	return nil
}

func (server *Server) startWS(apis []gethrpc.API) (err error) {
	server.wsServer = gethrpc.NewServer()
	if err = registerApis(server.wsServer, apis); err != nil {
		return err
	}

	server.wsListener, err = tmrpcserver.Listen(
		server.wsAddr, tmrpcserver.DefaultConfig()) // TODO: get config from config file
	if err != nil {
		return err
	}

	allowedOrigins := []string{"*"} // TODO: get from cmd line options or config file
	wsh := server.wsServer.WebsocketHandler(allowedOrigins)

	go tmrpcserver.Serve(server.wsListener, wsh, server.logger,
		tmrpcserver.DefaultConfig()) // TODO: get config from config file
	return nil
}

func (server *Server) OnStop() {
	server.stopHTTP()
	server.stopWS()
}

func (server *Server) stopHTTP() {
	if server.httpServer != nil {
		server.httpServer.Stop()
	}
	if server.httpListener != nil {
		_ = server.httpListener.Close()
	}
}
func (server *Server) stopWS() {
	if server.wsServer != nil {
		server.httpServer.Stop()
	}
	if server.wsListener != nil {
		_ = server.httpListener.Close()
	}
}

func registerApis(rpcServer *gethrpc.Server, apis []gethrpc.API) error {
	for _, _api := range apis {
		if err := rpcServer.RegisterName(_api.Namespace, _api.Service); err != nil {
			return err
		}
	}
	return nil
}
