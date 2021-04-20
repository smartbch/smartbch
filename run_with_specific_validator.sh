set -eux

go build -o build/smartbchd ./cmd/smartbchd

#todo: for test, not production
rm -rf ~/.smartbchd/

./build/smartbchd init smart_node --chain-id 0x539 \
  --init-balance=10000000000000000000000 \
  --test-keys="$1"

# shellcheck disable=SC2046
./build/smartbchd add-genesis-validator $(./build/smartbchd generate-genesis-validator "$1")

./build/smartbchd start