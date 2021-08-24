package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	rpcjson "github.com/gorilla/rpc/json"
	watcher "github.com/smartbch/smartbch/watcher/types"
)

type BlockInfoRespList []*watcher.BlockInfoResp
type TxInfoRespList []*watcher.TxInfoResp

var Blockcount int64
var BlockHeight2Hash map[int64]string
var BlockHash2Content map[string]*watcher.BlockInfoResp
var TxHash2Content map[string]*watcher.TxInfoResp

type BlockcountService struct{}

func (_ *BlockcountService) Call(r *http.Request, args *string, result *int64) error {
	*result = Blockcount
	return nil
}

type BlockhashService struct{}

func (_ *BlockhashService) Call(r *http.Request, args *int64, result *string) error {
	var ok bool
	*result, ok = BlockHeight2Hash[*args]
	if !ok {
		return errors.New("No such height")
	}
	return nil
}

type BlockService struct{}

func (_ *BlockService) Call(r *http.Request, args *string, result *watcher.BlockInfo) error {
	info, ok := BlockHash2Content[*args]
	if !ok {
		return errors.New("No such block hash")
	}
	*result = info.Result
	return nil
}

type TxService struct{}

func (_ *TxService) Call(r *http.Request, args *string, result *watcher.TxInfo) error {
	info, ok := TxHash2Content[*args]
	if !ok {
		return errors.New("No such tx hash")
	}
	*result = info.Result
	return nil
}

func readBytes(fname string) []byte {
	jsonFile, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}
	return byteValue
}

func readBlockInfoList() {
	byteValue := readBytes("block.json")
	var biList BlockInfoRespList
	err := json.Unmarshal(byteValue, &biList)
	if err != nil {
		panic(err)
	}
	for _, bi := range biList {
		if Blockcount < bi.Result.Height {
			Blockcount = bi.Result.Height
		}
		BlockHeight2Hash[bi.Result.Height] = bi.Result.Hash
		BlockHash2Content[bi.Result.Hash] = bi
	}
}

func readTxInfoList() {
	byteValue := readBytes("tx.json")
	var txList TxInfoRespList
	err := json.Unmarshal(byteValue, &txList)
	if err != nil {
		panic(err)
	}
	for _, tx := range txList {
		TxHash2Content[tx.Result.Hash] = tx
	}
}

func main() {
	BlockHeight2Hash = make(map[int64]string)
	BlockHash2Content = make(map[string]*watcher.BlockInfoResp)
	TxHash2Content = make(map[string]*watcher.TxInfoResp)
	readBlockInfoList()
	readTxInfoList()
	fmt.Println("Load finished")
	s := rpc.NewServer()
	s.RegisterCodec(rpcjson.NewCodec(), "text/plain")
	_ = s.RegisterService(new(BlockcountService), "getblockcount")
	_ = s.RegisterService(new(BlockhashService), "getblockhash")
	_ = s.RegisterService(new(BlockService), "getblock")
	_ = s.RegisterService(new(TxService), "getrawtransaction")
	r := mux.NewRouter()
	r.Handle("/", s)
	_ = http.ListenAndServe(":1234", r)
}
