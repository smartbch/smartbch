package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	ReqStrSyncBlock = `{"jsonrpc": "2.0", "method": "sbch_getSyncBlock", "params": ["%s"], "id":1}`
	ReqStrBlockNum  = `{"jsonrpc": "2.0", "method": "eth_blockNumber", "params": [], "id":1}`
)

type IStateProducer interface {
	GeLatestBlockHeight() (int64, error)
	GetSyncBlock(height uint64) (*modbtypes.ExtendedBlock, error)
}

type RpcClient struct {
	url         string
	user        string
	password    string
	err         error
	contentType string
	logger      log.Logger
}

func NewRpcClient(url, user, password, contentType string, logger log.Logger) *RpcClient {
	if url == "" {
		url = "http://0.0.0.0:8545"
	}
	return &RpcClient{
		url:         url,
		user:        user,
		password:    password,
		contentType: contentType,
		logger:      logger,
	}
}

func (client *RpcClient) sendRequest(reqStr string) ([]byte, error) {
	body := strings.NewReader(reqStr)
	req, err := http.NewRequest("POST", client.url, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(client.user, client.password)
	req.Header.Set("Content-Type", client.contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	return respData, nil
}

type jsonrpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func (client *RpcClient) GetSyncBlock(height uint64) (*modbtypes.ExtendedBlock, error) {
	var respData []byte
	respData, client.err = client.sendRequest(fmt.Sprintf(ReqStrSyncBlock, hexutil.Uint64(height).String()))
	if client.err != nil {
		return nil, client.err
	}
	var m jsonrpcMessage
	client.err = json.Unmarshal(respData, &m)
	if client.err != nil {
		return nil, client.err
	}
	var eBlockString string
	client.err = json.Unmarshal(m.Result, &eBlockString)
	if client.err != nil {
		return nil, client.err
	}
	var eBlockBytes []byte
	eBlockBytes, client.err = hexutil.Decode(eBlockString)
	if client.err != nil {
		return nil, client.err
	}
	var eBlock modbtypes.ExtendedBlock
	_, client.err = eBlock.UnmarshalMsg(eBlockBytes)
	if client.err != nil {
		return nil, client.err
	}
	return &eBlock, nil
}

func (client *RpcClient) GeLatestBlockHeight() (int64, error) {
	var respData []byte
	respData, client.err = client.sendRequest(ReqStrBlockNum)
	if client.err != nil {
		return -1, client.err
	}
	var m jsonrpcMessage
	client.err = json.Unmarshal(respData, &m)
	if client.err != nil {
		return -1, client.err
	}
	var latestBlockHeight hexutil.Uint64
	client.err = json.Unmarshal(m.Result, &latestBlockHeight)
	if client.err != nil {
		return -1, client.err
	}
	return int64(latestBlockHeight), nil
}
