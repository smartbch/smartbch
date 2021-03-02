#!/usr/bin/env bash

pushd ../MoeingEVM/evmwrap || return; make clean; make
popd || return

cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so .
export EVMWRAP=$PWD/libevmwrap.so

go build ./...
go test ./...
