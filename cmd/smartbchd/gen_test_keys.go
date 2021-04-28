package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/smartbch/smartbch/internal/testutils"
)

const (
	flagNumber   = "number"
	flagShowAddr = "show-address"
)

func GenTestKeysCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-test-keys",
		Short: "generate private keys for test purpose",
		RunE: func(cmd *cobra.Command, args []string) error {
			n := viper.GetInt(flagNumber)
			showAddr := viper.GetBool(flagShowAddr)
			for i := 0; i < n; i++ {
				key, addr := testutils.GenKeyAndAddr()
				if !showAddr {
					fmt.Println(key)
				} else {
					fmt.Println(key, addr.Hex())
				}
			}

			return nil
		},
	}

	cmd.Flags().UintP(flagNumber, "n", 10, "how many test keys to generate")
	cmd.Flags().Bool(flagShowAddr, false, "show address")
	return cmd
}
