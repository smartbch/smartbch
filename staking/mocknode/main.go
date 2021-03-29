package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	rpcjson "github.com/gorilla/rpc/json"

	"github.com/smartbch/smartbch/staking"
)

type BlockInfoList []*staking.BlockInfo
type TxInfoList []*staking.TxInfo

var Blockcount int64
var BlockHeight2Hash map[int64]string
var BlockHash2Content map[string]*staking.BlockInfo
var TxHash2Content map[string]*staking.TxInfo

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

func (_ *BlockService) Call(r *http.Request, args *string, result *staking.BlockInfo) error {
	info, ok := BlockHash2Content[*args]
	if !ok {
		return errors.New("No such block hash")
	}
	*result = *info
	return nil
}

type TxService struct{}

func (_ *TxService) Call(r *http.Request, args *string, result *staking.TxInfo) error {
	info, ok := TxHash2Content[*args]
	if !ok {
		return errors.New("No such tx hash")
	}
	*result = *info
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
	var biList BlockInfoList
	err := json.Unmarshal(byteValue, &biList)
	if err != nil {
		panic(err)
	}
	for _, bi := range biList {
		if Blockcount < bi.Height {
			Blockcount = bi.Height
		}
		BlockHeight2Hash[bi.Height] = bi.Hash
		BlockHash2Content[bi.Hash] = bi
	}
}

func readTxInfoList() {
	byteValue := readBytes("tx.json")
	var txList TxInfoList
	err := json.Unmarshal(byteValue, &txList)
	if err != nil {
		panic(err)
	}
	for _, tx := range txList {
		TxHash2Content[tx.Hash] = tx
	}
}

func main() {
	s := rpc.NewServer()
	s.RegisterCodec(rpcjson.NewCodec(), "text/plain")
	s.RegisterService(new(BlockcountService), "getblockcount")
	s.RegisterService(new(BlockhashService), "getblockhash")
	s.RegisterService(new(BlockService), "getblock")
	s.RegisterService(new(TxService), "getrawtransaction")
	r := mux.NewRouter()
	r.Handle("/", s)
	http.ListenAndServe(":1234", r)
}
