package main

import (
	_ "embed"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
)

var (
	//go:embed html/index.html
	indexHTML string

	//go:embed html/result.html
	resultHTML string
)

func startServer() {
	http.HandleFunc("/faucet", hello)
	http.HandleFunc("/sendBCH", sendBCH)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func hello(w http.ResponseWriter, req *http.Request) {
	_, _ = fmt.Fprint(w, indexHTML)
}

func sendBCH(w http.ResponseWriter, req *http.Request) {
	fmt.Println("time:", time.Now())
	toAddrHex, err := getQueryParam(req, "addr")
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	fmt.Println("addr:", toAddrHex)
	toAddr := gethcmn.HexToAddress(toAddrHex)

	idx := rand.Intn(len(faucetKeys))
	fromAddr := faucetAddrs[idx]
	key := faucetKeys[idx]

	nonce, err := getNonce(fromAddr)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	txData, err := makeAndSignTx(key, uint64(nonce), toAddr)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	sendRawTxResp, err := sendRawTx(txData)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write([]byte(fmt.Sprintf(resultHTML, sendRawTxResp)))
}
