package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchutil"
	"github.com/tendermint/tendermint/libs/log"

	motypes "github.com/smartbch/moeingevm/types"
	sbchapi "github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/crosschain"
	"github.com/smartbch/smartbch/crosschain/covenant"
	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
	"github.com/smartbch/smartbch/staking"
	watchertypes "github.com/smartbch/smartbch/watcher/types"
)

var _ SbchAPI = (*sbchAPI)(nil)

type SbchAPI interface {
	GetStandbyTxQueue()
	QueryTxBySrc(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error)
	QueryTxByDst(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error)
	QueryTxByAddr(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error)
	QueryLogs(addr gethcmn.Address, topics []gethcmn.Hash, startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*gethtypes.Log, error)
	GetTxListByHeight(height gethrpc.BlockNumber) ([]map[string]interface{}, error)
	GetTxListByHeightWithRange(height gethrpc.BlockNumber, start, end hexutil.Uint64) ([]map[string]interface{}, error)
	GetAddressCount(kind string, addr gethcmn.Address) hexutil.Uint64
	GetSep20AddressCount(kind string, contract, addr gethcmn.Address) hexutil.Uint64
	getVoteInfos(start, end hexutil.Uint64) ([]*watchertypes.VoteInfo, error)
	GetEpochList(from string) ([]*StakingEpoch, error)
	GetCurrEpoch(includesPosVotes *bool) (*StakingEpoch, error)
	HealthCheck(latestBlockTooOldAge hexutil.Uint64) map[string]interface{}
	GetTransactionReceipt(hash gethcmn.Hash) (map[string]interface{}, error)
	Call(args rpctypes.CallArgs, blockNr gethrpc.BlockNumberOrHash) (*CallDetail, error)
	ValidatorsInfo() json.RawMessage
	GetSyncBlock(height hexutil.Uint64) (hexutil.Bytes, error)
	GetOperators() []OperatorInfo
	GetMonitors() []MonitorInfo
	GetCcCovenantInfo() CcCovenantInfo
	GetRedeemingUtxosForMonitors() []*UtxoInfo
	GetRedeemingUtxosForOperators() ([]*UtxoInfo, error)
	GetToBeConvertedUTXOsForMonitors() []*UtxoInfo
	GetToBeConvertedUTXOsForOperators() ([]*UtxoInfo, error)
}

var (
	errCrossChainPaused = errors.New("cross chain paused")
)

type sbchAPI struct {
	backend sbchapi.BackendService
	logger  log.Logger
}

func newSbchAPI(backend sbchapi.BackendService, logger log.Logger) SbchAPI {
	return sbchAPI{
		backend: backend,
		logger:  logger,
	}
}

func (sbch sbchAPI) GetStandbyTxQueue() {
	sbch.logger.Debug("sbch_getStandbyTxQueue")
	panic("implement me")
}

func (sbch sbchAPI) GetTxListByHeight(height gethrpc.BlockNumber) ([]map[string]interface{}, error) {
	sbch.logger.Debug("sbch_getTxListByHeight")
	return sbch.GetTxListByHeightWithRange(height, 0, 0)
}

func (sbch sbchAPI) GetTxListByHeightWithRange(height gethrpc.BlockNumber, start, end hexutil.Uint64) ([]map[string]interface{}, error) {
	sbch.logger.Debug("sbch_getTxListByHeightWithRange")
	if height == gethrpc.LatestBlockNumber {
		height = gethrpc.BlockNumber(sbch.backend.LatestHeight())
	}

	iStart := int(start)
	iEnd := int(end)
	if iEnd == 0 {
		iEnd = -1
	}
	txs, _, err := sbch.backend.GetTxListByHeightWithRange(uint32(height), iStart, iEnd)
	if err != nil {
		return nil, err
	}
	return txsToReceiptsWithInternalTxs(txs), nil
}

func (sbch sbchAPI) QueryTxBySrc(addr gethcmn.Address,
	startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error) {

	sbch.logger.Debug("sbch_queryTxBySrc")
	_start, _end := sbch.prepareHeightRange(startHeight, endHeight)
	txs, sigs, err := sbch.backend.QueryTxBySrc(addr, _start, _end, uint32(limit))
	if err != nil {
		return nil, err
	}

	return txsToRpcResp(txs, sigs), nil
}

func (sbch sbchAPI) QueryTxByDst(addr gethcmn.Address,
	startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error) {

	sbch.logger.Debug("sbch_queryTxByDst")
	_start, _end := sbch.prepareHeightRange(startHeight, endHeight)
	txs, sigs, err := sbch.backend.QueryTxByDst(addr, _start, _end, uint32(limit))
	if err != nil {
		return nil, err
	}

	return txsToRpcResp(txs, sigs), nil
}

func (sbch sbchAPI) QueryTxByAddr(addr gethcmn.Address,
	startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error) {

	sbch.logger.Debug("sbch_queryTxByAddr")
	_start, _end := sbch.prepareHeightRange(startHeight, endHeight)
	txs, sigs, err := sbch.backend.QueryTxByAddr(addr, _start, _end, uint32(limit))
	if err != nil {
		return nil, err
	}

	return txsToRpcResp(txs, sigs), nil
}

func (sbch sbchAPI) prepareHeightRange(startHeight, endHeight gethrpc.BlockNumber) (uint32, uint32) {
	if startHeight == gethrpc.LatestBlockNumber {
		startHeight = gethrpc.BlockNumber(sbch.backend.LatestHeight())
	}
	if endHeight == gethrpc.LatestBlockNumber {
		endHeight = gethrpc.BlockNumber(sbch.backend.LatestHeight())
	}
	if startHeight > endHeight {
		startHeight++
	} else {
		endHeight++
	}
	return uint32(startHeight), uint32(endHeight)
}

func (sbch sbchAPI) QueryLogs(addr gethcmn.Address, topics []gethcmn.Hash,
	startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*gethtypes.Log, error) {

	sbch.logger.Debug("sbch_queryLogs")
	if startHeight == gethrpc.LatestBlockNumber {
		startHeight = gethrpc.BlockNumber(sbch.backend.LatestHeight())
	}
	if endHeight == gethrpc.LatestBlockNumber {
		endHeight = gethrpc.BlockNumber(sbch.backend.LatestHeight())
	}

	logs, err := sbch.backend.SbchQueryLogs(addr, topics,
		uint32(startHeight), uint32(endHeight), uint32(limit))
	if err != nil {
		return nil, err
	}
	return motypes.ToGethLogs(logs), nil
}

func (sbch sbchAPI) GetAddressCount(kind string, addr gethcmn.Address) hexutil.Uint64 {
	sbch.logger.Debug("sbch_getAddressCount")
	fromCount, toCount := int64(0), int64(0)
	if kind == "from" || kind == "both" {
		fromCount = sbch.backend.GetFromAddressCount(addr)
	}
	if kind == "to" || kind == "both" {
		toCount = sbch.backend.GetToAddressCount(addr)
	}
	if kind == "from" {
		return hexutil.Uint64(fromCount)
	} else if kind == "to" {
		return hexutil.Uint64(toCount)
	} else if kind == "both" {
		return hexutil.Uint64(fromCount + toCount)
	}
	return hexutil.Uint64(0)
}

func (sbch sbchAPI) GetSep20AddressCount(kind string, contract, addr gethcmn.Address) hexutil.Uint64 {
	sbch.logger.Debug("sbch_getSep20AddressCount")
	fromCount, toCount := int64(0), int64(0)
	if kind == "from" || kind == "both" {
		fromCount = sbch.backend.GetSep20FromAddressCount(contract, addr)
	}
	if kind == "to" || kind == "both" {
		toCount = sbch.backend.GetSep20ToAddressCount(contract, addr)
	}
	if kind == "from" {
		return hexutil.Uint64(fromCount)
	} else if kind == "to" {
		return hexutil.Uint64(toCount)
	} else if kind == "both" {
		return hexutil.Uint64(fromCount + toCount)
	}
	return hexutil.Uint64(0)
}

func (sbch sbchAPI) getVoteInfos(start, end hexutil.Uint64) ([]*watchertypes.VoteInfo, error) {
	sbch.logger.Debug("sbch_getVoteInfos")
	if end == 0 {
		end = start + 10
	}
	return sbch.backend.GetVoteInfos(uint64(start), uint64(end))
}
func (sbch sbchAPI) GetEpochList(from string) ([]*StakingEpoch, error) {
	epochs, err := sbch.backend.GetEpochList(from)
	if err != nil {
		return nil, err
	}
	return castStakingEpochs(epochs), nil
}
func (sbch sbchAPI) GetCurrEpoch(includesPosVotes *bool) (*StakingEpoch, error) {
	epoch := sbch.backend.GetCurrEpoch()
	epoch.Number = sbch.backend.ValidatorsInfo().CurrEpochNum
	ret := castStakingEpoch(epoch)

	if includesPosVotes != nil && *includesPosVotes {
		posVotes := sbch.backend.GetPosVotes()
		for pubKey, coinDays := range posVotes {
			ret.PosVotes = append(ret.PosVotes, &PosVote{
				Pubkey:       pubKey,
				CoinDaysSlot: (*hexutil.Big)(coinDays),
				CoinDays:     coinDaysSlotToFloat(coinDays),
			})
		}
	}

	return ret, nil
}

func coinDaysSlotToFloat(coindaysSlot *big.Int) float64 {
	fCoinDays, _ := big.NewFloat(0).Quo(
		big.NewFloat(0).SetInt(coindaysSlot),
		big.NewFloat(0).SetInt(staking.CoindayUnit.ToBig()),
	).Float64()
	return fCoinDays
}

func (sbch sbchAPI) HealthCheck(latestBlockTooOldAge hexutil.Uint64) map[string]interface{} {
	sbch.logger.Debug("sbch_healthCheck")
	if latestBlockTooOldAge == 0 {
		latestBlockTooOldAge = 30
	}

	var latestBlockHeight hexutil.Uint64
	var latestBlockTimestamp hexutil.Uint64
	var ok bool
	var msg string

	b, err := sbch.backend.CurrentBlock()
	if err == nil {
		latestBlockHeight = hexutil.Uint64(b.Number)
		latestBlockTimestamp = hexutil.Uint64(b.Timestamp)

		latestBlockAge := time.Now().Unix() - b.Timestamp
		ok = latestBlockAge < int64(latestBlockTooOldAge)
		if !ok {
			msg = fmt.Sprintf("latest block is too old: %ds", latestBlockAge)
		}
	} else {
		msg = err.Error()
	}

	return map[string]interface{}{
		"latestBlockHeight":    latestBlockHeight,
		"latestBlockTimestamp": latestBlockTimestamp,
		"ok":                   ok,
		"error":                msg,
	}
}

func (sbch sbchAPI) GetTransactionReceipt(hash gethcmn.Hash) (map[string]interface{}, error) {
	sbch.logger.Debug("sbch_getTransactionReceipt")
	tx, _, err := sbch.backend.GetTransaction(hash)
	if err != nil {
		// the transaction is not yet available
		return nil, nil
	}
	ret := txToReceiptWithInternalTxs(tx)
	return ret, nil
}

func (sbch sbchAPI) Call(args rpctypes.CallArgs, blockNr gethrpc.BlockNumberOrHash) (*CallDetail, error) {
	sbch.logger.Debug("sbch_call")

	tx, from := createGethTxFromCallArgs(args)
	height, err := getHeightArg(sbch.backend, blockNr)
	if err != nil {
		return nil, err
	}

	callDetail := sbch.backend.CallForSbch(tx, from, height)
	return toRpcCallDetail(callDetail), nil
}

func (sbch sbchAPI) ValidatorsInfo() json.RawMessage {
	sbch.logger.Debug("sbch_validatorsInfo")
	info := sbch.backend.ValidatorsInfo()
	bytes, _ := json.Marshal(info)
	return bytes
}

func (sbch sbchAPI) GetSyncBlock(height hexutil.Uint64) (hexutil.Bytes, error) {
	sbch.logger.Debug("sbch_getSyncBlock")
	return sbch.backend.GetSyncBlock(int64(height))
}

func (sbch sbchAPI) GetCcCovenantInfo() CcCovenantInfo {
	sbch.logger.Debug("sbch_getCcCovenantInfo")
	operatorPubkeys, monitorPubkeys := sbch.backend.GetOperatorAndMonitorPubkeys()
	cccInfo := CcCovenantInfo{
		OperatorPubkeys: make([]hexutil.Bytes, len(operatorPubkeys)),
		MonitorPubkeys:  make([]hexutil.Bytes, len(monitorPubkeys)),
	}

	for i, operatorPubkey := range operatorPubkeys {
		cccInfo.OperatorPubkeys[i] = operatorPubkey
	}
	for i, monitorPubkey := range monitorPubkeys {
		cccInfo.MonitorPubkeys[i] = monitorPubkey
	}

	return cccInfo
}

func (sbch sbchAPI) GetOperators() []OperatorInfo {
	sbch.logger.Debug("sbch_getOperators")
	allOperatorsInfo := sbch.backend.GetAllOperatorsInfo()

	var operators []OperatorInfo
	for _, operatorInfo := range allOperatorsInfo {
		if operatorInfo.ElectedTime.Uint64() > 0 {
			operators = append(operators, OperatorInfo{
				Address: operatorInfo.Addr,
				Pubkey:  operatorInfo.Pubkey,
				RpcUrl:  string(bytes.TrimLeft(operatorInfo.RpcUrl, string([]byte{0}))),
				Intro:   string(bytes.TrimLeft(operatorInfo.Intro, string([]byte{0}))),
			})
		}
	}

	return operators
}

func (sbch sbchAPI) GetMonitors() []MonitorInfo {
	sbch.logger.Debug("sbch_getMonitors")
	allMonitorsInfo := sbch.backend.GetAllMonitorsInfo()

	var monitors []MonitorInfo
	for _, monitorInfo := range allMonitorsInfo {
		if monitorInfo.ElectedTime.Uint64() > 0 {
			monitors = append(monitors, MonitorInfo{
				Address: monitorInfo.Addr,
				Pubkey:  monitorInfo.Pubkey,
				Intro:   string(bytes.TrimLeft(monitorInfo.Intro, string([]byte{0}))),
			})
		}
	}

	return monitors
}

func (sbch sbchAPI) GetRedeemingUtxosForMonitors() []*UtxoInfo {
	sbch.logger.Debug("sbch_getRedeemingUtxosForMonitors")
	utxoRecords := sbch.backend.GetRedeemingUTXOs()
	utxoInfos := castUtxoRecords(utxoRecords)
	return utxoInfos
}

func (sbch sbchAPI) GetRedeemingUtxosForOperators() ([]*UtxoInfo, error) {
	sbch.logger.Debug("sbch_getRedeemingUtxosForOperators")
	if sbch.backend.IsCrossChainPaused() {
		return nil, errCrossChainPaused
	}

	operatorPubkeys, monitorPubkeys := sbch.backend.GetOperatorAndMonitorPubkeys()
	ccc, err := covenant.NewDefaultCcCovenant(operatorPubkeys, monitorPubkeys)
	if err != nil {
		sbch.logger.Error("failed to create CcCovenant", "err", err.Error())
		return nil, err
	}

	currBlock, err := sbch.backend.CurrentBlock()
	if err != nil {
		sbch.logger.Info("failed to get current block", "err", err.Error())
		return nil, err
	}

	currTS := currBlock.Timestamp
	utxoRecords := sbch.backend.GetRedeemingUTXOs()

	utxoInfos := make([]*UtxoInfo, 0, len(utxoRecords))
	for _, utxoRecord := range utxoRecords {
		if utxoRecord.ExpectedSignTime > currTS {
			continue
		}

		utxoInfo := castUtxoRecord(utxoRecord)
		amt := utxoInfo.Amount
		txid := utxoRecord.Txid[:]
		vout := utxoRecord.Index

		addr, err := bchutil.NewAddressPubKeyHash(utxoRecord.RedeemTarget[:], &chaincfg.MainNetParams)
		if err != nil {
			sbch.logger.Error("failed to derive BCH address", "err", err)
			continue
		}

		toAddr := addr.EncodeAddress()
		_, sigHash, err := ccc.GetRedeemByUserTxSigHash(txid, vout, int64(amt), toAddr)
		if err != nil {
			sbch.logger.Error("failed to call GetRedeemByUserTxSigHash", "err", err)
			continue
		}

		utxoInfo.TxSigHash = sigHash
		utxoInfos = append(utxoInfos, utxoInfo)
	}

	return utxoInfos, nil
}

func (sbch sbchAPI) GetToBeConvertedUTXOsForMonitors() []*UtxoInfo {
	sbch.logger.Debug("sbch_getToBeConvertedUTXOsForMonitors")
	utxoRecords, _ := sbch.backend.GetToBeConvertedUTXOs()
	utxoInfos := castUtxoRecords(utxoRecords)
	return utxoInfos
}

func (sbch sbchAPI) GetToBeConvertedUTXOsForOperators() ([]*UtxoInfo, error) {
	sbch.logger.Debug("sbch_getToBeConvertedUTXOsForOperators")
	if sbch.backend.IsCrossChainPaused() {
		return nil, errCrossChainPaused
	}

	currBlock, err := sbch.backend.CurrentBlock()
	if err != nil {
		sbch.logger.Info("failed to get current block", "err", err.Error())
		return nil, err
	}

	currTS := currBlock.Timestamp
	utxoRecords, lastCovenantAddrChangeTime := sbch.backend.GetToBeConvertedUTXOs()
	if lastCovenantAddrChangeTime+crosschain.ExpectedConvertSignTimeDelay > currTS {
		return nil, nil
	}

	return castUtxoRecords(utxoRecords), nil
}
