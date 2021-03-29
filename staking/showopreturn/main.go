package main

import (
	"github.com/smartbch/smartbch/staking"
)

// To summmarize all the op_return vouts since the first SLP transaction:
// gawk 'BEGIN {start=0; count=0; all=0} $1=="Height:"&&$2==543376 {start=1;} $2==5262419&&start==1 {count++} $1=="OP_RETURN"&&start==1 {all++} END {print count" "all;}'

// grep TX data.log  |gawk 'BEGIN {print "["} {print substr($0,3)","; last=$2} END {print last"]"}' > tx.json
// grep BLOCK data.log  |gawk 'BEGIN {print "["} {print $2","; last=$2} END {print last"]"}' > block.json

func main() {
	client := staking.NewRpcClient("http://127.0.0.1:8332/", "user", "dummypassword")

	client.PrintAllOpReturn(1, 41000)
	//client.PrintAllOpReturn(519995, 679995)
}
