package main

import (
	"crypto/ecdsa"
	_ "embed"
	"fmt"
	"math/big"
	"math/rand"
	"net/http"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/internal/ethutils"
)

var (
	//go:embed html/index.html
	indexHTML string

	//go:embed html/maintance.html
	maintenanceHTML string

	//go:embed html/result.html
	resultHTML string
)

var (
	chainID  = big.NewInt(10001)
	gasPrice = big.NewInt(0)
	gasLimit = uint64(1000000)
)

type faucetServer struct {
	port        int64
	faucetAddrs []gethcmn.Address
	faucetKeys  []*ecdsa.PrivateKey
	rpcClient   rpcClient
	sendAmt     *big.Int
	logger      log.Logger
	maintenance bool
}

func (s faucetServer) start() {
	http.HandleFunc("/faucet", func(w http.ResponseWriter, r *http.Request) {
		if s.maintenance {
			_, _ = fmt.Fprint(w, maintenanceHTML)
		} else {
			_, _ = fmt.Fprint(w, indexHTML)
		}
	})
	http.HandleFunc("/sendBCH", func(w http.ResponseWriter, r *http.Request) {
		s.sendBCH(w, r)
	})

	//fmt.Println("start server on port ", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
	if err != nil {
		panic(err)
	}
}

func (s faucetServer) sendBCH(w http.ResponseWriter, req *http.Request) {
	toAddrHex, err := getQueryParam(req, "addr")
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	toAddr := gethcmn.HexToAddress(toAddrHex)
	idx := rand.Intn(len(s.faucetKeys))
	faucetAddr := s.faucetAddrs[idx]
	faucetKey := s.faucetKeys[idx]
	s.logger.Info("sending BCH to: " + toAddrHex + ", faucetAddr:" + faucetAddr.Hex())

	nonce, err := s.rpcClient.getNonce(faucetAddr)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	tx, err := s.makeAndSignTx(faucetKey, uint64(nonce), toAddr)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	sendRawTxResp, err := s.rpcClient.sendRawTx(tx)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write([]byte(fmt.Sprintf(resultHTML, sendRawTxResp)))
}

func (s faucetServer) makeAndSignTx(privKey *ecdsa.PrivateKey, nonce uint64, toAddr gethcmn.Address) (*gethtypes.Transaction, error) {
	txData := &gethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       &toAddr,
		Value:    s.sendAmt,
		Data:     nil,
	}
	tx := gethtypes.NewTx(txData)
	tx, err := ethutils.SignTx(tx, chainID, privKey)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func getQueryParam(req *http.Request, key string) (string, error) {
	if req.Method == "POST" {
		if err := req.ParseForm(); err != nil {
			return "", err
		}
		return req.Form.Get(key), nil
	}

	return req.URL.Query().Get(key), nil
}
