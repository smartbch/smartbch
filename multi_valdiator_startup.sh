#!/usr/bin/env bash

set -eux

export EVMWRAP=libevmwrap.so
# go build -tags cppbtree github.com/smartbch/smartbch/cmd/smartbchd

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

 #init key:0xAb5D62788E207646fA60EB3eEbDC4358C7F5686c
 #init key:0x3E144eB45c5fF912B2b29B2823FA674C972e9EC0
 #init key:0x0C60B56403637dC9059fff3603a58DB3D5D76D38
 #init key:0xEAB1B601da26611D134299845035214a046508B8
 #init key:0x44F9ba3cfa79f1504f1C2d1eb0389fbB32e5A00c
 #init key:0xb53e0a1dCf2ad9FA6ec8da77121B1765E68e768f
 #init key:0x09F236e4067f5FcA5872d0c09f92Ce653377aE41
 #init key:0xc5787370b6188b2b6F947117BB2F68ADF732b207
 #init key:0xB9D95550558d2a163F77F5A523dFe605746cB95B
 #init key:0xee5d82886766296640d8CA194e997341a0DeDEDe

rm -rf $NODE_HOME
echo 'initializing node ...'
./smartbchd init m1 --home=$NODE_HOME --chain-id 0x2711 \
  --init-balance=10000000000000000000000 \
  --test-keys=$TEST_KEYS # --test-keys-file='keys10K.txt,keys1M.txt'
sed -i '.bak' 's/timeout_commit = "5s"/timeout_commit = "1s"/g' $NODE_HOME/config/config.toml

echo 'generating consensus key info ...'
CPK=$(./smartbchd generate-consensus-key-info --home=$NODE_HOME)
echo 'generating genesis validator ...'
VAL=$(./smartbchd generate-genesis-validator --home=$NODE_HOME \
  --validator-address=0xAb5D62788E207646fA60EB3eEbDC4358C7F5686c \
  --consensus-pubkey $CPK \
  --staking-coin 10000000000000000000000 \
  --voting-power 10\
  --introduction="tester1"
  )

mv ./priv_validator_key.json $NODE_HOME/config/

echo 'adding genesis validator ...'
./smartbchd add-genesis-validator --home=$NODE_HOME $VAL

echo 'generating and add second validator ...'
CPK=$(./smartbchd generate-consensus-key-info --home=$NODE_HOME)
VAL=$(./smartbchd generate-genesis-validator --home=$NODE_HOME \
  --validator-address=0x3E144eB45c5fF912B2b29B2823FA674C972e9EC0 \
  --consensus-pubkey $CPK \
  --staking-coin 10000000000000000000000 \
  --voting-power 1 \
  --introduction="tester2"
  )
./smartbchd add-genesis-validator --home=$NODE_HOME $VAL

#echo 'generating and add third validator ...'
#CPK=$(./smartbchd generate-consensus-key-info --home=$NODE_HOME)
#VAL=$(./smartbchd generate-genesis-validator --home=$NODE_HOME \
#  --validator-address=0x0C60B56403637dC9059fff3603a58DB3D5D76D38 \
#  --consensus-pubkey $CPK \
#  --staking-coin 10000000000000000000000 \
#  --voting-power 1 \
#  --introduction="tester3"
#  )
#./smartbchd add-genesis-validator --home=$NODE_HOME $VAL
#
#echo 'generating and add forth validator ...'
#CPK=$(./smartbchd generate-consensus-key-info --home=$NODE_HOME)
#VAL=$(./smartbchd generate-genesis-validator --home=$NODE_HOME \
#  --validator-address=0xEAB1B601da26611D134299845035214a046508B8 \
#  --consensus-pubkey $CPK \
#  --staking-coin 10000000000000000000000 \
#  --voting-power 1 \
#  --introduction="tester4"
#  )
#./smartbchd add-genesis-validator --home=$NODE_HOME $VAL

#export NODIASM=1
#export NOSTACK=1
#export NOINSTLOG=1
OTHER_NODE_HOME=~/.smart/
rm -rf $OTHER_NODE_HOME
./smartbchd init m2 --home=$OTHER_NODE_HOME --chain-id 0x2711 \
  --init-balance=10000000000000000000000 \
  --test-keys=$TEST_KEYS # --test-keys-file='keys10K.txt,keys1M.txt'

cp $NODE_HOME/config/genesis.json $OTHER_NODE_HOME/config/

echo $(./smartbchd --home=$NODE_HOME node_key)

echo 'starting node ...'
./smartbchd start --home $NODE_HOME --unlock $TEST_KEYS --https.addr=off --wss.addr=off \
  --log_level='json-rpc:debug,*:info'


# ./smartbchd start --home=/Users/matrix/.smart  --http.addr=tcp://127.0.0.1:28548 --wss.addr=tcp://127.0.0.1:28549 --https.addr=tcp://127.0.0.1:29545 --ws.addr=tcp://127.0.0.1:29546 --rpc.laddr=tcp://127.0.0.1:26667 --p2p.laddr=tcp://127.0.0.1:26666 \
  --log_level='json-rpc:debug,*:info'