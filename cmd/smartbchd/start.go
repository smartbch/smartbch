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
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	pvm "github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmrpcserver "github.com/tendermint/tendermint/rpc/jsonrpc/server"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/param"
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
	flagRetainBlocks         = "retain"
	flagUnlock               = "unlock"
	flagGenesisMainnetHeight = "mainnet-genesis-height"
	flagMainnetUrl           = "mainnet-url"
	flagMainnetRpcUser       = "mainnet-user"
	flagMainnetRpcPassword   = "mainnet-password"
	flagSmartBchUrl          = "smartbch-url"
	flagWatcherSpeedup       = "watcher-speedup"
	flagLogValidators        = "log-validators"
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
	cmd.PersistentFlags().String("log_level", ctx.Config.LogLevel, "Log level")
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
	cmd.Flags().Bool(flagLogValidators, false, "Log detailed validators info")

	return cmd
}

func startInProcess(ctx *Context, appCreator AppCreator) (*node.Node, error) {
	paramConfig := param.DefaultConfig()
	cfg := ctx.Config
	cfg.SetRoot(viper.GetString(cli.HomeFlag))
	cfg.TxIndex.Indexer = "null"
	cfg.Mempool.Size = 10000
	cfg.Mempool.MaxTxsBytes = 4 * 1024 * 1024 * 1024
	paramConfig.NodeConfig = cfg
	paramConfig.AppDataPath = filepath.Join(cfg.RootDir, param.AppDataPath)
	paramConfig.ModbDataPath = filepath.Join(cfg.RootDir, param.ModbDataPath)
	paramConfig.RetainBlocks = viper.GetInt64(flagRetainBlocks)
	paramConfig.MainnetRPCUrl = viper.GetString(flagMainnetUrl)
	paramConfig.MainnetRPCUserName = viper.GetString(flagMainnetRpcUser)
	paramConfig.MainnetRPCPassword = viper.GetString(flagMainnetRpcPassword)
	paramConfig.SmartBchRPCUrl = viper.GetString(flagSmartBchUrl)
	paramConfig.Speedup = viper.GetBool(flagWatcherSpeedup)
	paramConfig.LogValidatorsInfo = viper.GetBool(flagLogValidators)

	chainID, err := getChainID(ctx)
	if err != nil {
		return nil, err
	}
	_app := appCreator(ctx.Logger, chainID, paramConfig)
	appImpl := _app.(*app.App)

	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	fmt.Printf("This Node ID: %s\n", nodeKey.ID())
	if err != nil {
		return nil, err
	}
	tmNode, err := node.NewNode(
		cfg,
		pvm.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(_app),
		node.DefaultGenesisDocProviderFunc(cfg),
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		ctx.Logger.With("module", "node"),
	)
	if err != nil {
		return nil, err
	}
	fmt.Println("Load LatestBlock...")
	//todo: make sure this is the latest committed block
	//latestBlock := tmNode.BlockStore().LoadBlock(tmNode.BlockStore().Height())
	//if latestBlock != nil {
	//	blk := moetypes.Block{}
	//	fmt.Println(latestBlock.String())
	//	copy(blk.Hash[:], latestBlock.Hash().Bytes())
	//	copy(blk.Miner[:], latestBlock.Header.ProposerAddress)
	//	blk.Number = latestBlock.Height
	//	blk.Timestamp = latestBlock.Time.Unix()
	//	appImpl.Init(&blk)
	//} else {
	//	appImpl.Init(nil)
	//}

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
	ctx.Logger.Info("rpc server config: " + string(rpcServerCfgJSON))

	rpcBackend := api.NewBackend(tmNode, appImpl)
	rpcAddr := viper.GetString(flagRpcAddr)
	wsAddr := viper.GetString(flagWsAddr)
	corsDomain := viper.GetString(flagCorsDomain)
	unlockedKeys := viper.GetString(flagUnlock)
	certfileDir := filepath.Join(cfg.RootDir, "config/cert.pem")
	keyfileDir := filepath.Join(cfg.RootDir, "config/key.pem")
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
	gDoc, err := tmtypes.GenesisDocFromFile(ctx.Config.GenesisFile())
	if err != nil {
		return nil, err
	}

	chainID, err := parseChainID(gDoc.ChainID)
	if err != nil {
		return nil, err
	}

	return chainID, nil
}
