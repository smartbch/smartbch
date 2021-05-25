package main

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

const (
	sendRawTxReqFmt = `{"jsonrpc":"2.0", "method":"eth_sendRawTransaction", "params":["%s"], "id":%d}`
)

func RunReplayBlocksWS() {
	addr := "localhost:8546"
	u := url.URL{Scheme: "ws", Host: addr, Path: "/"}
	fmt.Println("connecting to ", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	blkDB := NewBlockDB(blockDir)
	h := uint32(0)
	for {
		h++
		blk := blkDB.LoadBlock(h)
		if blk == nil {
			break
		}

		for _, tx := range blk.TxList {
			//fmt.Printf("\rblock: %9d, tx: %3d", h, i)
			sendRawTxWS(tx, c, true)
			time.Sleep(100 * time.Millisecond)
		}
	}
	fmt.Println("\nDONE!")
}

func sendRawTxWS(tx []byte, c *websocket.Conn, printsLog bool) {
	sendRawTxReq := fmt.Sprintf(sendRawTxReqFmt, "0x"+hex.EncodeToString(tx), 123)
	if printsLog {
		fmt.Println("write:", sendRawTxReq)
	}

	err := c.WriteMessage(websocket.TextMessage, []byte(sendRawTxReq))
	if err != nil {
		if printsLog {
			fmt.Println("write error:", err)
		}
		return
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		if printsLog {
			fmt.Println("read error:", err)
		}
		return
	}
	if printsLog {
		fmt.Println("read:", string(message))
	}
}
