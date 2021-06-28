package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/smartbch/smartbch/staking"
)

// To summmarize all the op_return vouts since the first SLP transaction:
// gawk 'BEGIN {start=0; count=0; all=0} $1=="Height:"&&$2==543376 {start=1;} $2==5262419&&start==1 {count++} $1=="OP_RETURN"&&start==1 {all++} END {print count" "all;}'

// grep TX data.log  |gawk 'BEGIN {print "["} {print substr($0,3)","; last=$2} END {print last"]"}' > tx.json
// grep BLOCK data.log  |gawk 'BEGIN {print "["} {print $2","; last=$2} END {print last"]"}' > block.json

func main() {
	if len(os.Args) != 6 {
		fmt.Println("Usage: showopreturn <rpcURL> <username> <password> <startHeight> <endHeight>")
		return
	}

	rpcURL := os.Args[1]
	rpcUsername := os.Args[2]
	rpcPassword := os.Args[3]

	startH, err := strconv.ParseInt(os.Args[4], 10, 32)
	if err != nil {
		panic(err)
	}

	endH, err := strconv.ParseInt(os.Args[5], 10, 32)
	if err != nil {
		panic(err)
	}

	client := staking.NewRpcClient(rpcURL, rpcUsername, rpcPassword, "text/plain;")
	client.PrintAllOpReturn(startH, endH)
	//client.PrintAllOpReturn(519995, 679995)
}
