package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/pelletier/go-toml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/libs/cli"
)

func ConfigCmd(defaultCLIHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <config type> <key> [value]",
		Short: "modify configuration file",
		RunE:  runConfigCmd,
		Args:  cobra.RangeArgs(1, 3),
	}

	cmd.Flags().String(cli.HomeFlag, defaultCLIHome,
		"set home directory for configuration")
	return cmd
}

func runConfigCmd(cmd *cobra.Command, args []string) error {
	cfgFile, err := ensureConfFile(viper.GetString(cli.HomeFlag), args[0])
	if err != nil {
		return err
	}

	// load configuration
	tree, err := loadConfigFile(cfgFile)
	if err != nil {
		return err
	}

	// print the config and exit
	if len(args) == 1 {
		s, err := tree.ToTomlString()
		if err != nil {
			return err
		}
		fmt.Print(s)
		return nil
	}

	if len(args) != 3 {
		return fmt.Errorf("wrong number of arguments")
	}
	key := args[1]
	value := args[2]

	// set config value for a given key
	if args[0] == "node" {
		switch key {
		case "p2p.seeds", "tx_index.indexer", "log_level":
			tree.Set(key, value)

		case "mempool.max_tx_bytes", "mempool.max_txs_bytes", "mempool.size":
			uintVal, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			tree.Set(key, uintVal)
		default:
			return errUnknownConfigKey(key)
		}
	} else /*update app.toml*/ {
		switch key {
		case "mainnet-rpc-url", "mainnet-rpc-username", "mainnet-rpc-password", "smartbch-rpc-url":
			tree.Set(key, value)

		case "watcher-speedup", "use_litedb", "log-validators":
			boolVal, err := strconv.ParseBool(value)
			if err != nil {
				return err
			}
			tree.Set(key, boolVal)
		case "retain-blocks", "retain_interval_blocks", "get_logs_max_results",
			"blocks_kept_ads", "blocks_kept_modb", "prune_every_n",
			"recheck_threshold", "sig_cache_size", "trunk_cache_size":
			uintVal, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			tree.Set(key, uintVal)
		default:
			return errUnknownConfigKey(key)
		}
	}

	// save configuration to disk
	if err := saveConfigFile(cfgFile, tree); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stderr, "configuration saved to %s\n", cfgFile)
	return nil
}

func ensureConfFile(rootDir, configType string) (string, error) {
	cfgPath := path.Join(rootDir, "config")
	if err := os.MkdirAll(cfgPath, os.ModePerm); err != nil {
		return "", err
	}
	if configType == "node" {
		return path.Join(cfgPath, "config.toml"), nil
	}
	return path.Join(cfgPath, "app.toml"), nil
}

func loadConfigFile(cfgFile string) (*toml.Tree, error) {
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		_, _ = fmt.Fprintf(os.Stderr, "%s does not exist\n", cfgFile)
		return toml.Load(``)
	}

	bz, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	tree, err := toml.LoadBytes(bz)
	if err != nil {
		return nil, err
	}

	return tree, nil
}

func saveConfigFile(cfgFile string, tree *toml.Tree) error {
	fp, err := os.OpenFile(cfgFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fp.Close()

	_, err = tree.WriteTo(fp)
	return err
}

func errUnknownConfigKey(key string) error {
	return fmt.Errorf("unknown configuration key: %q", key)
}
