#!/usr/bin/env bash

pushd ../moeingevm/evmwrap || return; make
popd || return

cp ../moeingevm/evmwrap/host_bridge/libevmwrap.so .
export EVMWRAP=$PWD/libevmwrap.so

golangci-lint run

go fmt ./...
go build ./...
go test ./...
