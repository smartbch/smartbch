package rpc

import (
	"net"
	"net/http"

	tmlog "github.com/tendermint/tendermint/libs/log"
	tmservice "github.com/tendermint/tendermint/libs/service"
	tmrpcserver "github.com/tendermint/tendermint/rpc/jsonrpc/server"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/cors"

	"github.com/smartbch/smartbch/api"
	rpcapi "github.com/smartbch/smartbch/rpc/api"
)

var _ tmservice.Service = (*Server)(nil)

// serve JSON-RPC over HTTP & WebSocket
type Server struct {
	tmservice.BaseService

	rpcAddr      string // listen address of rest-server
	wsAddr       string // listen address of ws server
	rpcHttpsAddr string //listen address of https rest-server
	wssAddr      string //listen address of https ws server

	logger  tmlog.Logger
	backend api.BackendService

	httpServer   *gethrpc.Server
	httpListener net.Listener
	wsServer     *gethrpc.Server
	wsListener   net.Listener

	httpsListener     net.Listener
	wssListener       net.Listener
	certFile, keyFile string

	testKeys []string
}

func NewServer(rpcAddr string, wsAddr string,
	backend api.BackendService, certFile, keyFile string,
	logger tmlog.Logger, testKeys []string) tmservice.Service {

	impl := &Server{
		rpcAddr:      rpcAddr,
		wsAddr:       wsAddr,
		backend:      backend,
		logger:       logger,
		testKeys:     testKeys,
		certFile:     certFile,
		keyFile:      keyFile,
		rpcHttpsAddr: "tcp://:9545",
		wssAddr:      "tcp://:9546",
	}
	return tmservice.NewBaseService(logger, "", impl)
}

func (server *Server) OnStart() error {
	apis := rpcapi.GetAPIs(server.backend,
		server.logger, server.testKeys)
	if err := server.startHTTPAndHTTPS(apis); err != nil {
		return err
	}
	return server.startWSAndWSS(apis)
}

func (server *Server) startHTTPAndHTTPS(apis []gethrpc.API) (err error) {
	server.httpServer = gethrpc.NewServer()
	if err = registerApis(server.httpServer, apis); err != nil {
		return err
	}

	server.httpListener, err = tmrpcserver.Listen(
		server.rpcAddr, tmrpcserver.DefaultConfig())
	if err != nil {
		return err
	}

	server.httpsListener, err = tmrpcserver.Listen(
		server.rpcHttpsAddr, tmrpcserver.DefaultConfig())
	if err != nil {
		return err
	}

	allowedOrigins := []string{"*"} // TODO: get from cmd line options or config file
	handler := newCorsHandler(server.httpServer, allowedOrigins)
	go tmrpcserver.Serve(server.httpListener, handler, server.logger,
		tmrpcserver.DefaultConfig()) // TODO: get config from config file
	go tmrpcserver.ServeTLS(server.httpsListener, handler,
		server.certFile, server.keyFile, server.logger, tmrpcserver.DefaultConfig())
	return nil
}

func (server *Server) startWSAndWSS(apis []gethrpc.API) (err error) {
	server.wsServer = gethrpc.NewServer()
	if err = registerApis(server.wsServer, apis); err != nil {
		return err
	}

	server.wsListener, err = tmrpcserver.Listen(
		server.wsAddr, tmrpcserver.DefaultConfig()) // TODO: get config from config file
	if err != nil {
		return err
	}

	server.wssListener, err = tmrpcserver.Listen(
		server.wssAddr, tmrpcserver.DefaultConfig())
	if err != nil {
		return err
	}
	allowedOrigins := []string{"*"} // TODO: get from cmd line options or config file
	wsh := server.wsServer.WebsocketHandler(allowedOrigins)

	go tmrpcserver.Serve(server.wsListener, wsh, server.logger,
		tmrpcserver.DefaultConfig()) // TODO: get config from config file

	go tmrpcserver.ServeTLS(server.wssListener, wsh,
		server.certFile, server.keyFile, server.logger, tmrpcserver.DefaultConfig()) // TODO: get config from config file
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

func newCorsHandler(srv http.Handler, allowedOrigins []string) http.Handler {
	// disable CORS support if user has not specified a custom CORS configuration
	if len(allowedOrigins) == 0 {
		return srv
	}
	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodPost, http.MethodGet},
		AllowedHeaders: []string{"*"},
		MaxAge:         600,
	})
	return c.Handler(srv)
}
