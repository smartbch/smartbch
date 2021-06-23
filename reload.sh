set -eux

go build --tags cppbtree -o ./smartbchd  ./cmd/smartbchd
rm -rf ~/.smartbchd
./smartbchd init man --chain-id=0x2711
cp ./genesis.json ~/.smartbchd/config/
cp ./config.toml ~/.smartbchd/config/
./smartbchd start --mainnet-url=http://34.88.14.23:1234 --smartbch-url=http://34.88.14.23:8545 --watcher-speedup=true