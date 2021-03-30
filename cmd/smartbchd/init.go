package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/types"

	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/internal/testutils"
)

type printInfo struct {
	Moniker    string          `json:"moniker" yaml:"moniker"`
	ChainID    string          `json:"chain_id" yaml:"chain_id"`
	NodeID     string          `json:"node_id" yaml:"node_id"`
	GenTxsDir  string          `json:"gentxs_dir" yaml:"gentxs_dir"`
	AppMessage json.RawMessage `json:"app_message" yaml:"app_message"`
}

func newPrintInfo(moniker, chainID, nodeID string) printInfo {
	return printInfo{
		Moniker: moniker,
		ChainID: chainID,
		NodeID:  nodeID,
	}
}

func displayInfo(info printInfo) error {
	out, _ := json.Marshal(info)
	_, err := fmt.Fprintf(os.Stderr, "%s\n", string(out))
	return err
}

func InitCmd(ctx *Context, defaultNodeHome string) *cobra.Command { // nolint: golint
	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize private validator files",
		Long:  `Initialize validator's and node's configuration files.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			config := ctx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))
			chainID := viper.GetString(FlagChainID)
			if chainID == "" {
				chainID = "test-chain"
			}
			nodeID, _, err := InitializeNodeValidatorFiles(config)
			if err != nil {
				return err
			}
			config.Moniker = args[0]
			genFile := config.GenesisFile()
			if !viper.GetBool(FlagOverwrite) && FileExists(genFile) {
				return fmt.Errorf("genesis.json file already exists: %v", genFile)
			}
			genDoc := &types.GenesisDoc{}
			if _, err := os.Stat(genFile); err != nil {
				if !os.IsNotExist(err) {
					return err
				}
			} else {
				genDoc, err = types.GenesisDocFromFile(genFile)
				if err != nil {
					return err
				}
			}
			genDoc.ChainID = chainID
			genDoc.AppState, err = getAppState()
			if err != nil {
				return err
			}

			if err := ExportGenesisFile(genDoc, genFile); err != nil {
				return err
			}
			toPrint := newPrintInfo(config.Moniker, chainID, nodeID)
			cfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)
			return displayInfo(toPrint)
		},
	}
	cmd.Flags().String(cli.HomeFlag, defaultNodeHome, "node's home directory")
	cmd.Flags().BoolP(FlagOverwrite, "o", false, "overwrite the genesis.json file")
	cmd.Flags().String(FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	cmd.Flags().String(FlagTestKeys, "", "comma separated list of hex private keys used for test")
	cmd.Flags().String(FlagInitBal, "1000000000000000000", "initial balance for test accounts")
	return cmd
}

func getAppState() ([]byte, error) {
	testKeysCSV := viper.GetString(FlagTestKeys)
	initBalance := viper.GetString(FlagInitBal)

	initBal, ok := bigutils.ParseU256(initBalance)
	if !ok {
		return nil, errors.New("invalid init balance")
	}

	var testKeys []string
	if testKeysCSV != "" {
		testKeys = strings.Split(testKeysCSV, ",")
	}

	alloc := testutils.KeysToGenesisAlloc(initBal, testKeys)
	genData := app.GenesisData{Alloc: alloc}
	appState, err := json.Marshal(genData)
	if err != nil {
		return nil, err
	}
	return appState, nil
}
