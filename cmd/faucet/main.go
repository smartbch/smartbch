package main

import (
	"crypto/ecdsa"
	_ "embed"
	"fmt"
	gethcmn "github.com/ethereum/go-ethereum/common"
	"os"

	"github.com/smartbch/smartbch/internal/ethutils"
)

var (
	faucetAddrs []gethcmn.Address
	faucetKeys  []*ecdsa.PrivateKey
)

func main() {
	parsePrivKeys(os.Args[1:])
	startServer()
}

func parsePrivKeys(privKeys []string) {
	for _, hexKey := range privKeys {
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
