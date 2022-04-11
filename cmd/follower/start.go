package main

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	tmservice "github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/node"
	tmrpcserver "github.com/tendermint/tendermint/rpc/jsonrpc/server"

	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/rpc"
)

const (
	flagRpcAddr            = "http.addr"
	flagRpcAddrSecure      = "https.addr"
	flagRpcAPI             = "http.api"
	flagCorsDomain         = "http.corsdomain"
	flagWsAddr             = "ws.addr"
	flagWsAddrSecure       = "wss.addr"
	flagWsAPI              = "ws.api"
	flagMaxOpenConnections = "rpc.max-open-connections"
	flagReadTimeout        = "rpc.read-timeout"
	flagWriteTimeout       = "rpc.write-timeout"
	flagMaxBodyBytes       = "rpc.max-body-bytes"
	flagMaxHeaderBytes     = "rpc.max-header-bytes"
	flagUnlock             = "unlock"

	flagSmartBchUrl = "smartbch-url"
	flagArchiveMode = "archive-mode"
)

func StartCmd(ctx *Context, appCreator AppCreator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the full node",

		RunE: func(cmd *cobra.Command, args []string) error {
			ctx.Logger.Info("starting SmartBCH Chain with Tendermint")
			_, err := startInProcess(ctx, appCreator)
			return err
		},
	}
	tcmd.AddNodeFlags(cmd)
	_ = cmd.Flags().MarkHidden("rpc.laddr")
	_ = cmd.Flags().MarkHidden("rpc.grpc_laddr")
	_ = cmd.Flags().MarkHidden("rpc.unsafe")
	_ = cmd.Flags().MarkHidden("rpc.pprof_laddr")
	_ = cmd.Flags().MarkHidden("proxy_app")

	defaultRpcCfg := tmrpcserver.DefaultConfig()
	cmd.PersistentFlags().String("log_level", ctx.Config.LogLevel, "Log level")
	cmd.Flags().String(flagRpcAddr, "tcp://:8545", "HTTP-RPC server listening address")
	cmd.Flags().String(flagRpcAddrSecure, "tcp://:9545", "HTTPS-RPC server listening address, use special value \"off\" to disable HTTPS")
	cmd.Flags().String(flagWsAddr, "tcp://:8546", "WS-RPC server listening address")
	cmd.Flags().String(flagWsAddrSecure, "tcp://:9546", "WSS-RPC server listening address, use special value \"off\" to disable WSS")
	cmd.Flags().String(flagCorsDomain, "*", "Comma separated list of domains from which to accept cross origin requests (browser enforced)")
	cmd.Flags().Uint(flagMaxOpenConnections, uint(defaultRpcCfg.MaxOpenConnections), "max open connections of RPC server")
	cmd.Flags().Uint(flagReadTimeout, 10, "read timeout (in seconds) of RPC server")
	cmd.Flags().Uint(flagWriteTimeout, 10, "write timeout (in seconds) of RPC server")
	cmd.Flags().Uint(flagMaxHeaderBytes, uint(defaultRpcCfg.MaxHeaderBytes), "max header bytes of RPC server")
	cmd.Flags().Uint(flagMaxBodyBytes, uint(defaultRpcCfg.MaxBodyBytes), "max body bytes of RPC server")
	cmd.Flags().String(flagUnlock, "", "Comma separated list of private keys to unlock (only for testing)")
	cmd.Flags().String(flagSmartBchUrl, "tcp://:8545", "SmartBch RPC URL")
	cmd.Flags().String(flagRpcAPI, "eth,web3,net,txpool,sbch,tm", "API's offered over the HTTP-RPC interface")
	cmd.Flags().String(flagWsAPI, "eth,web3,net,txpool,sbch,tm", "API's offered over the WS-RPC interface")
	cmd.Flags().Bool(flagArchiveMode, false, "enable archive-mode")

	return cmd
}

func startInProcess(ctx *Context, appCreator AppCreator) (*node.Node, error) {
	_app := appCreator(ctx.Logger, ctx.Config)
	rpcServer, err := startRPCServer(_app, ctx)
	if err != nil {
		ctx.Logger.Info("start rpc server failed", "error", err)
		return nil, err
	}
	ctx.Logger.Info("start rpc server successful")
	TrapSignal(func() {
		_ = rpcServer.Stop()
		ctx.Logger.Info("exiting...")
	})
	select {}
}

func startRPCServer(app *app.App, ctx *Context) (tmservice.Service, error) {
	serverCfg := tmrpcserver.DefaultConfig()
	if n := viper.GetUint(flagMaxOpenConnections); n > 0 {
		serverCfg.MaxOpenConnections = int(n)
	}
	if n := viper.GetUint(flagReadTimeout); n > 0 {
		serverCfg.ReadTimeout = time.Duration(n) * time.Second
	}
	if n := viper.GetUint(flagWriteTimeout); n > 0 {
		serverCfg.WriteTimeout = time.Duration(n) * time.Second
	}
	if n := viper.GetUint(flagMaxHeaderBytes); n > 0 {
		serverCfg.MaxHeaderBytes = int(n)
	}
	if n := viper.GetUint(flagMaxBodyBytes); n > 0 {
		serverCfg.MaxBodyBytes = int64(n)
	}
	rpcBackend := api.NewBackend(app)
	certFileDir := filepath.Join(ctx.Config.RootPath, "nodeCfg/cert.pem")
	keyFileDir := filepath.Join(ctx.Config.RootPath, "nodeCfg/key.pem")
	rpcServer := rpc.NewServer(viper.GetString(flagRpcAddr), viper.GetString(flagWsAddr),
		viper.GetString(flagRpcAddrSecure), viper.GetString(flagWsAddrSecure), viper.GetString(flagCorsDomain), certFileDir, keyFileDir,
		serverCfg, rpcBackend, ctx.Logger, strings.Split(viper.GetString(flagUnlock), ","),
		viper.GetString(flagRpcAPI), viper.GetString(flagWsAPI))
	if err := rpcServer.Start(); err != nil {
		return nil, err
	}
	return rpcServer, nil
}
