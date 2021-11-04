#!/bin/bash

while [[ "$#" -gt 0 ]]; do
    case $1 in
        -t|--target) target="$2"; shift ;;
        *) echo "Unknown parameter passed: $1, please pass  -t or --target to specify target bch node ip"; exit 1 ;;
    esac
    shift
done

echo "BCH Node RPC URL: $target"

echo "Init chain id"
docker-compose run smartbch_node init sync_node --chain-id 0x2710


echo "Adding config files"
sudo cp -rf mainnet-config/* smartbch_node_data/

# replacing line that starts with "mainnet-rpc-url" with $rpc
echo "Configuring RPC"
rpc=\"$target:8332\"
sudo sed -i "s/^mainnet-rpc-url.*/mainnet-rpc-url = $rpc/" ./smartbch_node_data/config/app.toml
echo

echo "Finished!"
