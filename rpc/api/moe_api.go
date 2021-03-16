package api

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	motypes "github.com/smartbch/moeingevm/types"
	moeapi "github.com/smartbch/smartbch/api"
	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
)

var _ MoeAPI = (*moeAPI)(nil)

type MoeAPI interface {
	GetStandbyTxQueue()
	QueryTxBySrc(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber) ([]*rpctypes.Transaction, error)
	QueryTxByDst(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber) ([]*rpctypes.Transaction, error)
	QueryTxByAddr(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber) ([]*rpctypes.Transaction, error)
	QueryLogs(addr gethcmn.Address, topics []gethcmn.Hash, startHeight, endHeight gethrpc.BlockNumber) ([]*gethtypes.Log, error)
	GetTxListByHeight(height gethrpc.BlockNumber) ([]*rpctypes.Transaction, error)
	// TODO: more methods
}

type moeAPI struct {
	backend moeapi.BackendService
}

func newMoeAPI(backend moeapi.BackendService) MoeAPI {
	return moeAPI{backend: backend}
}

func (moe moeAPI) GetStandbyTxQueue() {
	panic("implement me")
}

func (moe moeAPI) GetTxListByHeight(height gethrpc.BlockNumber) ([]*rpctypes.Transaction, error) {
	if height == gethrpc.LatestBlockNumber {
		height = gethrpc.BlockNumber(moe.backend.LatestHeight())
	}
	txs, err := moe.backend.GetTxListByHeight(uint32(height))
	if err != nil {
		return nil, err
	}
	return txsToRpcResp(txs), nil
}

func (moe moeAPI) QueryTxBySrc(addr gethcmn.Address,
	startHeight, endHeight gethrpc.BlockNumber) ([]*rpctypes.Transaction, error) {

	if startHeight == gethrpc.LatestBlockNumber {
		startHeight = gethrpc.BlockNumber(moe.backend.LatestHeight())
	}
	if endHeight == gethrpc.LatestBlockNumber {
		endHeight = gethrpc.BlockNumber(moe.backend.LatestHeight())
	}

	txs, err := moe.backend.QueryTxBySrc(addr, uint32(startHeight), uint32(endHeight)+1)
	if err != nil {
		return nil, err
	}
	return txsToRpcResp(txs), nil
}

func (moe moeAPI) QueryTxByDst(addr gethcmn.Address,
	startHeight, endHeight gethrpc.BlockNumber) ([]*rpctypes.Transaction, error) {

	if startHeight == gethrpc.LatestBlockNumber {
		startHeight = gethrpc.BlockNumber(moe.backend.LatestHeight())
	}
	if endHeight == gethrpc.LatestBlockNumber {
		endHeight = gethrpc.BlockNumber(moe.backend.LatestHeight())
	}

	txs, err := moe.backend.QueryTxByDst(addr, uint32(startHeight), uint32(endHeight)+1)
	if err != nil {
		return nil, err
	}
	return txsToRpcResp(txs), nil
}

func (moe moeAPI) QueryTxByAddr(addr gethcmn.Address,
	startHeight, endHeight gethrpc.BlockNumber) ([]*rpctypes.Transaction, error) {

	if startHeight == gethrpc.LatestBlockNumber {
		startHeight = gethrpc.BlockNumber(moe.backend.LatestHeight())
	}
	if endHeight == gethrpc.LatestBlockNumber {
		endHeight = gethrpc.BlockNumber(moe.backend.LatestHeight())
	}

	txs, err := moe.backend.QueryTxByAddr(addr, uint32(startHeight), uint32(endHeight)+1)
	if err != nil {
		return nil, err
	}
	return txsToRpcResp(txs), nil
}

func (moe moeAPI) QueryLogs(addr gethcmn.Address, topics []gethcmn.Hash,
	startHeight, endHeight gethrpc.BlockNumber) ([]*gethtypes.Log, error) {

	logs, err := moe.backend.MoeQueryLogs(addr, topics, uint32(startHeight), uint32(endHeight))
	if err != nil {
		return nil, err
	}
	return motypes.ToGethLogs(logs), nil
}
