package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const (
	sendRawTxReqFmt = `{"jsonrpc":"2.0", "method":"eth_sendRawTransaction", "params":["%s"], "id":%d}`
	getTxListReqFmt = `{"jsonrpc":"2.0", "method":"sbch_getTxListByHeight", "params":["0x%x"], "id":%d}`
)

var reqID uint64

func RunReplayBlocksWS(url string) {
	fmt.Println("connecting to ", url)

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	blkDB := NewBlockDB(blockDir)
	allBlocks := getTotalHeight(blkDB)

	h := uint32(0)
	retryCount := 10
	okTxCount := 0
	failedTxCount := 0
	startTime := time.Now().Unix()
	limiter := time.Tick(1 * time.Millisecond)

	for {
		h++
		blk := blkDB.LoadBlock(h)
		if blk == nil {
			break
		}

		for i, tx := range blk.TxList {
			<-limiter
			tps := 0
			timeElapsed := time.Now().Unix() - startTime
			if timeElapsed > 0 {
				tps = okTxCount / int(timeElapsed)
			}
			fmt.Printf("\rblock: %d, tx: %d; total sent tx: %d, total failed tx: %d, time: %ds, tps:%d, progress:%f%%",
				h, i, okTxCount, failedTxCount, timeElapsed, tps, float64(h)/float64(allBlocks)*100)
			if sendRawTxWithRetry(c, tx, false, retryCount) {
				okTxCount++
			} else {
				failedTxCount++
			}
		}
	}
	fmt.Println("\nDONE!")
}

func getTotalHeight(blkDB *BlockDB) uint32 {
	h := uint32(1)
	for blkDB.LoadBlock(h) != nil {
		fmt.Printf("\rtotal blocks: %d", h)
		h += 100
	}
	h -= 100
	for blkDB.LoadBlock(h) != nil {
		fmt.Printf("\rtotal blocks: %d", h)
		h++
	}
	fmt.Println()
	return h
}

func sendRawTxWithRetry(c *websocket.Conn, tx []byte, logsMsg bool, retryCount int) bool {
	reqID++
	req := []byte(fmt.Sprintf(sendRawTxReqFmt, "0x"+hex.EncodeToString(tx), reqID))
	for i := 0; i < retryCount; i++ {
		//time.Sleep(100 * time.Millisecond)
		resp := sendReq(c, req, logsMsg)
		if !bytes.Contains(resp, []byte("error")) {
			return true
		}

		// retry
		if i < retryCount-1 {
			time.Sleep(50 * time.Millisecond)
		} else {
			fmt.Println("failed to send tx:", string(resp))
			return false
		}
	}
	return false
}

func sendReq(c *websocket.Conn, req []byte, logsMsg bool) []byte {
	if logsMsg {
		fmt.Println("write:", string(req))
	}

	err := c.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		if logsMsg {
			fmt.Println("write error:", err)
		}
		return []byte("error:" + err.Error())
	}

	_, resp, err := c.ReadMessage()
	if err != nil {
		if logsMsg {
			fmt.Println("read error:", err)
		}
		return []byte("error:" + err.Error())
	}
	if logsMsg {
		fmt.Println("read:", string(resp))
	}
	return resp
}

type RespObj struct {
	Result []TxReceipt `json:"result"`
}
type TxReceipt struct {
	TransactionHash string `json:"transactionHash"`
	Status          string `json:"status"`
	StatusStr       string `json:"statusStr"`
}

func RunQueryTxsWS(url string) {
	fmt.Println("connecting to ", url)

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	h := 0
	for {
		h++
		reqID++
		req := []byte(fmt.Sprintf(getTxListReqFmt, reqID, h))
		resp := sendReq(c, req, false)

		var respObj RespObj
		if err := json.Unmarshal(resp, &respObj); err != nil {
			fmt.Println(err.Error())
		} else {
			//println("ok")
		}

		fmt.Printf("height: %d, all tx: %d, failed tx: %d\n",
			h, len(respObj.Result), getFailedTxCount(respObj))
	}
}

func getFailedTxCount(resp RespObj) int {
	n := 0
	for _, tx := range resp.Result {
		//fmt.Println(tx.Status)
		if tx.Status != "0x1" {
			n++
		}
	}
	return n
}
