#!/usr/bin/env bash

export EVMWRAP=libevmwrap.so
go build -tags cppbtree github.com/smartbch/smartbch/cmd/smartbchd

NODE_HOME=~/.smartbchd/
TEST_KEYS="\
0xe3d9be2e6430a9db8291ab1853f5ec2467822b33a1a08825a22fab1425d2bff9,\
0x5a09e9d6be2cdc7de8f6beba300e52823493cd23357b1ca14a9c36764d600f5e,\
0x7e01af236f9c9536d9d28b07cea24ccf21e21c9bc9f2b2c11471cd82dbb63162,\
0x1f67c31733dc3fd02c1f9ce9cb9e05b1d2f1b7b5463fef8acf6cf17f3bd01467,\
0x8aa75c97b22e743e2d14a0472406f03cc5b4a050e8d4300040002096f50c0c6f,\
0x84a453fe127ae889de1cfc28590bf5168d2843b50853ab3c5080cd5cf9e18b4b,\
0x40580320383dbedba7a5305a593ee2c46581a4fd56ff357204c3894e91fbaf48,\
0x0e3e6ba041d8ad56b0825c549b610e447ec55a72bb90762d281956c56146c4b3,\
0x867b73f28bea9a0c83dfc233b8c4e51e0d58197de7482ebf666e40dd7947e2b6,\
0xa3ff378a8d766931575df674fbb1024f09f7072653e1aa91641f310b3e1c5275"

rm -rf $NODE_HOME
echo 'initializing node ...'
./smartbchd init m1 --home=$NODE_HOME --chain-id 0x2711 \
  --init-balance=100000000000000000000 \
  --test-keys=$TEST_KEYS # --test-keys-file='keys10K.txt,keys1M.txt'
sed -i '.bak' 's/timeout_commit = "5s"/timeout_commit = "1s"/g' $NODE_HOME/config/config.toml

echo 'generating consensus key info ...'
CPK=$(./smartbchd generate-consensus-key-info --home=$NODE_HOME)
echo 'generating genesis validator ...'
VAL=$(./smartbchd generate-genesis-validator --home=$NODE_HOME \
  --validator-address=0xEAB1B601da26611D134299845035214a046508B8 \
  --consensus-pubkey $CPK \
  --staking-coin 10000000000000000000000 \
  --voting-power 1 \
  --introduction="tester"
  )

mv ./priv_validator_key.json $NODE_HOME/config/

echo 'adding genesis validator ...'
./smartbchd add-genesis-validator --home=$NODE_HOME $VAL

#export NODIASM=1
#export NOSTACK=1
#export NOINSTLOG=1
echo 'starting node ...'
./smartbchd start --home $NODE_HOME --unlock $TEST_KEYS --https.addr=off --wss.addr=off \
  --log_level='json-rpc:debug,*:info'
