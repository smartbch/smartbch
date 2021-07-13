package param

import (
	"os"
	"path/filepath"

	"github.com/tendermint/tendermint/config"
)

const (
	DefaultRpcEthGetLogsMaxResults = 10000
	DefaultRetainBlocks            = -1
	DefaultNumKeptBlocks           = 10000
	DefaultNumKeptBlocksInMoDB     = -1
	DefaultSignatureCache          = 20000
	DefaultRecheckThreshold        = 1000
	DefaultTrunkCacheSize          = 200
	DefaultChangeRetainEveryN      = 100
	DefaultPruneEveryN             = 10
)

type ChainConfig struct {
	NodeConfig *config.Config
	//app config:
	AppDataPath  string `json:"app_data_path,omitempty"`
	ModbDataPath string `json:"modb_data_path,omitempty"`
	// rpc config
	RpcEthGetLogsMaxResults int
	// tm db config
	RetainBlocks       int64
	ChangeRetainEveryN int64
	// Use LiteDB instead of MoDB
	UseLiteDB bool
	// the number of kept recent blocks for moeingads
	NumKeptBlocks int64
	// the number of kept recent blocks for moeingdb
	NumKeptBlocksInMoDB int64
	// the entry count of the signature cache
	SigCacheSize   int
	TrunkCacheSize int
	PruneEveryN    int64
	// How many transactions are allowed to left in the mempool
	// If more than this threshold, no further transactions can go in mempool
	RecheckThreshold int
	//watcher config
	MainnetRPCUrl      string
	MainnetRPCUserName string
	MainnetRPCPassword string
	SmartBchRPCUrl     string
	Speedup            bool
	LogValidatorsInfo  bool
}

var (
	AppDataPath  = "app"
	ModbDataPath = "modb"

	home                = os.ExpandEnv("$HOME/.smartbchd")
	defaultAppDataPath  = filepath.Join(home, "data", AppDataPath)
	defaultModbDataPath = filepath.Join(home, "data", ModbDataPath)
)

func DefaultConfig() *ChainConfig {
	os.LookupEnv("HOME")
	c := &ChainConfig{
		NodeConfig:              config.DefaultConfig(),
		AppDataPath:             defaultAppDataPath,
		ModbDataPath:            defaultModbDataPath,
		RpcEthGetLogsMaxResults: DefaultRpcEthGetLogsMaxResults,
		RetainBlocks:            DefaultRetainBlocks,
		NumKeptBlocks:           DefaultNumKeptBlocks,
		NumKeptBlocksInMoDB:     DefaultNumKeptBlocksInMoDB,
		SigCacheSize:            DefaultSignatureCache,
		RecheckThreshold:        DefaultRecheckThreshold,
		TrunkCacheSize:          DefaultTrunkCacheSize,
		ChangeRetainEveryN:      DefaultChangeRetainEveryN,
		PruneEveryN:             DefaultPruneEveryN,
	}
	c.NodeConfig.TxIndex.Indexer = "null"
	return c
}
