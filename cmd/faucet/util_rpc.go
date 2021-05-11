package main

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/smartbch/smartbch/internal/ethutils"
)

var (
	//go:embed json/get_tx_count_req.json
	getTxCountReqJSON string

	//go:embed json/send_raw_tx_req.json
	sendRawTxReqJSON string
)

type GetTxCountResp struct {
	Result string `json:"result"`
}

func getNonce(addr gethcmn.Address) (int64, error) {
	getNonceReq := fmt.Sprintf(getTxCountReqJSON, addr.Hex())
	fmt.Println("get nonce req: ", getNonceReq)
	getNonceResp, err := sendPost(rpcURL, getNonceReq)
	if err != nil {
		return 0, err
	}

	fmt.Println("get nonce resp:", getNonceResp)
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

func sendRawTx(tx *gethtypes.Transaction) (string, error) {
	txData, err := ethutils.EncodeTx(tx)
	if err != nil {
		return "", err
	}
	txJson, _ := tx.MarshalJSON()
	fmt.Println("sendRawTx tx:", string(txJson))

	sendRawTxReq := fmt.Sprintf(sendRawTxReqJSON, "0x"+hex.EncodeToString(txData))
	fmt.Println("sendRawTx req:", sendRawTxReq)

	sendRawTxResp, err := sendPost(rpcURL, sendRawTxReq)
	if err != nil {
		return "", err
	}

	fmt.Println("sendRawTx resp:", sendRawTxResp)
	return sendRawTxResp, nil
}
