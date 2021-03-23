package main

import (
	"fmt"

	"github.com/holiman/uint256"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	pvm "github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"

	moetypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/rpc"
)

const (
	flagRpcAddr = "http.addr"
	flagWsAddr  = "ws.addr"
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
	cmd.Flags().String(flagRpcAddr, "tcp://:8545", "HTTP-RPC server listening address")
	cmd.Flags().String(flagWsAddr, "tcp://:8546", "WS-RPC server listening address")
	return cmd
}

func startInProcess(ctx *Context, appCreator AppCreator) (*node.Node, error) {
	cfg := ctx.Config
	chainID, err := getChainID(ctx)
	if err != nil {
		return nil, err
	}
	_app := appCreator(ctx.Logger, chainID)
	appImpl := _app.(*app.App)

	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
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
	//todo: make sure this is the latest committed block
	latestBlock := tmNode.BlockStore().LoadBlock(tmNode.BlockStore().Height())
	fmt.Println("Load LatestBlock...")
	if latestBlock != nil {
		blk := moetypes.Block{}
		fmt.Println(latestBlock.String())
		copy(blk.Hash[:], latestBlock.Hash().Bytes())
		copy(blk.Miner[:], latestBlock.Header.ProposerAddress)
		blk.Number = latestBlock.Height
		blk.Timestamp = latestBlock.Time.Unix()
		appImpl.Init(&blk)
	} else {
		appImpl.Init(nil)
	}

	if err := tmNode.Start(); err != nil {
		return nil, err
	}

	rpcBackend := api.NewBackend(tmNode, appImpl)
	rpcAddr := viper.GetString(flagRpcAddr)
	wsAddr := viper.GetString(flagWsAddr)
	//rpcAddr = viper.GetString("")
	rpcServer := rpc.NewServer(rpcAddr, wsAddr,
		rpcBackend, ctx.Logger, appImpl.TestKeys())

	if err := rpcServer.Start(); err != nil {
		return nil, err
	}
	TrapSignal(func() {
		if tmNode.IsRunning() {
			_ = rpcServer.Stop()
			_ = tmNode.Stop()
			appImpl.Stop()
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
	return parseChainID(gDoc.ChainID)
}
