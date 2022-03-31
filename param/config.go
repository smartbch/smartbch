package param

import (
	"os"
	"path/filepath"

	"github.com/tendermint/tendermint/config"
)

const (
	DefaultRpcEthGetLogsMaxResults = 10000
	DefaultNumKeptBlocks           = 10000
	DefaultNumKeptBlocksInMoDB     = -1
	DefaultTrunkCacheSize          = 200
	DefaultPruneEveryN             = 10

	AppDataPath  = "app"
	ModbDataPath = "modb"
)

type AppConfig struct {
	//app config:
	AppDataPath  string `mapstructure:"app_data_path"`
	ModbDataPath string `mapstructure:"modb_data_path"`
	// rpc config
	RpcEthGetLogsMaxResults int `mapstructure:"get_logs_max_results"`
	// Use LiteDB instead of MoDB
	UseLiteDB bool `mapstructure:"use_litedb"`
	// the number of kept recent blocks for moeingads
	NumKeptBlocks int64 `mapstructure:"blocks_kept_ads"`
	// the number of kept recent blocks for moeingdb
	NumKeptBlocksInMoDB int64 `mapstructure:"blocks_kept_modb"`
	// the entry count of the signature cache
	TrunkCacheSize int    `mapstructure:"trunk_cache_size"`
	PruneEveryN    int64  `mapstructure:"prune_every_n"`
	SmartBchRPCUrl string `mapstructure:"smartbch-rpc-url"`

	ArchiveMode bool `mapstructure:"archive-mode"`
}

type ChainConfig struct {
	NodeConfig *config.Config `mapstructure:"node_config"`
	AppConfig  *AppConfig     `mapstructure:"app_config"`
}

var (
	defaultHome = os.ExpandEnv("$HOME/.smartbchd")
)

func DefaultAppConfig() *AppConfig {
	return DefaultAppConfigWithHome(defaultHome)
}
func DefaultAppConfigWithHome(home string) *AppConfig {
	if home == "" {
		home = defaultHome
	}
	return &AppConfig{
		AppDataPath:             filepath.Join(home, "data", AppDataPath),
		ModbDataPath:            filepath.Join(home, "data", ModbDataPath),
		RpcEthGetLogsMaxResults: DefaultRpcEthGetLogsMaxResults,
		NumKeptBlocks:           DefaultNumKeptBlocks,
		NumKeptBlocksInMoDB:     DefaultNumKeptBlocksInMoDB,
		TrunkCacheSize:          DefaultTrunkCacheSize,
		PruneEveryN:             DefaultPruneEveryN,
	}
}

func DefaultConfig() *ChainConfig {
	c := &ChainConfig{
		NodeConfig: config.DefaultConfig(),
		AppConfig:  DefaultAppConfig(),
	}
	c.NodeConfig.TxIndex.Indexer = "null"
	return c
}
