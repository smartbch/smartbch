#!/bin/bash

# ==============
# Genesis node
# ==============

echo "=============="
echo "Genesis Node"
echo "=============="
echo "Generating Keys"
mkdir keys 
docker-compose run smartbch_genesis gen-test-keys -n 10 > test-keys.txt
echo

 

echo "Init the node, include the keys from the last step as a comma separated list."

# put json node id into json_node_id.txt file
# then tail it out so that bash would recognize it as a string
# there should be a better way to do this...
NID=$(docker-compose run smartbch_genesis init mynode --chain-id 0x2711 \
    --init-balance=10000000000000000000 \
    --test-keys=`paste -d, -s test-keys.txt` \
    --home=/root/.smartbchd --overwrite \
    | tee json_node_id.txt
    )
echo

# getting nodeId from json_node_id file
K1=$(tail -1 json_node_id.txt)

# splitting K1 for node Id
IFS='\"' #colon as delimiter
read -ra BIT <<<"$K1" # split string into :array name BIT
NODEID=${BIT[11]} # choose index 11 of BIT array

echo Genesis node Id: $NODEID
echo $NODEID > genesis_node_id.txt
rm -fr json_node_id.txt
CPK=$(docker-compose run -w /root/.smartbchd/ smartbch_genesis generate-consensus-key-info)
docker-compose run --entrypoint mv smartbch_genesis /root/.smartbchd/priv_validator_key.json /root/.smartbchd/config
echo

echo "Generate genesis validator"

K1=$(head -1 test-keys.txt)
VAL=$(docker-compose run smartbch_genesis generate-genesis-validator $K1 \
  --consensus-pubkey $CPK \
  --staking-coin 10000000000000000000000 \
  --voting-power 1 \
  --introduction "happygenesis" \
  --home /root/.smartbchd
  )
docker-compose run smartbch_genesis add-genesis-validator --home=/root/.smartbchd $VAL

echo "Copy genesis.json"
cp smartbch_genesis_data/config/genesis.json .
mv ./test-keys.txt ./keys
mv ./genesis_node_id.txt ./keys

echo
echo "Genesis node setup Finished!"


# ==============
# Sync-node
# ==============

echo "=============="
echo "Sync Node"
echo "=============="

echo "Init chain id"
docker-compose run smartbch_node init sync_node --chain-id 0x2711

echo "Replace genesis.json"
cp -fr genesis.json smartbch_node_data/config/.
rm genesis.json
# get localhost ip

# replacing line that starts with "seeds =" with $seed_address
echo "Configuring p2p seeds"
seed_address=\"$NODEID@smartbch_genesis:26656\"
sed -i "s/^seeds =.*/seeds = $seed_address/" smartbch_node_data/config/config.toml
echo

# replacing line that starts with "mainnet-rpc-url" with $rpc
echo "Configuring RPC"
rpc=\"smartbch_genesis:8545\"
sudo sed -i "s/^mainnet-rpc-url.*/mainnet-rpc-url = $rpc/" ./smartbch_node_data/config/app.toml
echo
echo "Keys and genesis node id can be found at ./keys"
echo "Sync node setup Finished!"