package main

import (
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/cli"
)

func InitCmd(ctx *Context, defaultNodeHome string) *cobra.Command { // nolint: golint
	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize follower files",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.Flags().String(cli.HomeFlag, defaultNodeHome, "follower's home directory")
	return cmd
}
