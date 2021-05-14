package main

import (
	"crypto/ecdsa"
	_ "embed"
	"errors"
	"io/ioutil"
	"math/big"
	"os"
	"strings"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/internal/ethutils"
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
	cmd.Flags().Bool("maintenance", false, "start in maintenance mode")

	_ = cmd.MarkFlagRequired("port")

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func startFaucetServer(cmd *cobra.Command, args []string) error {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	port, err := cmd.Flags().GetInt64("port")
	if err != nil {
		return err
	}
	logger.Info("flag", "port", port)

	rpcURL, err := cmd.Flags().GetString("rpc-url")
	if err != nil {
		return err
	}
	logger.Info("flag", "rpc-url", rpcURL)

	privKeysFile, err := cmd.Flags().GetString("priv-keys-file")
	if err != nil {
		return err
	}
	logger.Info("flag", "priv-keys-file", privKeysFile)

	sendAmt, err := cmd.Flags().GetString("send-amount")
	sendAmtBig := big.NewInt(0)
	_, ok := sendAmtBig.SetString(sendAmt, 10)
	if !ok {
		return errors.New("incorrect send amount")
	}
	logger.Info("flag", "send-amount", sendAmtBig.String())

	isMaintenance, _ := cmd.Flags().GetBool("maintenance")

	var keys []*ecdsa.PrivateKey
	var addrs []gethcmn.Address
	if privKeysFile != "" {
		keys, addrs = parsePrivKeysFromFile(logger, privKeysFile)
	} else {
		keys, addrs = parsePrivKeys(logger, args)
	}

	server := faucetServer{
		port:        port,
		faucetKeys:  keys,
		faucetAddrs: addrs,
		sendAmt:     sendAmtBig,
		logger:      logger,
		maintenance: isMaintenance,
		rpcClient: rpcClient{
			rpcURL: rpcURL,
			logger: logger,
		},
	}
	server.start()
	return nil
}

func parsePrivKeysFromFile(logger log.Logger, filename string) (keys []*ecdsa.PrivateKey, addrs []gethcmn.Address) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return parsePrivKeys(logger, strings.Split(string(bytes), "\n"))
}

func parsePrivKeys(logger log.Logger, privKeys []string) (keys []*ecdsa.PrivateKey, addrs []gethcmn.Address) {
	for _, hexKey := range privKeys {
		hexKey = strings.TrimSpace(hexKey)
		if len(hexKey) == 64 {
			key, _, err := ethutils.HexToPrivKey(hexKey)
			if err != nil {
				panic(err)
			}

			addr := ethutils.PrivKeyToAddr(key)
			keys = append(keys, key)
			addrs = append(addrs, addr)
			logger.Info("faucet account", "addr", addr.Hex())
		}
	}
	return
}
