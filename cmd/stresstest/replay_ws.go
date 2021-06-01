package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	sendRawTxReqFmt = `{"jsonrpc":"2.0", "method":"eth_sendRawTransaction", "params":["%s"], "id":%d}`
	getTxListReqFmt = `{"jsonrpc":"2.0", "method":"sbch_getTxListByHeight", "params":["0x%x"], "id":%d}`
	getBlkByNumFmt  = `{"jsonrpc":"2.0", "method":"eth_getBlockByNumber", "params":["0x%x", false], "id":%d}`
)

var reqID uint64

func RunReplayBlocksWS(url string, fromHeight, fromTx int) {
	fmt.Println("fromHeight:", fromHeight, "fromTx:", fromTx)
	fmt.Println("connecting to ", url)

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	blkDB := NewBlockDB(blockDir)
	allBlocks := getTotalHeight(blkDB)

	h := uint32(0)
	retryCount := 1000000
	okTxCount := 0
	failedTxCount := 0
	startTime := time.Now().Unix()
	limiter := time.Tick(3 * time.Millisecond)

blockLoop:
	for {
		h++
		if h < uint32(fromHeight) {
			continue blockLoop
		}

		blk := blkDB.LoadBlock(h)
		if blk == nil {
			break
		}

	txLoop:
		for i, tx := range blk.TxList {
			if h == uint32(fromHeight) && i < fromTx {
				continue txLoop
			}

			<-limiter
			tps := 0
			timeElapsed := time.Now().Unix() - startTime
			if timeElapsed > 0 {
				tps = okTxCount / int(timeElapsed)
			}
			fmt.Printf("\rblock: %d, tx: %d; total sent tx: %d, total failed tx: %d, time: %ds, tps: %d, progress: %f%%",
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
			time.Sleep(200 * time.Millisecond)
			fmt.Println("\nfailed to send tx:", string(resp))
		} else {
			fmt.Println("\nfailed to send tx:", string(resp))
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

type GetTxListRespObj struct {
	Result []TxReceipt `json:"result"`
}
type TxReceipt struct {
	TransactionHash string `json:"transactionHash"`
	GasUsed         string `json:"gasUsed"`
	Status          string `json:"status"`
	StatusStr       string `json:"statusStr"`
}

func RunQueryTxsWS(url string, maxHeight int) {
	fmt.Println("connecting to ", url)

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	for h := 1; h <= maxHeight; h++ {
		req := []byte(fmt.Sprintf(getTxListReqFmt, h, h))
		resp := sendReq(c, req, false)

		var respObj GetTxListRespObj
		if err := json.Unmarshal(resp, &respObj); err != nil {
			fmt.Println(err.Error())
		}

		failedTxCount := getFailedTxCount(respObj)
		totalGasUsed := sumGasUsed(respObj)
		fmt.Printf("height: %d, all tx: %d, failed tx: %d, total gas used: %d\n",
			h, len(respObj.Result), failedTxCount, totalGasUsed)
	}
}

func getFailedTxCount(resp GetTxListRespObj) int {
	n := 0
	for _, tx := range resp.Result {
		//fmt.Println(tx.Status)
		if tx.Status != "0x1" {
			n++
		}
	}
	return n
}

func sumGasUsed(resp GetTxListRespObj) uint64 {
	totalGasUsed := uint64(0)
	for _, tx := range resp.Result {
		//fmt.Println(tx.Status)
		if tx.Status == "0x1" {
			gasUsed := strings.TrimPrefix(tx.GasUsed, "0x")
			if n, err := strconv.ParseUint(gasUsed, 16, 32); err == nil {
				totalGasUsed += n
			}
		}
	}
	return totalGasUsed
}

type GetBlkByNumRespObj struct {
	Result BlockInfo `json:"result"`
}
type BlockInfo struct {
	Number       string   `json:"number"`
	Size         string   `json:"size"`
	GasUsed      string   `json:"gasUsed"`
	Timestamp    string   `json:"timestamp"`
	Miner        string   `json:"miner"`
	Transactions []string `json:"transactions"`
}

func RunQueryBlocksWS(url string, maxHeight int, minHeight int, genCharts bool) {
	fmt.Println("connecting to ", url)

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	lastT := uint64(0)
	var blocks []BlockInfo
	for h := minHeight; h <= maxHeight; h++ {
		req := []byte(fmt.Sprintf(getBlkByNumFmt, h, h))
		resp := sendReq(c, req, false)

		var respObj GetBlkByNumRespObj
		if err := json.Unmarshal(resp, &respObj); err != nil {
			fmt.Println(err.Error())
		}

		size, _ := strconv.ParseUint(strings.TrimPrefix(respObj.Result.Size, "0x"), 16, 32)
		gasUsed, _ := strconv.ParseUint(strings.TrimPrefix(respObj.Result.GasUsed, "0x"), 16, 32)
		t, _ := strconv.ParseUint(strings.TrimPrefix(respObj.Result.Timestamp, "0x"), 16, 32)
		if t == 0 {
			break
		}
		fmt.Printf("height: %d, time: %s-%d, all tx: %d, size: %fK, gas used: %fM miner: %s\n",
			h, respObj.Result.Timestamp, t-lastT, len(respObj.Result.Transactions),
			float64(size)/1024, float64(gasUsed)/1_000_000, respObj.Result.Miner)
		lastT = t

		if genCharts {
			blocks = append(blocks, respObj.Result)
		}
	}

	if genCharts {
		genChartsHTML(blocks)
	}
}

var chartsTmpl = []byte(`<html>
  <head>
    <script type="text/javascript" src="https://www.gstatic.com/charts/loader.js"></script>
    <script type="text/javascript">
      google.charts.load('current', {'packages':['corechart']});
      google.charts.setOnLoadCallback(drawChart);

      function drawChart() {
        var data = google.visualization.arrayToDataTable( {{ data_txCount }} );
        var options = {title: 'TxCount', curveType: 'function', legend: { position: 'bottom' }};
        var chart = new google.visualization.LineChart(document.getElementById('curve_txCount'));
        chart.draw(data, options);
      }
    </script>
	<script type="text/javascript">
      google.charts.load('current', {'packages':['corechart']});
      google.charts.setOnLoadCallback(drawChart);

      function drawChart() {
        var data = google.visualization.arrayToDataTable( {{ data_blockSize }} );
        var options = {title: 'BlockSize', curveType: 'function', legend: { position: 'bottom' }};
        var chart = new google.visualization.LineChart(document.getElementById('curve_blockSize'));
        chart.draw(data, options);
      }
    </script>
	<script type="text/javascript">
      google.charts.load('current', {'packages':['corechart']});
      google.charts.setOnLoadCallback(drawChart);

      function drawChart() {
        var data = google.visualization.arrayToDataTable( {{ data_gasUsed }} );
        var options = {title: 'GasUsed', curveType: 'function', legend: { position: 'bottom' }};
        var chart = new google.visualization.LineChart(document.getElementById('curve_gasUsed'));
        chart.draw(data, options);
      }
    </script>
	<script type="text/javascript">
      google.charts.load('current', {'packages':['corechart']});
      google.charts.setOnLoadCallback(drawChart);

      function drawChart() {
        var data = google.visualization.arrayToDataTable( {{ data_blockTime }} );
        var options = {title: 'BlockTime', curveType: 'function', legend: { position: 'bottom' }};
        var chart = new google.visualization.LineChart(document.getElementById('curve_blockTime'));
        chart.draw(data, options);
      }
    </script>
  </head>
  <body>
    <div id="curve_txCount" style="width: 900px; height: 500px"></div>
    <div id="curve_blockSize" style="width: 900px; height: 500px"></div>
    <div id="curve_gasUsed" style="width: 900px; height: 500px"></div>
    <div id="curve_blockTime" style="width: 900px; height: 500px"></div>
  </body>
</html>`)

func genChartsHTML(blocks []BlockInfo) {
	html := bytes.Replace(chartsTmpl, []byte("{{ data_txCount }}"), getTxCountData(blocks), 1)
	html = bytes.Replace(html, []byte("{{ data_blockSize }}"), getBlockSizeData(blocks), 1)
	html = bytes.Replace(html, []byte("{{ data_gasUsed }}"), getGasUsedData(blocks), 1)
	html = bytes.Replace(html, []byte("{{ data_blockTime }}"), getBlockTimeData(blocks), 1)
	_ = ioutil.WriteFile("./charts.html", html, 0644)
}

const blockBundleSize = 100

func getTxCountData(blocks []BlockInfo) []byte {
	var data [][2]interface{}
	data = append(data, [2]interface{}{"Block", "TxCount"})
	sum := 0
	for i, block := range blocks {
		sum += len(block.Transactions)
		if (i+1) % blockBundleSize == 0 {
			h, _ := strconv.ParseUint(strings.TrimPrefix(block.Number, "0x"), 16, 32)
			data = append(data, [2]interface{}{h, sum})
			sum = 0
		}
	}
	s, _ := json.Marshal(data)
	return s
}
func getBlockSizeData(blocks []BlockInfo) []byte {
	var data [][2]interface{}
	data = append(data, [2]interface{}{"Block", "BlockSize"})
	sum := 0
	for i, block := range blocks {
		size, _ := strconv.ParseUint(strings.TrimPrefix(block.Size, "0x"), 16, 32)
		sum += int(size)
		if (i+1) % blockBundleSize == 0 {
			h, _ := strconv.ParseUint(strings.TrimPrefix(block.Number, "0x"), 16, 32)
			data = append(data, [2]interface{}{h, sum})
			sum = 0
		}
	}
	s, _ := json.Marshal(data)
	return s
}
func getGasUsedData(blocks []BlockInfo) []byte {
	var data [][2]interface{}
	data = append(data, [2]interface{}{"Block", "GasUsed"})
	sum := 0
	for i, block := range blocks {
		gasUsed, _ := strconv.ParseUint(strings.TrimPrefix(block.GasUsed, "0x"), 16, 32)
		sum += int(gasUsed)
		if (i+1) % blockBundleSize == 0 {
			h, _ := strconv.ParseUint(strings.TrimPrefix(block.Number, "0x"), 16, 32)
			data = append(data, [2]interface{}{h, sum})
			sum = 0
		}
	}
	s, _ := json.Marshal(data)
	return s
}
func getBlockTimeData(blocks []BlockInfo) []byte {
	var data [][2]interface{}
	data = append(data, [2]interface{}{"Block", "BlockTime"})
	lastTime := uint64(0)
	for i, block := range blocks {
		if i % blockBundleSize == 1 {
			h, _ := strconv.ParseUint(strings.TrimPrefix(block.Number, "0x"), 16, 32)
			t, _ := strconv.ParseUint(strings.TrimPrefix(block.Timestamp, "0x"), 16, 32)
			if lastTime > 0 {
				data = append(data, [2]interface{}{h, t - lastTime})
			}
			lastTime = t
		}
	}
	s, _ := json.Marshal(data)
	return s
}
