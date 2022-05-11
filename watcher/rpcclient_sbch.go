package watcher

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/tendermint/tendermint/libs/log"

	cctypes "github.com/smartbch/smartbch/crosschain/types"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
	"github.com/smartbch/smartbch/watcher/types"
)

const (
	ReqStrEpochs   = `{"jsonrpc": "2.0", "method": "sbch_getEpochs", "params": ["%s","%s"], "id":1}`
	ReqStrCCEpochs = `{"jsonrpc": "2.0", "method": "sbch_getCCEpochs", "params": ["%s","%s"], "id":1}`
)

type SbchRpcClient struct {
	url         string
	user        string
	password    string
	err         error
	contentType string
	logger      log.Logger
}

type smartBchJsonrpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type smartBchJsonrpcMessage struct {
	Version string                `json:"jsonrpc,omitempty"`
	ID      json.RawMessage       `json:"id,omitempty"`
	Method  string                `json:"method,omitempty"`
	Params  json.RawMessage       `json:"params,omitempty"`
	Error   *smartBchJsonrpcError `json:"error,omitempty"`
	Result  json.RawMessage       `json:"result,omitempty"`
}

var _ types.SbchRpcClient = (*SbchRpcClient)(nil)

func NewSbchRpcClient(url, user, password, contentType string, logger log.Logger) *SbchRpcClient {
	if url == "" {
		return nil
	}
	return &SbchRpcClient{
		url:         url,
		user:        user,
		password:    password,
		contentType: contentType,
		logger:      logger,
	}
}

func (client *SbchRpcClient) GetEpochs(start, end uint64) []*stakingtypes.Epoch {
	var epochs []*stakingtypes.Epoch
	for epochs == nil {
		epochs = client.getEpochs(start, end)
		if client.err != nil {
			client.logger.Debug("GetEpochs failed", client.err.Error())
			time.Sleep(10 * time.Second)
		}
	}
	return epochs
}

func (client *SbchRpcClient) getEpochs(start, end uint64) []*stakingtypes.Epoch {
	var respData []byte
	respData, client.err = client.sendRequest(fmt.Sprintf(ReqStrEpochs, hexutil.Uint64(start).String(), hexutil.Uint64(end).String()))
	if client.err != nil {
		return nil
	}
	var m smartBchJsonrpcMessage
	client.err = json.Unmarshal(respData, &m)
	if client.err != nil {
		return nil
	}
	var epochsResp []*stakingtypes.Epoch
	client.err = json.Unmarshal(m.Result, &epochsResp)
	if client.err != nil {
		return nil
	}
	return epochsResp
}

func (client *SbchRpcClient) GetCCEpochs(start, end uint64) []*cctypes.CCEpoch {
	var respData []byte
	respData, client.err = client.sendRequest(fmt.Sprintf(ReqStrCCEpochs, hexutil.Uint64(start).String(), hexutil.Uint64(end).String()))
	if client.err != nil {
		return nil
	}
	var m smartBchJsonrpcMessage
	client.err = json.Unmarshal(respData, &m)
	if client.err != nil {
		return nil
	}
	var epochsResp []*cctypes.CCEpoch
	client.err = json.Unmarshal(m.Result, &epochsResp)
	if client.err != nil {
		return nil
	}
	return epochsResp
}

func (client *SbchRpcClient) sendRequest(reqStr string) ([]byte, error) {
	return sendRequest(client.url, client.user, client.password, client.contentType, reqStr)
}
