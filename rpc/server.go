package rpc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

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
	rpcHttpsAddr string // listen address of https rest-server
	wssAddr      string // listen address of https ws server
	corsDomain   string
	certFile     string
	keyFile      string
	httpAPIs     []string
	wsAPIs       []string
	serverConfig *tmrpcserver.Config

	logger  tmlog.Logger
	backend api.BackendService

	httpServer   *gethrpc.Server
	httpListener net.Listener
	wsServer     *gethrpc.Server
	wsListener   net.Listener

	httpsListener net.Listener
	wssListener   net.Listener

	unlockedKeys []string
}

func NewServer(rpcAddr, wsAddr, rpcAddrSecure, wsAddrSecure, corsDomain, certFile, keyFile string,
	serverCfg *tmrpcserver.Config, backend api.BackendService,
	logger tmlog.Logger, unlockedKeys []string,
	httpAPI string, wsAPI string) tmservice.Service {

	impl := &Server{
		rpcAddr:      rpcAddr,
		wsAddr:       wsAddr,
		corsDomain:   corsDomain,
		certFile:     certFile,
		keyFile:      keyFile,
		serverConfig: serverCfg,
		backend:      backend,
		logger:       logger,
		unlockedKeys: unlockedKeys,
		rpcHttpsAddr: rpcAddrSecure, //"tcp://:9545",
		wssAddr:      wsAddrSecure,  //"tcp://:9546",
		httpAPIs:     splitAndTrim(httpAPI),
		wsAPIs:       splitAndTrim(wsAPI),
	}
	return tmservice.NewBaseService(logger, "", impl)
}

func splitAndTrim(input string) (ret []string) {
	l := strings.Split(input, ",")
	for _, r := range l {
		if r = strings.TrimSpace(r); r != "" {
			ret = append(ret, r)
		}
	}
	return ret
}

func (server *Server) OnStart() error {
	apis := rpcapi.GetAPIs(server.backend, server.logger, server.unlockedKeys)
	if err := server.startHTTPAndHTTPS(apis); err != nil {
		return err
	}
	return server.startWSAndWSS(apis)
}

func (server *Server) startHTTPAndHTTPS(apis []gethrpc.API) (err error) {
	server.httpServer = gethrpc.NewServer()
	if err = registerApis(server.httpServer, server.httpAPIs, apis); err != nil {
		return err
	}

	allowedOrigins := strings.Split(server.corsDomain, ",")
	handler := newCorsHandler(server.httpServer, allowedOrigins)

	server.httpListener, err = tmrpcserver.Listen(
		server.rpcAddr, server.serverConfig)
	if err != nil {
		return err
	}
	go func() {
		err := tmrpcserver.Serve(server.httpListener, handler, server.logger,
			server.serverConfig)
		if err != nil {
			server.logger.Error(err.Error())
		}
	}()

	if server.rpcHttpsAddr != "off" {
		server.httpsListener, err = tmrpcserver.Listen(
			server.rpcHttpsAddr, server.serverConfig)
		if err != nil {
			return err
		}
		if server.certFile == "" {
			go func() {
				go func() {
					time.Sleep(3 * time.Second)
					server.backend.WaitRpcKeySet()
					server.StopHttpsListener()
				}()
				err := ServeTLSWithSelfSignedCertificate(server.httpsListener, handler,
					server.serverConfig, server.logger)
				if err != nil {
					server.logger.Error(err.Error())
				}
			}()
		} else {
			go func() {
				err := tmrpcserver.ServeTLS(server.httpsListener, handler,
					server.certFile, server.keyFile, server.logger,
					server.serverConfig)
				if err != nil {
					server.logger.Error(err.Error())
				}
			}()
		}
	}
	return nil
}

func ServeTLSWithSelfSignedCertificate(
	listener net.Listener,
	handler http.Handler,
	config *tmrpcserver.Config,
	logger tmlog.Logger,
) error {
	s := &http.Server{
		Handler:        tmrpcserver.RecoverAndLogHandler(handler, logger),
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		MaxHeaderBytes: config.MaxHeaderBytes,
		TLSConfig:      CreateCertificate("smartbch"),
	}
	err := s.ServeTLS(listener, "", "")

	logger.Error("RPC HTTPS server stopped", "err", err)
	return err
}

func CreateCertificate(serverName string) *tls.Config {
	template := &x509.Certificate{
		SerialNumber: &big.Int{},
		Subject:      pkix.Name{CommonName: serverName},
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		DNSNames:     []string{serverName},
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cert, _ := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	tlsCfg := tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{cert},
				PrivateKey:  priv,
			},
		},
	}
	return &tlsCfg
}

func (server *Server) startWSAndWSS(apis []gethrpc.API) (err error) {
	server.wsServer = gethrpc.NewServer()
	if err = registerApis(server.wsServer, server.wsAPIs, apis); err != nil {
		return err
	}

	allowedOrigins := strings.Split(server.corsDomain, ",")
	wsh := server.wsServer.WebsocketHandler(allowedOrigins)

	server.wsListener, err = tmrpcserver.Listen(
		server.wsAddr, server.serverConfig)
	if err != nil {
		return err
	}
	go func() {
		err := tmrpcserver.Serve(server.wsListener, wsh, server.logger,
			server.serverConfig)
		if err != nil {
			server.logger.Error(err.Error())
		}
	}()

	if server.wssAddr != "off" {
		server.wssListener, err = tmrpcserver.Listen(
			server.wssAddr, server.serverConfig)
		if err != nil {
			return err
		}
		go func() {
			err := tmrpcserver.ServeTLS(server.wssListener, wsh,
				server.certFile, server.keyFile, server.logger,
				server.serverConfig)
			if err != nil {
				server.logger.Error(err.Error())
			}
		}()
	}
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
	if server.httpsListener != nil {
		_ = server.httpsListener.Close()
	}
	if server.httpsListener != nil {
		_ = server.httpsListener.Close()
	}
}
func (server *Server) stopWS() {
	if server.wsServer != nil {
		server.wsServer.Stop()
	}
	if server.wsListener != nil {
		_ = server.httpListener.Close()
	}
	if server.wssListener != nil {
		_ = server.wssListener.Close()
	}
}

func (server *Server) StopHttpsListener() {
	if server.httpsListener != nil {
		_ = server.httpsListener.Close()
	}
}

func registerApis(rpcServer *gethrpc.Server, namespaces []string, apis []gethrpc.API) error {
	for _, _api := range apis {
		if exists(namespaces, _api.Namespace) {
			if err := rpcServer.RegisterName(_api.Namespace, _api.Service); err != nil {
				return err
			}
		}
	}
	return nil
}

func exists(set []string, find string) bool {
	for _, s := range set {
		if s == find {
			return true
		}
	}
	return false
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
