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

	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
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
	flagRpcAddr              = "http.addr"
	flagCorsDomain           = "http.corsdomain"
	flagWsAddr               = "ws.addr"
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
	defaultRpcCfg := tmrpcserver.DefaultConfig()
	cmd.PersistentFlags().String("log_level", ctx.Config.NodeConfig.LogLevel, "Log level")
	cmd.Flags().Int64(flagRetainBlocks, -1, "Latest blocks this node retain, default retain all blocks")
	cmd.Flags().Int64(flagGenesisMainnetHeight, 0, "genesis bch mainnet height for validator voting watched")
	cmd.Flags().String(flagRpcAddr, "tcp://:8545", "HTTP-RPC server listening address")
	cmd.Flags().String(flagWsAddr, "tcp://:8546", "WS-RPC server listening address")
	cmd.Flags().String(flagCorsDomain, "*", "Comma separated list of domains from which to accept cross origin requests (browser enforced)")
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

	tmNode, err := node.NewNode(
		nodeCfg,
		pvm.LoadOrGenFilePV(nodeCfg.PrivValidatorKeyFile(), nodeCfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(_app),
		node.DefaultGenesisDocProviderFunc(nodeCfg),
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(nodeCfg.Instrumentation),
		ctx.Logger.With("module", "node"),
	)
	if err != nil {
		return nil, err
	}

	if err := tmNode.Start(); err != nil {
		return nil, err
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
	rpcAddr := viper.GetString(flagRpcAddr)
	wsAddr := viper.GetString(flagWsAddr)
	corsDomain := viper.GetString(flagCorsDomain)
	unlockedKeys := viper.GetString(flagUnlock)
	certfileDir := filepath.Join(nodeCfg.RootDir, "nodeCfg/cert.pem")
	keyfileDir := filepath.Join(nodeCfg.RootDir, "nodeCfg/key.pem")
	rpcServer := rpc.NewServer(rpcAddr, wsAddr, corsDomain, certfileDir, keyfileDir,
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
