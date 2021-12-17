package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/watcher"
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

	client := watcher.NewRpcClient(rpcURL, rpcUsername, rpcPassword, "text/plain;", log.NewNopLogger())
	printAllOpReturn(client, startH, endH)
	//client.PrintAllOpReturn(519995, 679995)
}

func printAllOpReturn(client *watcher.RpcClient, startHeight, endHeight int64) {
	for h := startHeight; h < endHeight; h++ {
		fmt.Printf("Height: %d\n", h)
		hash, err := client.GetBlockHash(h)
		if err != nil {
			fmt.Printf("Error when getBlockHashOfHeight %d %s\n", h, err.Error())
			continue
		}
		bi, err := client.GetBlockInfo(hash)
		if err != nil {
			fmt.Printf("Error when getBlock %d %s\n", h, err.Error())
			continue
		}
		found := false
		for _, txid := range bi.Tx {
			tx, err := client.GetTxInfo(txid, hash)
			if err != nil {
				fmt.Printf("Error when getTx %s %s\n", txid, err.Error())
				continue
			}
			for _, vout := range tx.VoutList {
				asm, ok := vout.ScriptPubKey["asm"]
				if !ok || asm == nil {
					continue
				}
				script, ok := asm.(string)
				if !ok {
					continue
				}
				if strings.HasPrefix(script, "OP_RETURN") {
					found = true
					fmt.Println(script)
				}
			}
		}
		if !found {
			fmt.Println("OP_RETURN not found!")
		}
	}
}
