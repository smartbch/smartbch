package main

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/internal/ethutils"
)

const (
	getTxCountReqFmt = `{"jsonrpc":"2.0", "method":"eth_getTransactionCount", "params":["%s", "latest"], "id":%d}`
	sendRawTxReqFmt  = `{"jsonrpc":"2.0", "method":"eth_sendRawTransaction", "params":["%s"], "id":%d}`
)

var (
	reqID int64
)

type GetTxCountResp struct {
	Result string `json:"result"`
}

type rpcClient struct {
	rpcURL string
	logger log.Logger
}

func (c rpcClient) getNonce(addr gethcmn.Address) (int64, error) {
	id := atomic.AddInt64(&reqID, 1)
	getNonceReq := fmt.Sprintf(getTxCountReqFmt, addr.Hex(), id)
	c.logger.Info("get nonce, req: " + getNonceReq)
	getNonceResp, err := sendPost(c.rpcURL, getNonceReq)
	if err != nil {
		return 0, err
	}

	c.logger.Info("get nonce, resp: " + getNonceResp)
	respObj := GetTxCountResp{}
	err = json.Unmarshal([]byte(getNonceResp), &respObj)
	if err != nil {
		return 0, err
	}

	if strings.HasPrefix(respObj.Result, "0x") {
		respObj.Result = respObj.Result[2:]
	}
	return strconv.ParseInt(respObj.Result, 16, 64)
}

func (c rpcClient) sendRawTx(tx *gethtypes.Transaction) (string, error) {
	txData, err := ethutils.EncodeTx(tx)
	if err != nil {
		return "", err
	}
	txJson, _ := tx.MarshalJSON()
	c.logger.Info("sendRawTx, tx: " + string(txJson))

	id := atomic.AddInt64(&reqID, 1)
	sendRawTxReq := fmt.Sprintf(sendRawTxReqFmt, "0x"+hex.EncodeToString(txData), id)
	c.logger.Info("sendRawTx, req: " + sendRawTxReq)

	sendRawTxResp, err := sendPost(c.rpcURL, sendRawTxReq)
	if err != nil {
		return "", err
	}

	c.logger.Info("sendRawTx, resp: " + sendRawTxResp)
	return sendRawTxResp, nil
}

func sendPost(url string, jsonStr string) (string, error) {
	body := bytes.NewReader([]byte(jsonStr))
	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), err
}
