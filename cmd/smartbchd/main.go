package main

import (
	"encoding/json"
	"fmt"
	"github.com/holiman/uint256"
	"github.com/spf13/cobra"

	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/privval"

	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/param"
)

type AppCreator func(logger log.Logger, chainId *uint256.Int) abci.Application

func main() {
	rootCmd := createMoeingdCmd()
	executor := cli.PrepareBaseCmd(rootCmd, "GA", app.DefaultNodeHome)
	err := executor.Execute()
	if err != nil {
		// handle with #870
		panic(err)
	}
}

func createMoeingdCmd() *cobra.Command {
	cobra.EnableCommandSorting = false
	ctx := NewDefaultContext()
	rootCmd := &cobra.Command{
		Use:               "smartbchd",
		Short:             "SmartBCH Chain Daemon (server)",
		PersistentPreRunE: PersistentPreRunEFn(ctx),
	}
	addInitCommands(ctx, rootCmd)
	rootCmd.AddCommand(StartCmd(ctx, newApp))
	return rootCmd
}

func addInitCommands(ctx *Context, rootCmd *cobra.Command) {
	initCmd := InitCmd(ctx, app.DefaultNodeHome)
	rootCmd.AddCommand(initCmd)
}

func newApp(logger log.Logger, chainId *uint256.Int) abci.Application {
	c := cfg.DefaultConfig()
	c.SetRoot(app.DefaultNodeHome)
	privValKeyFile := c.PrivValidatorKeyFile()
	privValStateFile := c.PrivValidatorStateFile()
	pv := privval.LoadFilePV(privValKeyFile, privValStateFile)
	var testValidators [10]crypto.PubKey
	for i := 0; i < 10; i++ {
		testValidators[i] = ed25519.GenPrivKey().PubKey()
	}
	bz, _ := json.Marshal(testValidators)
	fmt.Printf("testValidator:%s\n", bz)
	//initAmt.Mul(initAmt, bigutils.NewU256(500))
	cetChainApp := app.NewApp(param.DefaultConfig(), chainId, logger,
		pv.Key.PubKey)
	return cetChainApp
}
