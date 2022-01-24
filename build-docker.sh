#!/usr/bin/env bash
# Build a docker image using Dockerfile.optimized
# usage:
# ./build-docker.sh {amber|mainnet}

set -eux

NETWORK=${1:-"mainnet"}
if [ $NETWORK == "amber" ]; then
    docker build -f Dockerfile.optimized --build-arg SMARTBCH_VERSION=amber --build-arg CONFIG_VERSION=0.0.4 -t smartbch-amber:latest .
elif [ $NETWORK == "mainnet" ]; then
    docker build -f Dockerfile.optimized -t smartbch:latest .
else
    echo "Invalid argument, usage: ./build-docker.sh {amber|mainnet}"
fi
