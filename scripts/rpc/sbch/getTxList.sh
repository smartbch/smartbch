#!/usr/bin/env bash

set -ex

REQ=$(cat <<-END
{
  "jsonrpc":"2.0",
  "method":"sbch_getTxListByHeight",
  "params":["$1"],
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
