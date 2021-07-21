package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tmcmds "github.com/tendermint/tendermint/cmd/tendermint/commands"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/smartbch/smartbch/param"
)

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func ExportGenesisFile(genDoc *tmtypes.GenesisDoc, genFile string) error {
	if err := genDoc.ValidateAndComplete(); err != nil {
		return err
	}
	return genDoc.SaveAs(genFile)
}

func TrapSignal(cleanupFunc func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		if cleanupFunc != nil {
			cleanupFunc()
		}
		exitCode := 128
		switch sig {
		case syscall.SIGINT:
			exitCode += int(syscall.SIGINT)
		case syscall.SIGTERM:
			exitCode += int(syscall.SIGTERM)
		}
		os.Exit(exitCode)
	}()
}

func InitializeNodeValidatorFiles(config *cfg.Config,
) (nodeID string, valPubKey crypto.PubKey, err error) {

	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		return nodeID, valPubKey, err
	}

	nodeID = string(nodeKey.ID())
	//server.UpgradeOldPrivValFile(config)

	pvKeyFile := config.PrivValidatorKeyFile()
	if err := tmos.EnsureDir(filepath.Dir(pvKeyFile), 0777); err != nil {
		return nodeID, valPubKey, nil
	}

	pvStateFile := config.PrivValidatorStateFile()
	if err := tmos.EnsureDir(filepath.Dir(pvStateFile), 0777); err != nil {
		return nodeID, valPubKey, nil
	}

	valPubKey, _ = privval.LoadOrGenFilePV(pvKeyFile, pvStateFile).GetPubKey()

	return nodeID, valPubKey, nil
}

func PersistentPreRunEFn(context *Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		config, err := interceptLoadConfig()
		if err != nil {
			return err
		}
		logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
		// logger = log.NewFilter(logger, log.AllowInfo())
		logger, err = tmflags.ParseLogLevel(config.NodeConfig.LogLevel, logger, cfg.DefaultLogLevel)
		if err != nil {
			return err
		}
		logger = logger.With("module", "main")
		context.Config = config
		context.Logger = logger
		return nil
	}
}

func interceptLoadConfig() (conf *param.ChainConfig, err error) {
	tmpConf := cfg.DefaultConfig()
	err = viper.Unmarshal(tmpConf)
	if err != nil {
		panic(err)
	}
	rootDir := tmpConf.RootDir
	configFilePath := filepath.Join(rootDir, "config/config.toml")
	var nodeConfig *cfg.Config
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// the following parse config is needed to create directories
		nodeConfig, _ = tmcmds.ParseConfig() // NOTE: ParseConfig() creates dir/files as necessary.
		//conf.ProfListenAddress = "localhost:6060"
		nodeConfig.P2P.RecvRate = 5120000
		nodeConfig.P2P.SendRate = 5120000
		//conf.TxIndex.IndexAllKeys = true
		nodeConfig.Consensus.TimeoutCommit = 5 * time.Second
		cfg.WriteConfigFile(configFilePath, nodeConfig)
	}

	if nodeConfig == nil {
		nodeConfig, err = tmcmds.ParseConfig()
		if err != nil {
			panic(err)
		}
	}

	appConfigFilePath := filepath.Join(rootDir, "config/app.toml")
	var appConf *param.AppConfig
	if _, err := os.Stat(appConfigFilePath); os.IsNotExist(err) {
		appConf, _ = param.ParseConfig()
		param.WriteConfigFile(appConfigFilePath, appConf)
	}
	if appConf == nil {
		viper.SetConfigName("app")
		err = viper.MergeInConfig()
		appConf, _ = param.ParseConfig()
	}
	conf = &param.ChainConfig{}
	conf.NodeConfig = nodeConfig
	conf.AppConfig = appConf

	return conf, err
}
