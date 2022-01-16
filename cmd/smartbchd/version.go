package main

import (
	"fmt"
	"github.com/tendermint/tendermint/p2p"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/smartbch/smartbch/app"
)

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version numbers",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(app.ClientID)
			fmt.Println("Version:", app.GitTag)
			if app.GitCommit != "" {
				fmt.Println("Git Commit:", app.GitCommit)
			}
			if app.GitDate != "" {
				fmt.Println("Git Commit Date:", app.GitDate)
			}
			fmt.Println("Architecture:", runtime.GOARCH)
			fmt.Println("Go Version:", runtime.Version())
			fmt.Println("Operating System:", runtime.GOOS)
			fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
			fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
			return nil
		},
	}

	return cmd
}


func NodeKeyCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node_key",
		Short: "Print node key",
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeCfg := ctx.Config.NodeConfig
			nodeKey, err := p2p.LoadOrGenNodeKey(nodeCfg.NodeKeyFile())
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", nodeKey.ID())
			return nil
		},
	}

	return cmd
}