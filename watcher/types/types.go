package types

import (
	"encoding/hex"
	"strings"

	cctypes "github.com/smartbch/smartbch/crosschain/types"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

const (
	Identifier = "73424348" // ascii code for 'sBCH'
	Validator  = "00"
	Monitor    = "01"

	ShaGateAddress = "14f8c7e99fd4e867c34cbd5968e35575fd5919a4"
)

// These functions must be provided by a client connecting to a Bitcoin Cash's fullnode
type RpcClient interface {
	GetLatestHeight(retry bool) int64
	GetBlockByHeight(height int64, retry bool) *BCHBlock
	GetVoteInfoByEpochNumber(start, end uint64) []*VoteInfo
	GetBlockInfoByHeight(height int64, retry bool) *BlockInfo
}

type VoteInfo struct {
	Epoch       stakingtypes.Epoch
	MonitorVote cctypes.MonitorVoteInfo
}

// This struct contains the useful information of a BCH block
type BCHBlock struct {
	Height        int64
	Timestamp     int64
	HashId        [32]byte
	ParentBlk     [32]byte
	CCNominations []cctypes.Nomination
	Nominations   []stakingtypes.Nomination
}

//not check Nominations
func (b *BCHBlock) Equal(o *BCHBlock) bool {
	return b.Height == o.Height && b.Timestamp == o.Timestamp &&
		b.HashId == o.HashId && b.ParentBlk == o.ParentBlk
}

/***mainnet data structure*/
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
	Tx                []TxInfo `json:"tx"`
	RawTx             []TxInfo `json:"rawtx"` // BCHD
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
		prefix := "OP_RETURN " + Identifier + Validator
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

func (ti TxInfo) GetMonitorPubKey() (pubKey [33]byte, success bool) {
	for _, vout := range ti.VoutList {
		asm, ok := vout.ScriptPubKey["asm"]
		if !ok || asm == nil {
			continue
		}
		script, ok := asm.(string)
		if !ok {
			continue
		}
		prefix := "OP_RETURN " + Identifier + Monitor
		if !strings.HasPrefix(script, prefix) {
			continue
		}
		script = script[len(prefix):]
		if len(script) != 66 {
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
