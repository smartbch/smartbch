package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/smartbch/smartbch/internal/testutils"
)

func GenTestKeysCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-test-keys",
		Short: "generate private keys for test purpose",
		RunE: func(cmd *cobra.Command, args []string) error {
			n := viper.GetInt("number")
			for i := 0; i < n; i++ {
				key, _ := testutils.GenKeyAndAddr()
				fmt.Println(key)
			}

			return nil
		},
	}

	cmd.Flags().UintP("number", "n", 10, "how many test keys to generate")
	return cmd
}
