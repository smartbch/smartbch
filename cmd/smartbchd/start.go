package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/holiman/uint256"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	pvm "github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/rpc"
)

const (
	flagRpcAddr      = "http.addr"
	flagWsAddr       = "ws.addr"
	flagRetainBlocks = "retain"
	flagUnlock       = "unlock"
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
	cmd.PersistentFlags().String("log_level", ctx.Config.LogLevel, "Log level")
	cmd.Flags().Int64(flagRetainBlocks, -1, "Latest blocks this node retain, default retain all blocks")
	cmd.Flags().String(flagRpcAddr, "tcp://:8545", "HTTP-RPC server listening address")
	cmd.Flags().String(flagWsAddr, "tcp://:8546", "WS-RPC server listening address")
	cmd.Flags().String(flagUnlock, "", "Comma separated list of private keys to unlock (only for testing)")
	return cmd
}

func startInProcess(ctx *Context, appCreator AppCreator) (*node.Node, error) {
	paramConfig := param.DefaultConfig()
	cfg := ctx.Config
	cfg.SetRoot(viper.GetString(cli.HomeFlag))
	cfg.TxIndex.Indexer = "null"
	cfg.Mempool.Size = 20000
	cfg.Mempool.MaxTxsBytes = 4 * 1024*1024*1024
	paramConfig.NodeConfig = cfg
	paramConfig.AppDataPath = filepath.Join(cfg.RootDir, param.AppDataPath)
	paramConfig.ModbDataPath = filepath.Join(cfg.RootDir, param.ModbDataPath)
	paramConfig.RetainBlocks = viper.GetInt64(flagRetainBlocks)

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

	rpcBackend := api.NewBackend(tmNode, appImpl)
	rpcAddr := viper.GetString(flagRpcAddr)
	wsAddr := viper.GetString(flagWsAddr)
	unlockedKeys := viper.GetString(flagUnlock)
	//rpcAddr = viper.GetString("")
	certfileDir := filepath.Join(cfg.RootDir, "config/cert.pem")
	keyfileDir := filepath.Join(cfg.RootDir, "config/key.pem")
	rpcServer := rpc.NewServer(rpcAddr, wsAddr, rpcBackend, certfileDir, keyfileDir,
		ctx.Logger, strings.Split(unlockedKeys, ","))

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
