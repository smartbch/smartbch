#!/bin/sh

if [ ! -f regtest.init ]; then
  echo "init_regtest.sh: Initializing local single-node testnet"
  touch regtest.init
  if [ ! -s test-keys.txt ]; then
    echo "init_regtest.sh: Generating private keys..."
    smartbchd gen-test-keys -n 10 >> test-keys.txt
  fi;

  rm -rf /root/.smartbchd/*

  smartbchd init mynode --chain-id 0x2711 \
      --init-balance=10000000000000000000 \
      --test-keys=$(paste -d, -s test-keys.txt) \
      --home=/root/.smartbchd --overwrite

  CPK=$(smartbchd generate-consensus-key-info) && mv priv_validator_key.json /root/.smartbchd/config
  K1=$(head -1 test-keys.txt)
  VAL=$(smartbchd generate-genesis-validator $K1 \
    --consensus-pubkey $CPK \
    --staking-coin 10000000000000000000000 \
    --voting-power 1 \
    --introduction "tester" \
    --home /root/.smartbchd
    )
  smartbchd add-genesis-validator --home=/root/.smartbchd $VAL
fi;

smartbchd start
