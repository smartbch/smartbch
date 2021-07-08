#!/usr/bin/env bash

pushd ../moeingevm/evmwrap || return; make
popd || return

cp ../moeingevm/evmwrap/host_bridge/libevmwrap.so .
export EVMWRAP=$PWD/libevmwrap.so

golangci-lint run

go fmt ./...
go build ./...

TEST_RANDOM_TXS_COUNT=10 \
TEST_CALL_TRANSFER_RANDOM_COUNT=1 \
UT_CHECK_ALL_BALANCE=0 \
go test ./...
#go test -covermode=atomic -coverprofile=coverage.out ./...
