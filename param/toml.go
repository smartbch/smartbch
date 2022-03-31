package param

import (
	"bytes"
	"text/template"

	"github.com/spf13/viper"
	tmos "github.com/tendermint/tendermint/libs/os"
)

const defaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

# eth_getLogs max return items
get_logs_max_results = {{ .RpcEthGetLogsMaxResults }}

# retain blocks in TM
retain-blocks = {{ .RetainBlocks }}

# every retain_interval_blocks blocks execute TM blocks prune
retain_interval_blocks = {{ .ChangeRetainEveryN }}

# use liteDB
use_litedb = {{ .UseLiteDB }}

# How many recent blocks can be kept in moeingads (to prune the blocks which are older than them)
blocks_kept_ads = {{ .NumKeptBlocks }}

# How many recent blocks can be kept in moeingdb (to prune the blocks which are older than them)
blocks_kept_modb = {{ .NumKeptBlocksInMoDB }}

# The initial entry count in the trunk cache, which buffers the write operations of the last block
trunk_cache_size = {{ .TrunkCacheSize }}

# We try to prune the old blocks of moeingads every n blocks
prune_every_n = {{ .PruneEveryN }}

# smartBCH rpc url for epoch get
smartbch-rpc-url = "{{ .SmartBchRPCUrl }}"
`

var configTemplate *template.Template

func init() {
	var err error
	tmpl := template.New("appConfigFileTemplate")
	if configTemplate, err = tmpl.Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

func ParseConfig(home string) (*AppConfig, error) {
	conf := DefaultAppConfigWithHome(home)
	err := viper.Unmarshal(conf)
	return conf, err
}

func WriteConfigFile(configFilePath string, config *AppConfig) {
	var buffer bytes.Buffer
	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}
	tmos.MustWriteFile(configFilePath, buffer.Bytes(), 0644)
}
