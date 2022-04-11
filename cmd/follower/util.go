package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tmcli "github.com/tendermint/tendermint/libs/cli"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/param"
)

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

func PersistentPreRunEFn(context *Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		config, err := interceptLoadConfig()
		if err != nil {
			return err
		}
		logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
		// logger = log.NewFilter(logger, log.AllowInfo())
		logger, err = tmflags.ParseLogLevel(config.AppConfig.LogLevel, logger, "debug")
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
	rootDir := viper.GetString(tmcli.HomeFlag)
	appConfigFilePath := filepath.Join(rootDir, "config/app.toml")
	var appConf *param.AppConfig
	if _, err := os.Stat(appConfigFilePath); os.IsNotExist(err) {
		appConf, _ = param.ParseConfig(rootDir)
		param.WriteConfigFile(appConfigFilePath, appConf)
	}
	if appConf == nil {
		viper.SetConfigName("app")
		err = viper.MergeInConfig()
		appConf, _ = param.ParseConfig(rootDir)
	}
	conf = param.DefaultConfig()
	conf.AppConfig = appConf
	return conf, err
}
