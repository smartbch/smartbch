package main

import (
	"crypto/ecdsa"
	_ "embed"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"

	"github.com/smartbch/smartbch/internal/ethutils"
)

var (
	rpcURL  = "https://moeing.app:9545"
	sendAmt = big.NewInt(10000000000000000)

	faucetAddrs []gethcmn.Address
	faucetKeys  []*ecdsa.PrivateKey
)

func main() {
	cmd := &cobra.Command{
		Use:  "faucet",
		RunE: startFaucetServer,
	}

	cmd.Flags().Int64("port", 8080, "faucet server listening port")
	cmd.Flags().String("rpc-url", "https://moeing.app:9545", "testnet RPC URL")
	cmd.Flags().String("priv-keys-file", "", "private keys file, one key per line")
	cmd.Flags().String("send-amount", "10000000000000000", "the amount of BCH send per request")

	_ = cmd.MarkFlagRequired("port")

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func startFaucetServer(cmd *cobra.Command, args []string) error {
	port, err := cmd.Flags().GetInt64("port")
	if err != nil {
		return err
	}
	fmt.Println("port: ", port)

	_rpcURL, err := cmd.Flags().GetString("rpc-url")
	if err != nil {
		return err
	}
	rpcURL = _rpcURL
	fmt.Println("rpc-url: ", rpcURL)

	privKeysFile, err := cmd.Flags().GetString("priv-keys-file")
	if err != nil {
		return err
	}
	fmt.Println("priv-keys-file: ", privKeysFile)

	_sendAmt, err := cmd.Flags().GetString("send-amount")
	_, ok := sendAmt.SetString(_sendAmt, 10)
	if !ok {
		panic("incorrect send amount?")
	}
	fmt.Println("send-amount: ", sendAmt.String())

	if privKeysFile != "" {
		parsePrivKeysFromFile(privKeysFile)
	} else {
		parsePrivKeys(args)
	}

	startServer(port)
	return nil
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
