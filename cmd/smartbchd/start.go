package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/holiman/uint256"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	abci "github.com/tendermint/tendermint/abci/types"
	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	tmcfg "github.com/tendermint/tendermint/config"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	pvm "github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmrpcserver "github.com/tendermint/tendermint/rpc/jsonrpc/server"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/rpc"
)

const (
	flagRpcAddr              = "http.addr"       // old
	flagRpcAddrSecure        = "https.addr"      // old
	flagCorsDomain           = "http.corsdomain" // old
	flagWsAddr               = "ws.addr"         // old
	flagWsAddrSecure         = "wss.addr"        // old
	flagRpcAddrNew           = "rpc.http-addr"   // new
	flagRpcAddrSecureNew     = "rpc.https-addr"  // new
	flagRpcWsAddr            = "rpc.ws-addr"     // new
	flagRpcWsAddrSecure      = "rpc.wss-addr"    // new
	flagRpcCorsDomain        = "rpc.corsdomain"  // new
	flagMaxOpenConnections   = "rpc.max-open-connections"
	flagReadTimeout          = "rpc.read-timeout"
	flagWriteTimeout         = "rpc.write-timeout"
	flagMaxBodyBytes         = "rpc.max-body-bytes"
	flagMaxHeaderBytes       = "rpc.max-header-bytes"
	flagRetainBlocks         = "retain-blocks"
	flagUnlock               = "unlock"
	flagGenesisMainnetHeight = "mainnet-genesis-height"
	flagMainnetUrl           = "mainnet-rpc-url"
	flagMainnetRpcUser       = "mainnet-rpc-username"
	flagMainnetRpcPassword   = "mainnet-rpc-password"
	flagSmartBchUrl          = "smartbch-url"
	flagWatcherSpeedup       = "watcher-speedup"
	flagRpcOnly              = "rpc-only"
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
	cmd.PersistentFlags().String("log_level", ctx.Config.NodeConfig.LogLevel, "Log level")
	cmd.Flags().Int64(flagRetainBlocks, -1, "Latest blocks this node retain, default retain all blocks")
	cmd.Flags().Int64(flagGenesisMainnetHeight, 0, "genesis bch mainnet height for validator voting watched")
	cmd.Flags().String(flagRpcAddr, "", "HTTP-RPC server listening address")        // deprecated
	cmd.Flags().String(flagRpcAddrSecure, "", "HTTPS-RPC server listening address") // deprecated
	cmd.Flags().String(flagWsAddr, "", "WS-RPC server listening address")           // deprecated
	cmd.Flags().String(flagWsAddrSecure, "", "WSS-RPC server listening address")    // deprecated
	cmd.Flags().String(flagCorsDomain, "", "Comma separated list of domains ...")   // deprecated
	cmd.Flags().Uint(flagMaxOpenConnections, uint(defaultRpcCfg.MaxOpenConnections), "max open connections of RPC server")
	cmd.Flags().Uint(flagReadTimeout, 10, "read timeout (in seconds) of RPC server")
	cmd.Flags().Uint(flagWriteTimeout, 10, "write timeout (in seconds) of RPC server")
	cmd.Flags().Uint(flagMaxHeaderBytes, uint(defaultRpcCfg.MaxHeaderBytes), "max header bytes of RPC server")
	cmd.Flags().Uint(flagMaxBodyBytes, uint(defaultRpcCfg.MaxBodyBytes), "max body bytes of RPC server")
	cmd.Flags().String(flagUnlock, "", "Comma separated list of private keys to unlock (only for testing)")
	cmd.Flags().String(flagMainnetUrl, "tcp://:8432", "BCH Mainnet RPC URL")
	cmd.Flags().String(flagMainnetRpcUser, "user", "BCH Mainnet RPC user name")
	cmd.Flags().String(flagMainnetRpcPassword, "88888888", "BCH Mainnet RPC user password")
	cmd.Flags().String(flagSmartBchUrl, "tcp://:8545", "SmartBch RPC URL")
	cmd.Flags().Bool(flagWatcherSpeedup, false, "Watcher Speedup")
	cmd.Flags().Bool(flagRpcOnly, false, "Start RPC server even tmnode is not started correctly, only useful for debug purpose")
	cmd.Flags().String(flagRpcAddrNew, "tcp://:8545", "HTTP-RPC server listening address")
	cmd.Flags().String(flagRpcAddrSecureNew, "tcp://:9545", `HTTPS-RPC server listening address, use special value "off" to disable HTTPS`)
	cmd.Flags().String(flagRpcWsAddr, "tcp://:8546", "WS-RPC server listening address")
	cmd.Flags().String(flagRpcWsAddrSecure, "tcp://:9546", `WSS-RPC server listening address, use special value "off" to disable WSS`)
	cmd.Flags().String(flagRpcCorsDomain, "*", "Comma separated list of domains from which to accept cross origin requests (browser enforced)")

	_ = cmd.Flags().MarkDeprecated(flagRpcAddr, fmt.Sprintf("use --%s instead", flagRpcAddrNew))
	_ = cmd.Flags().MarkDeprecated(flagRpcAddrSecure, fmt.Sprintf("use --%s instead", flagRpcAddrSecureNew))
	_ = cmd.Flags().MarkDeprecated(flagWsAddr, fmt.Sprintf("use --%s instead", flagRpcWsAddr))
	_ = cmd.Flags().MarkDeprecated(flagWsAddrSecure, fmt.Sprintf("use --%s instead", flagRpcWsAddrSecure))
	_ = cmd.Flags().MarkDeprecated(flagCorsDomain, fmt.Sprintf("use --%s instead", flagRpcCorsDomain))

	return cmd
}

func startInProcess(ctx *Context, appCreator AppCreator) (*node.Node, error) {
	nodeCfg := ctx.Config.NodeConfig
	nodeCfg.TxIndex.Indexer = "null"
	nodeCfg.Mempool.Size = 10000
	nodeCfg.Mempool.MaxTxsBytes = 4 * 1024 * 1024 * 1024
	chainID, err := getChainID(ctx)
	if err != nil {
		return nil, err
	}
	_app := appCreator(ctx.Logger, chainID, ctx.Config)
	appImpl := _app.(*app.App)

	nodeKey, err := p2p.LoadOrGenNodeKey(nodeCfg.NodeKeyFile())
	if err != nil {
		return nil, err
	}
	fmt.Printf("This Node ID: %s\n", nodeKey.ID())

	rpcOnly := viper.GetBool(flagRpcOnly)
	tmNode, err := startTmNode(nodeCfg, nodeKey, _app,
		ctx.Logger.With("module", "node"))
	if err != nil {
		if !rpcOnly {
			return nil, err
		}
		ctx.Logger.Info("tmnode not started: " + err.Error())
	}

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

	rpcServerCfgJSON, _ := json.Marshal(serverCfg)
	ctx.Logger.Info("rpc server nodeCfg: " + string(rpcServerCfgJSON))

	rpcBackend := api.NewBackend(tmNode, appImpl)
	rpcAddr := getStringOption(flagRpcAddr, flagRpcAddrNew)
	wsAddr := getStringOption(flagWsAddr, flagRpcWsAddr)
	rpcAddrSecure := getStringOption(flagRpcAddrSecure, flagRpcAddrSecureNew)
	wsAddrSecure := getStringOption(flagWsAddrSecure, flagRpcWsAddrSecure)
	corsDomain := getStringOption(flagCorsDomain, flagRpcCorsDomain)
	unlockedKeys := viper.GetString(flagUnlock)
	certfileDir := filepath.Join(nodeCfg.RootDir, "nodeCfg/cert.pem")
	keyfileDir := filepath.Join(nodeCfg.RootDir, "nodeCfg/key.pem")
	rpcServer := rpc.NewServer(rpcAddr, wsAddr, rpcAddrSecure, wsAddrSecure, corsDomain, certfileDir, keyfileDir,
		serverCfg, rpcBackend, ctx.Logger, strings.Split(unlockedKeys, ","))

	if err := rpcServer.Start(); err != nil {
		return nil, err
	}
	TrapSignal(func() {
		if tmNode.IsRunning() {
			_ = rpcServer.Stop()
			_ = tmNode.Stop()
			//appImpl.Stop()
		}
		ctx.Logger.Info("exiting...")
	})

	// run forever (the node will not be returned)
	select {}
}

func getStringOption(oldFlag, newFlag string) string {
	val := viper.GetString(oldFlag)
	if val != "" {
		return val
	}
	return viper.GetString(newFlag)
}

func startTmNode(nodeCfg *tmcfg.Config,
	nodeKey *p2p.NodeKey,
	_app abci.Application,
	logger tmlog.Logger) (*node.Node, error) {

	tmNode, err := node.NewNode(
		nodeCfg,
		pvm.LoadOrGenFilePV(nodeCfg.PrivValidatorKeyFile(), nodeCfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(_app),
		node.DefaultGenesisDocProviderFunc(nodeCfg),
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(nodeCfg.Instrumentation),
		logger,
	)
	if err != nil {
		return nil, err
	}

	if err = tmNode.Start(); err != nil {
		return nil, err
	}

	return tmNode, nil
}

func getChainID(ctx *Context) (*uint256.Int, error) {
	gDoc, err := tmtypes.GenesisDocFromFile(ctx.Config.NodeConfig.GenesisFile())
	if err != nil {
		return nil, err
	}

	chainID, err := parseChainID(gDoc.ChainID)
	if err != nil {
		return nil, err
	}

	return chainID, nil
}
