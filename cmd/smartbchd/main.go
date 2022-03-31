package main

import (
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/param"
)

type AppCreator func(logger log.Logger, config *param.ChainConfig) *app.App

func main() {
	rootCmd := createSmartbchdCmd()
	executor := cli.PrepareBaseCmd(rootCmd, "GA", DefaultNodeHome)
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
	rootCmd.AddCommand(VersionCmd())
	return rootCmd
}

func addInitCommands(ctx *Context, rootCmd *cobra.Command) {
	initCmd := InitCmd(ctx, DefaultNodeHome)
	rootCmd.AddCommand(initCmd)
}

func newApp(logger log.Logger, config *param.ChainConfig) *app.App {
	return app.NewApp(config, logger)
}
