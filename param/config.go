package param

import (
	"os"
	"path/filepath"
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
	RootPath        string `mapstructure:"root_path"`
	GenesisFilePath string `mapstructure:"genesis_file_path"`
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
	// Output level for logging
	LogLevel string `mapstructure:"log_level"`
}

type ChainConfig struct {
	*AppConfig `mapstructure:"app_config"`
}

var (
	defaultHome = os.ExpandEnv("$HOME/.follower")
)

func DefaultAppConfig() *AppConfig {
	return DefaultAppConfigWithHome(defaultHome)
}
func DefaultAppConfigWithHome(home string) *AppConfig {
	if home == "" {
		home = defaultHome
	}
	return &AppConfig{
		RootPath:                home,
		GenesisFilePath:         filepath.Join(home, "config", "genesis.json"),
		AppDataPath:             filepath.Join(home, "data", AppDataPath),
		ModbDataPath:            filepath.Join(home, "data", ModbDataPath),
		RpcEthGetLogsMaxResults: DefaultRpcEthGetLogsMaxResults,
		NumKeptBlocks:           DefaultNumKeptBlocks,
		NumKeptBlocksInMoDB:     DefaultNumKeptBlocksInMoDB,
		TrunkCacheSize:          DefaultTrunkCacheSize,
		PruneEveryN:             DefaultPruneEveryN,
		SmartBchRPCUrl:          "http://0.0.0.0:8545",
		LogLevel:                "debug",
	}
}

func DefaultConfig() *ChainConfig {
	c := &ChainConfig{
		AppConfig: DefaultAppConfig(),
	}
	return c
}
