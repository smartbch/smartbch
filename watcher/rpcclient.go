package watcher

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/tendermint/tendermint/libs/log"

	cctypes "github.com/smartbch/smartbch/crosschain/types"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
	"github.com/smartbch/smartbch/watcher/types"
)

const (
	ReqStrBlockCount = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getblockcount", "params": [] }`
	ReqStrBlockHash  = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getblockhash", "params": [%d] }`
	ReqStrBlock      = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getblock", "params": ["%s"] }`
	ReqStrTx         = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getrawtransaction", "params": ["%s", true] }`
	ReqStrEpochs     = `{"jsonrpc": "2.0", "method": "sbch_getEpochs", "params": ["%s","%s"], "id":1}`
)

type RpcClient struct {
	url         string
	user        string
	password    string
	err         error
	contentType string
	logger      log.Logger
}

var _ types.RpcClient = (*RpcClient)(nil)

func NewRpcClient(url, user, password, contentType string, logger log.Logger) *RpcClient {
	if url == "" {
		return nil
	}
	return &RpcClient{
		url:         url,
		user:        user,
		password:    password,
		contentType: contentType,
		logger:      logger,
	}
}

func (client *RpcClient) GetLatestHeight() (height int64) {
	height = -1
	for height == -1 {
		height = client.getCurrHeight()
		if client.err != nil {
			client.logger.Debug("GetLatestHeight failed", client.err.Error())
			time.Sleep(10 * time.Second)
		}
	}
	return
}

func (client *RpcClient) GetBlockByHeight(height int64) *types.BCHBlock {
	var hash string
	var blk *types.BCHBlock
	for hash == "" {
		hash = client.getBlockHashOfHeight(height)
		if client.err != nil {
			client.logger.Debug("getBlockHashOfHeight failed", client.err.Error())
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Printf("get bch block hash\n")
	}
	for blk == nil {

		blk = client.getBCHBlock(hash)
		if client.err != nil {
			client.logger.Debug("getBCHBlock failed", client.err.Error())
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Printf("get bch block: %d\n", height)
	}
	return blk
}

func (client *RpcClient) GetEpochs(start, end uint64) []*stakingtypes.Epoch {
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

func (client *RpcClient) getBCHBlock(hash string) *types.BCHBlock {
	var bi *types.BlockInfo
	bi, client.err = client.getBlock(hash)
	if client.err != nil {
		return nil
	}
	bchBlock := &types.BCHBlock{
		Height:    bi.Height,
		Timestamp: bi.Time,
	}
	var bz []byte
	bz, client.err = hex.DecodeString(bi.Hash)
	copy(bchBlock.HashId[:], bz)
	if client.err != nil {
		return nil
	}
	bz, client.err = hex.DecodeString(bi.PreviousBlockhash)
	copy(bchBlock.ParentBlk[:], bz)
	if client.err != nil {
		return nil
	}
	if bi.Height > 0 {
		nomination := client.getNomination(bi.Tx[0])
		if nomination != nil {
			bchBlock.Nominations = append(bchBlock.Nominations, *nomination)
		}
		bchBlock.CCTransferInfos = append(bchBlock.CCTransferInfos, client.getCCTransferInfos(bi)...)
	}
	if client.err != nil {
		return nil
	}
	return bchBlock
}

func (client *RpcClient) getNomination(txHash string) *stakingtypes.Nomination {
	var coinbase *types.TxInfo
	coinbase, client.err = client.getTx(txHash)
	if client.err != nil {
		return nil
	}
	pubKey, ok := coinbase.GetValidatorPubKey()
	if ok {
		return &stakingtypes.Nomination{
			Pubkey:         pubKey,
			NominatedCount: 1,
		}
	}
	return nil
}

func (client *RpcClient) getCCTransferInfos(bi *types.BlockInfo) []*cctypes.CCTransferInfo {
	var info *types.TxInfo
	var ccInfos []*cctypes.CCTransferInfo
	for _, txHash := range bi.Tx {
		info, client.err = client.getTx(txHash)
		if client.err != nil {
			return nil
		}
		ccInfos = append(ccInfos, info.GetCCTransferInfos()...)
	}
	return ccInfos
}

func (client *RpcClient) getCurrHeight() int64 {
	var respData []byte
	respData, client.err = client.sendRequest(ReqStrBlockCount)
	if client.err != nil {
		return -1
	}
	var blockCountResp types.BlockCountResp
	client.err = json.Unmarshal(respData, &blockCountResp)
	if client.err != nil {
		return -1
	}
	if blockCountResp.Error != nil && blockCountResp.Error.Code < 0 {
		client.err = fmt.Errorf("getCurrHeight error, code:%d, msg:%s\n", blockCountResp.Error.Code, blockCountResp.Error.Message)
		return blockCountResp.Result
	}
	return blockCountResp.Result
}

func (client *RpcClient) getBlockHashOfHeight(height int64) string {
	var respData []byte
	respData, client.err = client.sendRequest(fmt.Sprintf(ReqStrBlockHash, height))
	if client.err != nil {
		return ""
	}
	var blockHashResp types.BlockHashResp
	client.err = json.Unmarshal(respData, &blockHashResp)
	if client.err != nil {
		return ""
	}
	if blockHashResp.Error != nil && blockHashResp.Error.Code < 0 {
		client.err = fmt.Errorf("getBlockHashOfHeight error, height:%d, code:%d, msg:%s\n",
			height, blockHashResp.Error.Code, blockHashResp.Error.Message)
		return ""
	}
	return blockHashResp.Result
}

func (client *RpcClient) getBlock(hash string) (*types.BlockInfo, error) {
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrBlock, hash))
	if err != nil {
		return nil, err
	}
	var blockInfoResp types.BlockInfoResp
	err = json.Unmarshal(respData, &blockInfoResp)
	if err != nil {
		return nil, err
	}
	if blockInfoResp.Error != nil && blockInfoResp.Error.Code < 0 {
		return &blockInfoResp.Result, fmt.Errorf("getBlock error, code:%d, msg:%s\n",
			blockInfoResp.Error.Code, blockInfoResp.Error.Message)
	}
	return &blockInfoResp.Result, nil
}

func (client *RpcClient) getTx(hash string) (*types.TxInfo, error) {
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrTx, hash))
	if err != nil {
		return nil, err
	}
	var txInfoResp types.TxInfoResp
	err = json.Unmarshal(respData, &txInfoResp)
	if err != nil {
		return nil, err
	}
	if txInfoResp.Error != nil && txInfoResp.Error.Code < 0 {
		return &txInfoResp.Result, fmt.Errorf("getTx error, code:%d, msg:%s\n",
			txInfoResp.Error.Code, txInfoResp.Error.Message)
	}
	return &txInfoResp.Result, nil
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

func (client *RpcClient) getEpochs(start, end uint64) []*stakingtypes.Epoch {
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

/***for tool*/
func (client *RpcClient) GetBlockHash(height int64) (string, error) {
	return client.getBlockHashOfHeight(height), client.err
}
func (client *RpcClient) GetBlockInfo(hash string) (*types.BlockInfo, error) {
	return client.getBlock(hash)
}
func (client *RpcClient) GetTxInfo(hash string) (*types.TxInfo, error) {
	return client.getTx(hash)
}
