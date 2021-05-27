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

	allBlocks := uint32(1)
	for blkDB.LoadBlock(allBlocks) != nil {
		fmt.Printf("\rtotal blocks: %d", allBlocks)
		allBlocks += 100
	}
	allBlocks -= 100
	for blkDB.LoadBlock(allBlocks) != nil {
		fmt.Printf("\rtotal blocks: %d", allBlocks)
		allBlocks++
	}
	fmt.Println()

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
			for i := 0; i < retryCount; i++ {
				//time.Sleep(100 * time.Millisecond)
				resp := sendRawTxWS(tx, c, false)
				if !bytes.Contains(resp, []byte("error")) {
					okTxCount++
					break // ok
				}

				// retry
				if i < retryCount-1 {
					time.Sleep(50 * time.Millisecond)
				} else {
					fmt.Println("failed to send tx:", string(resp))
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
