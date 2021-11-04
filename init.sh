#!/bin/bash

echo "Generating Keys"
docker-compose run smartbch_genesis gen-test-keys -n 10 > test-keys.txt
echo

echo "Init the node, include the keys from the last step as a comma separated list."
docker-compose run smartbch_genesis init mynode --chain-id 0x2711 \
    --init-balance=10000000000000000000 \
    --test-keys=`paste -d, -s test-keys.txt` \
    --home=/root/.smartbchd --overwrite
echo

CPK=$(docker-compose run -w /root/.smartbchd/ smartbch_genesis generate-consensus-key-info)
docker-compose run --entrypoint mv smartbch_genesis /root/.smartbchd/priv_validator_key.json /root/.smartbchd/config
echo

echo "Generate genesis validator"

K1=$(head -1 test-keys.txt)
VAL=$(docker-compose run smartbch_genesis generate-genesis-validator $K1 \
  --consensus-pubkey $CPK \
  --staking-coin 10000000000000000000000 \
  --voting-power 1 \
  --introduction "tester" \
  --home /root/.smartbchd
  )
docker-compose run smartbch_genesis add-genesis-validator --home=/root/.smartbchd $VAL
echo

echo "Finished!"
