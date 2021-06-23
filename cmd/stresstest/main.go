package main

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/tendermint/tendermint/p2p"
)

func main() {
	//randBlocks, fromSize, toSize, txPerBlock, fname := 100, 16, 160, 4, "keys1M.txt"
	//randBlocks, fromSize, toSize, txPerBlock, fname := 100, 5000, 1000_000, 1000, "keys6M.txt"
	//randBlocks, fromSize, toSize, txPerBlock, fname := 1000, 20000, 20000_000, 10000, "keys23M.txt"
	//randBlocks, fromSize, toSize, txPerBlock, fname := 1000, 50000, 50000_000, 10000, "keys60M.txt"

	randBlocks, fromSize, toSize, txPerBlock, fname := 1000, 500, 100_000, -100, "keys6M.txt"

	if len(os.Args) < 2 {
		fmt.Println("Usage: stresstest gen|replay|genkeys|genkeys60")
		return
	}

	if os.Args[1] == "gen" {
		RunRecordBlocks(randBlocks, fromSize, toSize, txPerBlock, fname)
	} else if os.Args[1] == "replay" {
		RunReplayBlocks(fromSize, fname)
	} else if os.Args[1] == "replayWS" {
		RunReplayBlocksWS(getWsURL(), getMinHeight(3), getMinHeight(4))
	} else if os.Args[1] == "queryWS" || os.Args[1] == "queryTxsWS" {
		RunQueryTxsWS(getWsURL(), getMaxHeight(3))
	} else if os.Args[1] == "queryBlocksWS" {
		RunQueryBlocksWS(getWsURL(), getMaxHeight(3), getMinHeight(4), len(os.Args) > 5)
	} else if os.Args[1] == "genkeys10K" {
		GenKeysToFile("keys10K.txt", 10_000)
	} else if os.Args[1] == "genkeys" {
		GenKeysToFile("keys1M.txt", 1_000_000)
	} else if os.Args[1] == "genkeys6" {
		GenKeysToFile("keys6M.txt", 6_000_000)
	} else if os.Args[1] == "genkeys60" {
		GenKeysToFile("keys60M.txt", 60_000_000)
	} else if os.Args[1] == "showNodeKey" {
		showNodeKey(os.Args[2])
	} else {
		panic("invalid argument")
	}
}

func showNodeKey(fname string) {
	nodeKey, err := p2p.LoadNodeKey(fname)
	if err != nil {
		panic(err)
	}
	fmt.Printf("P2P Node ID is %s\n", nodeKey.ID())
}

func getWsURL() string {
	url := "ws://localhost:8546"
	if len(os.Args) > 2 {
		url = os.Args[2]
	}
	return url
}

func getMaxHeight(n int) int {
	maxHeight := math.MaxUint32
	if len(os.Args) > n {
		if h, err := strconv.ParseInt(os.Args[n], 10, 32); err == nil {
			maxHeight = int(h)
		}
	}
	return maxHeight
}

func getMinHeight(n int) int {
	minHeight := 1
	if len(os.Args) > n {
		if h, err := strconv.ParseInt(os.Args[n], 10, 32); err == nil {
			minHeight = int(h)
		}
	}
	return minHeight
}
