#!/usr/bin/env bash

cd evmwrap; make
cd ..

cp evmwrap/host_bridge/libevmwrap.so .
cp evmwrap/host_bridge/libevmwrap.so app
cp evmwrap/host_bridge/libevmwrap.so cmd/moeingd
cp evmwrap/host_bridge/libevmwrap.so cmd/moeingcli
cp evmwrap/host_bridge/libevmwrap.so ebp
cp evmwrap/host_bridge/libevmwrap.so ebp/ebptests
cp evmwrap/host_bridge/libevmwrap.so rpc/api
cp evmwrap/host_bridge/libevmwrap.so rpc/api/filters
go build ./...
go test ./...
