#!/usr/bin/env bash

pushd ../MoeingEVM/evmwrap || return; make
popd || return

cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so ../MoeingEVM
cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so ../MoeingEVM/ebp
cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so ../MoeingEVM/ebp/ebptests
cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so ../MoeingEVM/evmwrap/evmwrap/evmtests
cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so ../MoeingEVM/evmwrap/evmwrap/host_bridge
cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so .
cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so app
cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so cmd/moeingd
cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so cmd/moeingcli
cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so rpc/
cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so rpc/api
cp ../MoeingEVM/evmwrap/host_bridge/libevmwrap.so rpc/api/filters

go build ./...
go test ./...
