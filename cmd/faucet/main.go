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
	faucetAddrs []gethcmn.Address
	faucetKeys  []*ecdsa.PrivateKey
)

func main() {
	switch len(os.Args) {
	case 1:
		fmt.Print(`
Usage: faucet <priv-keys-file>
   or: faucet key1 key2 key3 ...
`)
		return
	case 2:
		parsePrivKeysFromFile(os.Args[1])
	case 3:
		parsePrivKeys(os.Args[1:])
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
