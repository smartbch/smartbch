package main

import (
	"crypto/ecdsa"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/smartbch/smartbch/internal/ethutils"
)

const rpcURL = "http://45.32.38.25:8545"

var (
	//go:embed html/index.html
	indexHTML string

	//go:embed html/result.html
	resultHTML string

	//go:embed json/send_tx_req.json
	sendTxReqJSON string

	//go:embed json/get_tx_count_req.json
	getTxCountReqJSON string

	//go:embed json/send_raw_tx_req.json
	sendRawTxReqJSON string
)

var (
	faucetAddrs []gethcmn.Address
	faucetKeys  []*ecdsa.PrivateKey
)

type GetTxCountResp struct {
	Result string `json:"result"`
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

	getNonceReq := fmt.Sprintf(getTxCountReqJSON, fromAddr.Hex())
	fmt.Println("get nonce req:", getNonceReq)
	getNonceResp, err := sendPost(rpcURL, getNonceReq)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	fmt.Println("get nonce resp:", getNonceResp)
	respObj := GetTxCountResp{}
	err = json.Unmarshal([]byte(getNonceResp), &respObj)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	if strings.HasPrefix(respObj.Result, "0x") {
		respObj.Result = respObj.Result[2:]
	}
	nonce, err := strconv.ParseInt(respObj.Result, 16, 64)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	txData, err := makeAndSignTx(key, uint64(nonce), toAddr)
	sendRawTxReq := fmt.Sprintf(sendRawTxReqJSON, "0x"+hex.EncodeToString(txData))
	fmt.Println("sendRawTx req:", sendRawTxReq)

	sendRawTxResp, err := sendPost(rpcURL, sendRawTxReq)
	if err != nil {
		fmt.Println("err:", err.Error())
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	fmt.Println("sendRawTx resp:", sendRawTxResp)
	_, _ = w.Write([]byte(fmt.Sprintf(resultHTML, sendRawTxResp)))
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: faucet 'key1,key2,key3,...'")
		return
	}

	parsePrivKeys(os.Args[1])
	startServer()
}

func parsePrivKeys(csv string) {
	for _, hexKey := range strings.Split(csv, ",") {
		key, _, err := ethutils.HexToPrivKey(hexKey)
		if err != nil {
			panic(err)
		}

		addr := ethutils.PrivKeyToAddr(key)
		faucetKeys = append(faucetKeys, key)
		faucetAddrs = append(faucetAddrs, addr)
		fmt.Println("parsed faucet addr: ", addr.Hex())
	}
}

func startServer() {
	http.HandleFunc("/faucet", hello)
	http.HandleFunc("/sendBCH", sendBCH)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
