package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const (
	sendRawTxReqFmt = `{"jsonrpc":"2.0", "method":"eth_sendRawTransaction", "params":["%s"], "id":%d}`
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
	h := uint32(0)
	retryCount := 10
	okTxCount := 0
	failedTxCount := 0
	startTime := time.Now().UnixNano()
	for {
		h++
		blk := blkDB.LoadBlock(h)
		if blk == nil {
			break
		}

		for i, tx := range blk.TxList {
			now := time.Now().UnixNano()
			fmt.Printf("\rblock: %d, tx: %d; total sent tx: %d, total failed tx: %d, tps:%f",
				h, i, okTxCount, failedTxCount, float64(okTxCount)/(float64(now-startTime)/10e9))
			for i := 0; i < retryCount; i++ {
				//time.Sleep(100 * time.Millisecond)
				resp := sendRawTxWS(tx, c, false)
				if !bytes.Contains(resp, []byte("error")) {
					okTxCount++
					break // ok
				}

				// retry
				if i < retryCount-1 {
					time.Sleep(10 * time.Millisecond)
				} else {
					//fmt.Println("failed to send tx:", string(resp))
					failedTxCount++
				}
			}
		}
	}
	fmt.Println("\nDONE!")
}

func sendRawTxWS(tx []byte, c *websocket.Conn, printsLog bool) []byte {
	reqID++
	sendRawTxReq := fmt.Sprintf(sendRawTxReqFmt, "0x"+hex.EncodeToString(tx), reqID)
	if printsLog {
		fmt.Println("write:", sendRawTxReq)
	}

	err := c.WriteMessage(websocket.TextMessage, []byte(sendRawTxReq))
	if err != nil {
		if printsLog {
			fmt.Println("write error:", err)
		}
		return []byte("error:" + err.Error())
	}

	_, resp, err := c.ReadMessage()
	if err != nil {
		if printsLog {
			fmt.Println("read error:", err)
		}
		return []byte("error:" + err.Error())
	}
	if printsLog {
		fmt.Println("read:", string(resp))
	}
	return resp
}
