package main

import (
	"github.com/smartbch/smartbch/param"
	"os"

	"github.com/tendermint/tendermint/libs/log"
)

var (
	DefaultNodeHome = os.ExpandEnv("$HOME/.smartbchd")
)

type Context struct {
	Config *param.ChainConfig
	Logger log.Logger
}

func NewDefaultContext() *Context {
	return NewContext(
		param.DefaultConfig(),
		log.NewTMLogger(log.NewSyncWriter(os.Stdout)),
	)
}

func NewContext(config *param.ChainConfig, logger log.Logger) *Context {
	return &Context{config, logger}
}
