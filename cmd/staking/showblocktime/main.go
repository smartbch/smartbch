package main

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/watcher"
)

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
	printBlockTime(client, startH, endH)
}

func printBlockTime(client *watcher.RpcClient, startHeight, endHeight int64) {
	lastBlockTime := int64(0)
	diffTimeMin := int64(math.MaxInt64)
	diffTimeMax := int64(0)
	diffTimeSum := int64(0)
	diffTimeCount := 0

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

		if lastBlockTime > 0 {
			diffTime := bi.Time - lastBlockTime
			diffTimeSum += diffTime
			diffTimeCount++
			if diffTime < diffTimeMin {
				diffTimeMin = diffTime
			}
			if diffTime > diffTimeMax {
				diffTimeMax = diffTime
			}
			fmt.Println("block time: ", diffTime)
		}
		lastBlockTime = bi.Time
	}

	fmt.Println("blocks:", diffTimeCount)
	fmt.Println("block time max:", diffTimeMax)
	fmt.Println("block time min:", diffTimeMin)
	fmt.Println("block time avg:", float64(diffTimeSum)/float64(diffTimeCount))
}
