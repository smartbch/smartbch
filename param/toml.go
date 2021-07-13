package param

import (
	"bytes"
	"text/template"

	"github.com/spf13/viper"
	tmos "github.com/tendermint/tendermint/libs/os"
)

const defaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

# smartbchd history db path
app_data_path = "{{ .AppDataPath }}"

# moeing db path
modb_data_path = "{{ .ModbDataPath }}"

# eth_getLogs max return items
get_logs_max_results = {{ .RpcEthGetLogsMaxResults }}

# retain blocks in TM
retain-blocks = {{ .RetainBlocks }}

# every retain_interval_blocks blocks execute TM blocks prune
retain_interval_blocks = {{ .ChangeRetainEveryN }}

# use liteDB
use_litedb = {{ .UseLiteDB }}

# 
blocks_kept_ads = {{ .NumKeptBlocks }}

# 
blocks_kept_modb = {{ .NumKeptBlocksInMoDB }}

#
sig_cache_size = {{ .SigCacheSize }}

#
trunk_cache_size = {{ .TrunkCacheSize }}

#
prune_every_n = {{ .PruneEveryN }}

#
recheck_threshold = {{ .RecheckThreshold }}

# BCH mainnet rpc url
mainnet-rpc-url = "{{ .MainnetRPCUrl }}"

# BCH mainnet rpc username
mainnet-rpc-username = "{{ .MainnetRPCUsername }}"

# BCH mainnet rpc password
mainnet-rpc-password = "{{ .MainnetRPCPassword }}"

# smartBCH rpc url for epoch get
smartbch-rpc-url = "{{ .SmartBchRPCUrl }}"

# open epoch get to speedup mainnet block catch, work with "smartbch_rpc_url"
watcher-speedup = {{ .Speedup }}

# log validators info every block
log-validators = {{ .LogValidatorsInfo }}
`

var configTemplate *template.Template

func init() {
	var err error
	tmpl := template.New("appConfigFileTemplate")
	if configTemplate, err = tmpl.Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

func ParseConfig() (*AppConfig, error) {
	conf := DefaultAppConfig()
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
