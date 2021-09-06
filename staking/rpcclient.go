package staking

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/tendermint/tendermint/libs/log"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/smartbch/smartbch/staking/types"
)

const (
	ReqStrBlockCount = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getblockcount", "params": [] }`
	ReqStrBlockHash  = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getblockhash", "params": [%d] }`
	ReqStrBlock      = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getblock", "params": ["%s"] }`
	ReqStrTx         = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getrawtransaction", "params": ["%s", true, "%s"] }`
	ReqStrEpochs     = `{"jsonrpc": "2.0", "method": "sbch_getEpochs", "params": ["%s","%s"], "id":1}`
	Identifier       = "73424348" // ascii code for 'sBCH'
	Version          = "00"
)

type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type BlockCountResp struct {
	Result int64         `json:"result"`
	Error  *JsonRpcError `json:"error"`
	Id     string        `json:"id"`
}

type BlockHashResp struct {
	Result string        `json:"result"`
	Error  *JsonRpcError `json:"error"`
	Id     string        `json:"id"`
}

type BlockInfo struct {
	Hash              string   `json:"hash"`
	Confirmations     int      `json:"confirmations"`
	Size              int      `json:"size"`
	Height            int64    `json:"height"`
	Version           int      `json:"version"`
	VersionHex        string   `json:"versionHex"`
	Merkleroot        string   `json:"merkleroot"`
	Tx                []string `json:"tx"`
	Time              int64    `json:"time"`
	MedianTime        int64    `json:"mediantime"`
	Nonce             int      `json:"nonce"`
	Bits              string   `json:"bits"`
	Difficulty        float64  `json:"difficulty"`
	Chainwork         string   `json:"chainwork"`
	NumTx             int      `json:"nTx"`
	PreviousBlockhash string   `json:"previousblockhash"`
}

type BlockInfoResp struct {
	Result BlockInfo     `json:"result"`
	Error  *JsonRpcError `json:"error"`
	Id     string        `json:"id"`
}

type CoinbaseVin struct {
	Coinbase string `json:"coinbase"`
	Sequence int    `json:"sequence"`
}

type Vout struct {
	Value        float64                `json:"value"`
	N            int                    `json:"n"`
	ScriptPubKey map[string]interface{} `json:"scriptPubKey"`
}

type TxInfo struct {
	TxID          string                   `json:"txid"`
	Hash          string                   `json:"hash"`
	Version       int                      `json:"version"`
	Size          int                      `json:"size"`
	Locktime      int                      `json:"locktime"`
	VinList       []map[string]interface{} `json:"vin"`
	VoutList      []Vout                   `json:"vout"`
	Hex           string                   `json:"hex"`
	Blockhash     string                   `json:"blockhash"`
	Confirmations int                      `json:"confirmations"`
	Time          int64                    `json:"time"`
	BlockTime     int64                    `json:"blocktime"`
}

func (ti TxInfo) GetValidatorPubKey() (pubKey [32]byte, success bool) {
	for _, vout := range ti.VoutList {
		asm, ok := vout.ScriptPubKey["asm"]
		if !ok || asm == nil {
			continue
		}
		script, ok := asm.(string)
		if !ok {
			continue
		}
		prefix := "OP_RETURN " + Identifier + Version
		if !strings.HasPrefix(script, prefix) {
			continue
		}
		script = script[len(prefix):]
		if len(script) != 64 {
			continue
		}
		bz, err := hex.DecodeString(script)
		if err != nil {
			continue
		}
		copy(pubKey[:], bz)
		success = true
		break
	}
	return
}

type TxInfoResp struct {
	Result TxInfo        `json:"result"`
	Error  *JsonRpcError `json:"error"`
	Id     string        `json:"id"`
}

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

func (client *RpcClient) GetLatestHeight() (height int64) {
	height, client.err = client.getCurrHeight()
	if client.err != nil {
		client.logger.Debug("GetLatestHeight failed", client.err.Error())
	}
	return
}

func (client *RpcClient) GetBlockHash(height int64) (string, error) {
	return client.getBlockHashOfHeight(height)
}
func (client *RpcClient) GetBlockInfo(hash string) (*BlockInfo, error) {
	return client.getBlock(hash)
}
func (client *RpcClient) GetTxInfo(hash string, blockhash string) (*TxInfo, error) {
	return client.getTx(hash, blockhash)
}

func (client *RpcClient) GetBlockByHeight(height int64) *types.BCHBlock {
	var hash string
	hash, client.err = client.getBlockHashOfHeight(height)
	if client.err != nil {
		client.logger.Debug("getBlockHashOfHeight failed", client.err.Error())
		return nil
	}
	blk := client.getBCHBlock(hash)
	if client.err != nil {
		client.logger.Debug("getBCHBlock failed", client.err.Error())
	}
	return blk
}

func (client *RpcClient) GetBlockByHash(hash [32]byte) *types.BCHBlock {
	blk := client.getBCHBlock(hex.EncodeToString(hash[:]))
	if client.err != nil {
		client.logger.Debug("GetBlockByHash failed", client.err.Error())
	}
	return blk
}

func (client *RpcClient) GetEpochs(start, end uint64) []*types.Epoch {
	epochs := client.getEpochs(start, end)
	if client.err != nil {
		client.logger.Debug("GetEpochs failed", client.err.Error())
	}
	return epochs
}

func (client *RpcClient) getBCHBlock(hash string) *types.BCHBlock {
	var bi *BlockInfo
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
		var coinbase *TxInfo
		coinbase, client.err = client.getTx(bi.Tx[0], bi.Hash)
		if client.err != nil {
			return nil
		}
		pubKey, ok := coinbase.GetValidatorPubKey()
		if ok {
			nomination := types.Nomination{
				Pubkey:         pubKey,
				NominatedCount: 1,
			}
			bchBlock.Nominations = append(bchBlock.Nominations, nomination)
		}
	}
	return bchBlock
}

func (client *RpcClient) getCurrHeight() (int64, error) {
	respData, err := client.sendRequest(ReqStrBlockCount)
	if err != nil {
		return -1, err
	}
	var blockCountResp BlockCountResp
	err = json.Unmarshal(respData, &blockCountResp)
	if err != nil {
		return -1, err
	}
	if blockCountResp.Error != nil && blockCountResp.Error.Code < 0 {
		return blockCountResp.Result, fmt.Errorf("getCurrHeight error, code:%d, msg:%s\n",
			blockCountResp.Error.Code, blockCountResp.Error.Message)
	}
	return blockCountResp.Result, nil
}

func (client *RpcClient) getBlockHashOfHeight(height int64) (string, error) {
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrBlockHash, height))
	if err != nil {
		return "", err
	}
	var blockHashResp BlockHashResp
	err = json.Unmarshal(respData, &blockHashResp)
	if err != nil {
		return "", err
	}
	if blockHashResp.Error != nil && blockHashResp.Error.Code < 0 {
		return blockHashResp.Result, fmt.Errorf("getBlockHashOfHeight error, height:%d, code:%d, msg:%s\n",
			height, blockHashResp.Error.Code, blockHashResp.Error.Message)
	}
	return blockHashResp.Result, nil
}

func (client *RpcClient) getBlock(hash string) (*BlockInfo, error) {
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrBlock, hash))
	if err != nil {
		return nil, err
	}
	var blockInfoResp BlockInfoResp
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

func (client *RpcClient) getTx(hash string, blockhash string) (*TxInfo, error) {
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrTx, hash, blockhash))
	if err != nil {
		return nil, err
	}
	var txInfoResp TxInfoResp
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

func (client *RpcClient) getEpochs(start, end uint64) []*types.Epoch {
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
	var epochsResp []*types.Epoch
	client.err = json.Unmarshal(m.Result, &epochsResp)
	if client.err != nil {
		return nil
	}
	return epochsResp
}
