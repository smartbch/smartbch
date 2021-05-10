package main

import (
	"crypto/ecdsa"
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/smartbch/smartbch/internal/ethutils"
)

var (
	rpcURL = "http://45.32.38.25:8545"

	faucetAddrs []gethcmn.Address
	faucetKeys  []*ecdsa.PrivateKey
)

func main() {
	switch len(os.Args) {
	case 1, 2:
		fmt.Print(`
Usage: faucet <rpc-url> <priv-keys-file>
   or: faucet <rpc-url> key1 key2 key3 ...
`)
		return
	case 3:
		rpcURL = os.Args[1]
		parsePrivKeysFromFile(os.Args[2])
	case 4:
		rpcURL = os.Args[1]
		parsePrivKeys(os.Args[2:])
	}

	startServer()
}

func parsePrivKeysFromFile(filename string) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	parsePrivKeys(strings.Split(string(bytes), "\n"))
}

func parsePrivKeys(privKeys []string) {
	for _, hexKey := range privKeys {
		hexKey = strings.TrimSpace(hexKey)
		if len(hexKey) == 64 {
			key, _, err := ethutils.HexToPrivKey(hexKey)
			if err != nil {
				panic(err)
			}

			addr := ethutils.PrivKeyToAddr(key)
			faucetKeys = append(faucetKeys, key)
			faucetAddrs = append(faucetAddrs, addr)
			fmt.Println("faucet addr: ", addr.Hex())
		}
	}
}
