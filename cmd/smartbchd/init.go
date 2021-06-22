package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
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

const (
	flagChainID      = "chain-id"
	flagOverwrite    = "overwrite"
	flagTestKeys     = "test-keys"
	flagTestKeysFile = "test-keys-file"
	flagInitBal      = "init-balance"
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
			chainID := viper.GetString(flagChainID)
			if chainID == "" {
				chainID = "test-chain"
			}
			nodeID, _, err := InitializeNodeValidatorFiles(config)
			if err != nil {
				return err
			}
			config.Moniker = args[0]
			genFile := config.GenesisFile()
			if !viper.GetBool(flagOverwrite) && FileExists(genFile) {
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

			fmt.Println("saving genesis file ...")
			if err := ExportGenesisFile(genDoc, genFile); err != nil {
				return err
			}
			toPrint := newPrintInfo(config.Moniker, chainID, nodeID)
			cfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)
			return displayInfo(toPrint)
		},
	}
	cmd.Flags().String(cli.HomeFlag, defaultNodeHome, "node's home directory")
	cmd.Flags().BoolP(flagOverwrite, "o", false, "overwrite the genesis.json file")
	cmd.Flags().String(flagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	cmd.Flags().String(flagTestKeys, "", "comma separated list of hex private keys used for test")
	cmd.Flags().String(flagTestKeysFile, "", "file contains hex private keys, one key per line")
	cmd.Flags().String(flagInitBal, "1000000000000000000", "initial balance for test accounts")
	return cmd
}

func getAppState() ([]byte, error) {
	initBalance := viper.GetString(flagInitBal)
	initBal, ok := bigutils.ParseU256(initBalance)
	if !ok {
		return nil, errors.New("invalid init balance")
	}

	testKeys := getTestKeys()

	fmt.Println("preparing genesis file ...")
	alloc := testutils.KeysToGenesisAlloc(initBal, testKeys)
	genData := app.GenesisData{Alloc: alloc}
	appState, err := json.Marshal(genData)
	if err != nil {
		return nil, err
	}
	return appState, nil
}

func getTestKeys() []string {
	testKeysCSV := viper.GetString(flagTestKeys)
	if testKeysCSV != "" {
		return strings.Split(testKeysCSV, ",")
	}

	testKeyFiles := viper.GetString(flagTestKeysFile)
	if testKeyFiles != "" {
		var allKeys []string
		for _, testKeyFile := range strings.Split(testKeyFiles, ",") {
			count := math.MaxInt32
			if idx := strings.Index(testKeyFile, ":"); idx > 0 {
				n, err := strconv.ParseInt(testKeyFile[idx+1:], 10, 32)
				if err != nil {
					panic(err)
				}
				count = int(n)
				testKeyFile = testKeyFile[:idx]
			}

			keys := testutils.ReadKeysFromFile(testKeyFile, count)
			allKeys = append(allKeys, keys...)
		}
		return allKeys
	}

	return nil
}
