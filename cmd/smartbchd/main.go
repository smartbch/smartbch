package main

import (
	"github.com/holiman/uint256"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/param"
)

type AppCreator func(logger log.Logger, chainId *uint256.Int) abci.Application

func main() {
	rootCmd := createSmartbchdCmd()
	executor := cli.PrepareBaseCmd(rootCmd, "GA", app.DefaultNodeHome)
	err := executor.Execute()
	if err != nil {
		// handle with #870
		panic(err)
	}
}

func createSmartbchdCmd() *cobra.Command {
	cobra.EnableCommandSorting = false
	ctx := NewDefaultContext()
	rootCmd := &cobra.Command{
		Use:               "smartbchd",
		Short:             "SmartBCH Chain Daemon (server)",
		PersistentPreRunE: PersistentPreRunEFn(ctx),
	}
	addInitCommands(ctx, rootCmd)
	rootCmd.AddCommand(StartCmd(ctx, newApp))
	rootCmd.AddCommand(GenerateConsensusKeyInfoCmd(ctx))
	rootCmd.AddCommand(GenerateGenesisValidatorCmd(ctx))
	rootCmd.AddCommand(AddGenesisValidatorCmd(ctx))
	return rootCmd
}

func addInitCommands(ctx *Context, rootCmd *cobra.Command) {
	initCmd := InitCmd(ctx, app.DefaultNodeHome)
	genTestKeysCmd := GenTestKeysCmd(ctx)
	rootCmd.AddCommand(initCmd, genTestKeysCmd)
}

func newApp(logger log.Logger, chainId *uint256.Int) abci.Application {
	c := cfg.DefaultConfig()
	c.SetRoot(app.DefaultNodeHome)
	conf := param.DefaultConfig()
	conf.RetainBlocks = viper.GetInt64(flagRetainBlocks)
	cetChainApp := app.NewApp(conf, chainId, logger)
	return cetChainApp
}
