package staking

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	ReqStrTx         = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getrawtransaction", "params": ["%s", true] }`
	ReqStrEpochs     = `{"jsonrpc": "2.0", "method": "sbch_getEpochs", "params": ["%s","%s"], "id":1}`
	Identifier       = "73424348"
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
}

var _ types.RpcClient = (*RpcClient)(nil)

func NewRpcClient(url, user, password, contentType string) *RpcClient {
	fmt.Println("watcher rpc url:", url)
	fmt.Println("watcher rpc user:", user)
	fmt.Println("watcher rpc password:", password)

	return &RpcClient{
		url:         url,
		user:        user,
		password:    password,
		contentType: contentType,
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
	return
}

func (client *RpcClient) GetBlockByHeight(height int64) *types.BCHBlock {
	var hash string
	hash, client.err = client.getBlockHashOfHeight(height)
	if client.err != nil {
		fmt.Println("getblockByHeight error:", client.err)
		return nil
	}
	return client.getBCHBlock(hash)
}

func (client *RpcClient) GetBlockByHash(hash [32]byte) *types.BCHBlock {
	return client.getBCHBlock(hex.EncodeToString(hash[:]))
}

func (client *RpcClient) GetEpochs(start, end uint64) []*types.Epoch {
	return client.getEpochs(start, end)
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
		coinbase, client.err = client.getTx(bi.Tx[0])
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
		return blockHashResp.Result, fmt.Errorf("getBlockHashOfHeight error, code:%d, msg:%s\n",
			blockHashResp.Error.Code, blockHashResp.Error.Message)
	}
	return blockHashResp.Result, nil
}

func (client *RpcClient) getBlock(hash string) (*BlockInfo, error) {
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrBlock, hash))
	if err != nil {
		return nil, err
	}
	//fmt.Printf("BLOCK %s\n", string(respData))
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

func (client *RpcClient) getTx(hash string) (*TxInfo, error) {
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrTx, hash))
	if err != nil {
		return nil, err
	}
	//fmt.Printf("TX %s\n", string(respData))
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
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrEpochs, hexutil.Uint64(start).String(), hexutil.Uint64(end).String()))
	if err != nil {
		fmt.Printf("get epoch error, %s\n", err.Error())
		return nil
	}
	fmt.Println(string(respData))
	var m smartBchJsonrpcMessage
	err = json.Unmarshal(respData, &m)
	if err != nil {
		fmt.Printf("get epoch rpc result error, %s\n", err.Error())
		return nil
	}
	var epochsResp []*types.Epoch
	err = json.Unmarshal(m.Result, &epochsResp)
	if err != nil {
		fmt.Printf("get epoch error, %s\n", err.Error())
		return nil
	}
	fmt.Println(epochsResp)
	return epochsResp
}

func (client *RpcClient) PrintAllOpReturn(startHeight, endHeight int64) {
	for h := startHeight; h < endHeight; h++ {
		fmt.Printf("Height: %d\n", h)
		hash, err := client.getBlockHashOfHeight(h)
		if err != nil {
			fmt.Printf("Error when getBlockHashOfHeight %d %s\n", h, err.Error())
			continue
		}
		bi, err := client.getBlock(hash)
		if err != nil {
			fmt.Printf("Error when getBlock %d %s\n", h, err.Error())
			continue
		}
		found := false
		for _, txid := range bi.Tx {
			tx, err := client.getTx(txid)
			if err != nil {
				fmt.Printf("Error when getTx %s %s\n", txid, err.Error())
				continue
			}
			for _, vout := range tx.VoutList {
				asm, ok := vout.ScriptPubKey["asm"]
				if !ok || asm == nil {
					continue
				}
				script, ok := asm.(string)
				if !ok {
					continue
				}
				if strings.HasPrefix(script, "OP_RETURN") {
					found = true
					fmt.Println(script)
				}
			}
		}
		if !found {
			fmt.Println("OP_RETURN not found!")
		}
	}
}
