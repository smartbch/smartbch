#!/usr/bin/env bash

set -ex

curl -X POST --data '{"jsonrpc":"2.0","method":"net_version","params":[],"id":67}' \
  -H "Content-Type: application/json" http://localhost:8545

TESTDATA_DIR=$PWD/testdata

cd $TESTDATA_DIR/basic;   truffle test
cd $TESTDATA_DIR/counter; truffle test
cd $TESTDATA_DIR/erc20;   truffle test
cd $TESTDATA_DIR/uniswap; truffle test
cd $TESTDATA_DIR/seps;    truffle test
cd $TESTDATA_DIR/staking; truffle test
