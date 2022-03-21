#!/usr/bin/env bash

set -ex

pushd ../moeingevm/evmwrap || return; make
popd || return

#cp ../moeingevm/evmwrap/host_bridge/libevmwrap.a .
export CGO_LDFLAGS="-L../moeingevm/evmwrap/host_bridge"

golangci-lint run

go fmt ./...
go build ./...

TEST_RANDOM_TXS_COUNT=10 \
TEST_CALL_TRANSFER_RANDOM_COUNT=1 \
UT_CHECK_ALL_BALANCE=0 \
go test -tags params_testnet -p 1 ./...
#go test -covermode=atomic -coverprofile=coverage.out ./...
