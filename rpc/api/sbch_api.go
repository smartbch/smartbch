package api

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	motypes "github.com/smartbch/moeingevm/types"
	sbchapi "github.com/smartbch/smartbch/api"
	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
)

var _ SbchAPI = (*sbchAPI)(nil)

type SbchAPI interface {
	GetStandbyTxQueue()
	QueryTxBySrc(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber) ([]*rpctypes.Transaction, error)
	QueryTxByDst(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber) ([]*rpctypes.Transaction, error)
	QueryTxByAddr(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber) ([]*rpctypes.Transaction, error)
	QueryLogs(addr gethcmn.Address, topics []gethcmn.Hash, startHeight, endHeight gethrpc.BlockNumber) ([]*gethtypes.Log, error)
	GetTxListByHeight(height gethrpc.BlockNumber) ([]*rpctypes.Transaction, error)
	// TODO: more methods
}

type sbchAPI struct {
	backend sbchapi.BackendService
}

func newSbchAPI(backend sbchapi.BackendService) SbchAPI {
	return sbchAPI{backend: backend}
}

func (moe sbchAPI) GetStandbyTxQueue() {
	panic("implement me")
}

func (moe sbchAPI) GetTxListByHeight(height gethrpc.BlockNumber) ([]*rpctypes.Transaction, error) {
	if height == gethrpc.LatestBlockNumber {
		height = gethrpc.BlockNumber(moe.backend.LatestHeight())
	}
	txs, err := moe.backend.GetTxListByHeight(uint32(height))
	if err != nil {
		return nil, err
	}
	return txsToRpcResp(txs), nil
}

func (moe sbchAPI) QueryTxBySrc(addr gethcmn.Address,
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

func (moe sbchAPI) QueryTxByDst(addr gethcmn.Address,
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

func (moe sbchAPI) QueryTxByAddr(addr gethcmn.Address,
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

func (moe sbchAPI) QueryLogs(addr gethcmn.Address, topics []gethcmn.Hash,
	startHeight, endHeight gethrpc.BlockNumber) ([]*gethtypes.Log, error) {

	logs, err := moe.backend.MoeQueryLogs(addr, topics, uint32(startHeight), uint32(endHeight))
	if err != nil {
		return nil, err
	}
	return motypes.ToGethLogs(logs), nil
}
