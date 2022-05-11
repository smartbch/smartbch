package watcher

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	stakingtypes "github.com/smartbch/smartbch/staking/types"
	"github.com/smartbch/smartbch/watcher/types"
)

const (
	ReqStrBlockCount = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getblockcount", "params": [] }`
	ReqStrBlockHash  = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getblockhash", "params": [%d] }`
	//verbose = 2, show all txs rawdata
	ReqStrBlock = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getblock", "params": ["%s",2] }`
	ReqStrTx    = `{"jsonrpc": "1.0", "id":"smartbch", "method": "getrawtransaction", "params": ["%s", true, "%s"] }`
)

type BchRpcClient struct {
	url         string
	user        string
	password    string
	err         error
	contentType string
	logger      log.Logger
}

var _ types.BchRpcClient = (*BchRpcClient)(nil)

func NewRpcClient(url, user, password, contentType string, logger log.Logger) *BchRpcClient {
	if url == "" {
		return nil
	}
	return &BchRpcClient{
		url:         url,
		user:        user,
		password:    password,
		contentType: contentType,
		logger:      logger,
	}
}

func (client *BchRpcClient) GetLatestHeight(retry bool) (height int64) {
	height = -1
	for height == -1 {
		height = client.getCurrHeight()
		if !retry {
			return height
		}
		if client.err != nil {
			client.logger.Debug("GetLatestHeight failed", client.err.Error())
			time.Sleep(10 * time.Second)
		}
	}
	return
}

func (client *BchRpcClient) GetBlockByHeight(height int64, retry bool) *types.BCHBlock {
	var hash string
	var err error
	var blk *types.BCHBlock
	for hash == "" {
		hash, err = client.getBlockHashOfHeight(height)
		if err != nil {
			if !retry {
				return nil
			}
			client.logger.Debug(fmt.Sprintf("getBlockHashOfHeight %d failed", height), err.Error())
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Printf("get bch block hash\n")
	}
	for blk == nil {
		blk, err = client.getBCHBlock(hash)
		if !retry {
			return blk
		}
		if err != nil {
			client.logger.Debug(fmt.Sprintf("getBCHBlock %d failed", height), err.Error())
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Printf("get bch block: %d\n", height)
	}
	return blk
}

func (client *BchRpcClient) sendRequest(reqStr string) ([]byte, error) {
	return sendRequest(client.url, client.user, client.password, client.contentType, reqStr)
}

func (client *BchRpcClient) getBCHBlock(hash string) (*types.BCHBlock, error) {
	var bi *types.BlockInfo
	var err error
	bi, err = client.getBlock(hash)
	if err != nil {
		return nil, err
	}
	bchBlock := &types.BCHBlock{
		Height:    bi.Height,
		Timestamp: bi.Time,
	}
	var bz []byte
	bz, err = hex.DecodeString(bi.Hash)
	copy(bchBlock.HashId[:], bz)
	if err != nil {
		return nil, err
	}
	bz, err = hex.DecodeString(bi.PreviousBlockhash)
	copy(bchBlock.ParentBlk[:], bz)
	if err != nil {
		return nil, err
	}
	if bi.Height > 0 {
		nomination := getNomination(bi.Tx[0])
		if nomination != nil {
			bchBlock.Nominations = append(bchBlock.Nominations, *nomination)
		}
		//bchBlock.CCTransferInfos = append(bchBlock.CCTransferInfos, client.getCCTransferInfos(bi)...)
	}
	return bchBlock, nil
}

func getNomination(coinbase types.TxInfo) *stakingtypes.Nomination {
	pubKey, ok := coinbase.GetValidatorPubKey()
	if ok {
		return &stakingtypes.Nomination{
			Pubkey:         pubKey,
			NominatedCount: 1,
		}
	}
	return nil
}

//func (client *BchRpcClient) getCCTransferInfos(bi *types.BlockInfo) []*cctypes.CCTransferInfo {
//	var ccInfos []*cctypes.CCTransferInfo
//	for _, info := range bi.Tx {
//		ccInfos = append(ccInfos, info.GetCCTransferInfos()...)
//	}
//	return ccInfos
//}

func (client *BchRpcClient) getCurrHeight() int64 {
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

func (client *BchRpcClient) getBlockHashOfHeight(height int64) (string, error) {
	var respData []byte
	var err error
	respData, err = client.sendRequest(fmt.Sprintf(ReqStrBlockHash, height))
	if err != nil {
		return "", err
	}
	var blockHashResp types.BlockHashResp
	err = json.Unmarshal(respData, &blockHashResp)
	if err != nil {
		return "", err
	}
	if blockHashResp.Error != nil && blockHashResp.Error.Code < 0 {
		err = fmt.Errorf("getBlockHashOfHeight error, height:%d, code:%d, msg:%s\n",
			height, blockHashResp.Error.Code, blockHashResp.Error.Message)
		return "", err
	}
	return blockHashResp.Result, nil
}

func (client *BchRpcClient) getBlock(hash string) (*types.BlockInfo, error) {
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

	// try to adapt BCHD
	if len(blockInfoResp.Result.Tx) == 0 &&
		len(blockInfoResp.Result.RawTx) > 0 {

		blockInfoResp.Result.Tx = blockInfoResp.Result.RawTx
		for i := 0; i < len(blockInfoResp.Result.Tx); i++ {
			blockInfoResp.Result.Tx[i].Hash = blockInfoResp.Result.Tx[i].TxID
		}
	}

	return &blockInfoResp.Result, nil
}

func (client *BchRpcClient) getTx(hash string, blockhash string) (*types.TxInfo, error) {
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrTx, hash, blockhash))
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

/***for tool*/
func (client *BchRpcClient) GetBlockHash(height int64) (string, error) {
	return client.getBlockHashOfHeight(height)
}
func (client *BchRpcClient) GetBlockInfo(hash string) (*types.BlockInfo, error) {
	return client.getBlock(hash)
}
func (client *BchRpcClient) GetTxInfo(hash string, blockhash string) (*types.TxInfo, error) {
	return client.getTx(hash, blockhash)
}
