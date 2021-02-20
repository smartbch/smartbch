package main

import (
	"fmt"
	"github.com/spf13/cobra"

	tmcli "github.com/tendermint/tendermint/libs/cli"

	"github.com/moeing-chain/moeing-chain/app"
)

func main() {
	cobra.EnableCommandSorting = false

	rootCmd := &cobra.Command{
		Use:   "moeingcli",
		Short: "Command line interface for interacting with moeingd",
	}

	//rootCmd.PersistentFlags().String(sdkflags.FlagChainID, "", "Chain ID of moeing node")

	rootCmd.AddCommand(
	//sdkrpc.StatusCommand(),
	//sdkcli.ConfigCmd(app.DefaultCLIHome),
	//version.Cmd,
	)

	executor := tmcli.PrepareMainCmd(rootCmd, "MOE", app.DefaultCLIHome)

	err := executor.Execute()
	if err != nil {
		panic(fmt.Errorf("failed executing CLI command: %w", err))
	}
}
