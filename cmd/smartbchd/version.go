package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/param"
)

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version numbers",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(app.ClientID)
			fmt.Println("Version:", app.GitTag)
			fmt.Println("IsAmber:", param.IsAmber)
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
