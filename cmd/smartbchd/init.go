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
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/types"

	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/param"
)

const (
	flagChainID      = "chain-id"
	flagOverwrite    = "overwrite"
	flagTestKeys     = "test-keys"
	flagTestKeysFile = "test-keys-file"
	flagInitBal      = "init-balance"
	flagMainnet      = "mainnet"
)

const (
	mainnetChainId      = "0x2710"
	mainnetSbchRPCUrl   = "https://rpc.smartbch.org"
	mainnetDefaultSeeds = "d96aafcbdc92dcb295ee28b050b47104a1749e23@13.229.211.167:26656"
	mainnetGenesisJSON  = `{
  "genesis_time": "2021-07-30T04:28:16.955082878Z",
  "chain_id": "0x2710",
  "initial_height": "1",
  "consensus_params": {
    "block": {
      "max_bytes": "22020096",
      "max_gas": "-1",
      "time_iota_ms": "1000"
    },
    "evidence": {
      "max_age_num_blocks": "100000",
      "max_age_duration": "172800000000000",
      "max_bytes": "1048576"
    },
    "validator": {
      "pub_key_types": [
        "ed25519"
      ]
    },
    "version": {}
  },
  "app_hash": "",
  "app_state": {
    "validators": [
      {
        "address": "0x9a6dd2f7ceb71788de691844d16b6b6852f07aa3",
        "pubkey": "0xfbdc5c690ab36319d6a68ed50407a61d95d0ec6a6e9225a0c40d17bd8358010e",
        "reward_to": "0x9a6dd2f7ceb71788de691844d16b6b6852f07aa3",
        "voting_power": 10,
        "introduction": "matrixport",
        "staked_coins": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "is_retiring": false
      },
      {
        "address": "0x7dd41d92235cbbe0d2fe4ebd548cdd29f9befe5e",
        "pubkey": "0x45caa8b683a1838f6cf8c3de60ef826ceaac27351843bc9f8c84cedb7da9a8a0",
        "reward_to": "0x7dd41d92235cbbe0d2fe4ebd548cdd29f9befe5e",
        "voting_power": 1,
        "introduction": "btccom",
        "staked_coins": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "is_retiring": false
      },
      {
        "address": "0xde5ddf2a1101d9501aa3db39750acb1764aa5c5b",
        "pubkey": "0xfc609736388585e77dc106885dd401b1dab7be87e61a3597239db9d0483e9a46",
        "reward_to": "0xde5ddf2a1101d9501aa3db39750acb1764aa5c5b",
        "voting_power": 1,
        "introduction": "viabtc",
        "staked_coins": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "is_retiring": false
      }
    ],
    "alloc": {
      "0x9a6dd2f7ceb71788de691844d16b6b6852f07aa3": {
        "balance": "0x115eec47f6cf7e35000000"
      }
    }
  }
}
`
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
			config := ctx.Config.NodeConfig
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

			if viper.GetBool(flagMainnet) {
				// load mainnet genesis file
				chainID = mainnetChainId
				err = tmjson.Unmarshal([]byte(mainnetGenesisJSON), genDoc)
				if err != nil {
					return fmt.Errorf("failed to load mainnet genesis JSON file: %w", err)
				}

				// update config/config.toml
				config.P2P.Seeds = mainnetDefaultSeeds

				// update config/app.toml
				ctx.Config.AppConfig.SmartBchRPCUrl = mainnetSbchRPCUrl
				appCfgPath := filepath.Join(config.RootDir, "config", "app.toml")
				param.WriteConfigFile(appCfgPath, ctx.Config.AppConfig)

				fmt.Println("home:", config.RootDir)
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
	cmd.Flags().Bool(flagMainnet, false, "init for mainent node")
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
