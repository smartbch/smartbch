package api

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/node"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/smartbch/smartbch/app"
)

/*-----------------------tendermint info----------------------------*/

type NextBlock struct {
	Number    int64       `json:"number"`
	Timestamp int64       `json:"timestamp"`
	Hash      common.Hash `json:"hash"`
}

type Info struct {
	IsValidator     bool            `json:"is_validator"`
	ValidatorIndex  int64           `json:"validator_index"`
	Height          int64           `json:"height"`
	Seed            string          `json:"seed"`
	ConsensusPubKey hexutil.Bytes   `json:"consensus_pub_key"`
	AppState        json.RawMessage `json:"genesis_state"`
	NextBlock       NextBlock       `json:"next_block"`
}

/*-----------------------ITmNode----------------------------*/

type ITmNode interface {
	BroadcastTxSync(tx tmtypes.Tx) (common.Hash, error)
	GetNodeInfo() Info
}

type tmNode struct {
	node *node.Node
}

func NewTmNode(node *node.Node) ITmNode {
	if node == nil {
		panic("node is nil")
	}
	return &tmNode{node: node}
}

func (tmNode *tmNode) BroadcastTxSync(tx tmtypes.Tx) (common.Hash, error) {
	resCh := make(chan *abci.Response, 1)
	err := tmNode.node.Mempool().CheckTx(tx, func(res *abci.Response) {
		resCh <- res
	}, mempool.TxInfo{})
	if err != nil {
		return common.Hash{}, err
	}
	res := <-resCh
	r := res.GetCheckTx()
	if r.Code != abci.CodeTypeOK {
		return common.Hash{}, errors.New(r.String())
	}
	return common.BytesToHash(tx.Hash()), nil
}

func (tmNode *tmNode) GetNodeInfo() Info {
	i := Info{}
	i.Height = tmNode.node.BlockStore().Height()
	address, _ := tmNode.node.NodeInfo().NetAddress()
	if address != nil {
		i.Seed = address.String()
	}
	pubKey, _ := tmNode.node.PrivValidator().GetPubKey()
	i.ConsensusPubKey = pubKey.Bytes()
	i.AppState = tmNode.node.GenesisDoc().AppState
	genesisData := app.GenesisData{}
	err := json.Unmarshal(i.AppState, &genesisData)
	if err == nil {
		for k, v := range genesisData.Validators {
			if bytes.Equal(v.Pubkey[:], i.ConsensusPubKey) {
				i.IsValidator = true
				i.ValidatorIndex = int64(k)
			}
		}
	}
	//bi := tmNode.app.LoadBlockInfo()
	//i.NextBlock.Number = bi.Number
	//i.NextBlock.Timestamp = bi.Timestamp
	//i.NextBlock.Hash = bi.Hash
	return i
}
