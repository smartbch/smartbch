package testutils

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/types"
)

type MdbBlockBuilder struct {
	block types.Block
	txs   []types.Transaction
}

func NewMdbBlockBuilder() *MdbBlockBuilder {
	return &MdbBlockBuilder{}
}

func (bb *MdbBlockBuilder) Height(h int64) *MdbBlockBuilder {
	bb.block.Number = h
	return bb
}
func (bb *MdbBlockBuilder) Hash(hash gethcmn.Hash) *MdbBlockBuilder {
	bb.block.Hash = hash
	return bb
}

func (bb *MdbBlockBuilder) Tx(txHash gethcmn.Hash, logs ...types.Log) *MdbBlockBuilder {
	bb.block.Transactions = append(bb.block.Transactions, txHash)

	for _, log := range logs {
		log.BlockNumber = uint64(bb.block.Number)
		log.BlockHash = bb.block.Hash
		log.TxHash = txHash
	}
	tx := types.Transaction{
		BlockHash:   bb.block.Hash,
		BlockNumber: bb.block.Number,
		Hash:        txHash,
		Logs:        logs,
		Status:      1,
	}
	bb.txs = append(bb.txs, tx)
	return bb
}

func (bb *MdbBlockBuilder) TxWithAddr(txHash gethcmn.Hash, fromAddr, toAddr gethcmn.Address) *MdbBlockBuilder {
	bb.block.Transactions = append(bb.block.Transactions, txHash)

	tx := types.Transaction{
		BlockHash:   bb.block.Hash,
		BlockNumber: bb.block.Number,
		Hash:        txHash,
		From:        fromAddr,
		To:          toAddr,
	}
	bb.txs = append(bb.txs, tx)
	return bb
}

func (bb *MdbBlockBuilder) Build() *modbtypes.Block {
	mdbTxList := make([]modbtypes.Tx, len(bb.txs))
	for i, tx := range bb.txs {
		txBytes, _ := tx.MarshalMsg(nil)
		mdbTxList[i] = modbtypes.Tx{
			HashId:  tx.Hash,
			Content: txBytes,
			LogList: toMdbLogs(tx.Logs),
		}
		copy(mdbTxList[i].SrcAddr[:], tx.From[:])
		copy(mdbTxList[i].DstAddr[:], tx.To[:])
	}

	bb.block.LogsBloom = createBloom(mdbTxList)
	blockInfo, _ := bb.block.MarshalMsg(nil)

	return &modbtypes.Block{
		Height:    bb.block.Number,
		BlockHash: bb.block.Hash,
		BlockInfo: blockInfo,
		TxList:    mdbTxList,
	}
}

func toMdbLogs(logs []types.Log) []modbtypes.Log {
	mdbLogs := make([]modbtypes.Log, len(logs))
	for i, log := range logs {
		mdbLogs[i] = modbtypes.Log{
			Address: log.Address,
			Topics:  log.Topics,
		}
	}
	return mdbLogs
}

func createBloom(txList []modbtypes.Tx) gethtypes.Bloom {
	var bin gethtypes.Bloom
	for _, tx := range txList {
		for _, log := range tx.LogList {
			bin.Add(log.Address[:])
			for _, b := range log.Topics {
				bin.Add(b[:])
			}
		}
	}
	return bin
}
