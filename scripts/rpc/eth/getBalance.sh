#!/usr/bin/env bash

set -ex

REQ=$(cat <<-END
{
  "jsonrpc":"2.0",
  "method":"eth_getBalance",
  "params":["$1", "latest"],
  "id":1
}
END
)

# echo ">>>" $REQ
# printf "<<< "
curl -X POST \
	--data "$REQ" \
	-H "Content-Type: application/json" \
	"${RPC_URL:-http://moeing.app:8545}"
