#!/usr/bin/env bash

set -ex

curl -X POST --data '{"jsonrpc":"2.0","method":"net_version","params":[],"id":67}' \
  -H "Content-Type: application/json" http://localhost:8545

cd testdata/sol

truffle test --network=sbch_local
