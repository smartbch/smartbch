package api

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/gcash/bchutil"
	"github.com/tendermint/tendermint/libs/log"

	motypes "github.com/smartbch/moeingevm/types"
	sbchapi "github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/crosschain"
	"github.com/smartbch/smartbch/crosschain/covenant"
	"github.com/smartbch/smartbch/internal/ethutils"
	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
	sbchrpctypes "github.com/smartbch/smartbch/rpc/types"
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
	GetCcInfo() *sbchrpctypes.CcInfo
	GetRedeemingUtxosForMonitors() (*sbchrpctypes.UtxoInfos, error)
	GetRedeemingUtxosForOperators() (*sbchrpctypes.UtxoInfos, error)
	GetToBeConvertedUtxosForMonitors() (*sbchrpctypes.UtxoInfos, error)
	GetToBeConvertedUtxosForOperators() (*sbchrpctypes.UtxoInfos, error)
	GetRedeemableUtxos() *sbchrpctypes.UtxoInfos
	GetCcUtxo(txid hexutil.Bytes, idx uint32) *sbchrpctypes.UtxoInfos
	SetRpcKey(key string) error
	GetRpcPubkey() (string, error)
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

func (sbch sbchAPI) GetCcInfo() *sbchrpctypes.CcInfo {
	sbch.logger.Debug("sbch_getCcInfo")

	allOperatorsInfo := sbch.backend.GetAllOperatorsInfo()
	allMonitorsInfo := sbch.backend.GetAllMonitorsInfo()

	info := sbchrpctypes.CcInfo{}
	for _, operatorInfo := range allOperatorsInfo {
		if operatorInfo.ElectedTime.Uint64() > 0 {
			info.Operators = append(info.Operators, castOperatorInfo(operatorInfo))
		}
		if operatorInfo.OldElectedTime.Uint64() > 0 {
			info.OldOperators = append(info.OldOperators, castOperatorInfo(operatorInfo))
		}
	}

	for _, monitorInfo := range allMonitorsInfo {
		if monitorInfo.ElectedTime.Uint64() > 0 {
			info.Monitors = append(info.Monitors, castMonitorInfo(monitorInfo))
		}
		if monitorInfo.OldElectedTime.Uint64() > 0 {
			info.OldMonitors = append(info.OldMonitors, castMonitorInfo(monitorInfo))
		}
	}
	ctx := sbch.backend.GetCcContext()
	if ctx != nil {
		info.CurrCovenantAddress = gethcmn.Address(ctx.CurrCovenantAddr).String()
		info.LastCovenantAddress = gethcmn.Address(ctx.LastCovenantAddr).String()
		info.LastRescannedHeight = ctx.LastRescannedHeight
		info.RescannedHeight = ctx.RescanHeight
		info.RescanTime = ctx.RescanTime
		info.UTXOAlreadyHandled = ctx.UTXOAlreadyHandled
	}
	key := sbch.backend.GetRpcPrivateKey()
	if key != nil {
		bz, _ := json.Marshal(info)
		hash := sha256.Sum256(bz)
		sig, _ := crypto.Sign(hash[:], key)
		info.Signature = sig
	}
	return &info
}

func (sbch sbchAPI) GetRedeemingUtxosForMonitors() (*sbchrpctypes.UtxoInfos, error) {
	sbch.logger.Debug("sbch_getRedeemingUtxosForMonitors")
	return sbch.getRedeemingUtxos(false)
}

func (sbch sbchAPI) GetRedeemingUtxosForOperators() (*sbchrpctypes.UtxoInfos, error) {
	sbch.logger.Debug("sbch_getRedeemingUtxosForOperators")
	return sbch.getRedeemingUtxos(true)
}

func (sbch sbchAPI) getRedeemingUtxos(forOperators bool) (*sbchrpctypes.UtxoInfos, error) {
	if forOperators && sbch.backend.IsCrossChainPaused() {
		return nil, errCrossChainPaused
	}

	operatorPubkeys, monitorPubkeys := sbch.backend.GetOperatorAndMonitorPubkeys()
	ccc, err := covenant.NewDefaultCcCovenant(operatorPubkeys, monitorPubkeys)
	if err != nil {
		sbch.logger.Error("failed to create CcCovenant", "err", err.Error())
		return nil, err
	}

	var currTS int64
	if forOperators {
		currBlock, err := sbch.backend.CurrentBlock()
		if err != nil {
			sbch.logger.Info("failed to get current block", "err", err.Error())
			return nil, err
		}

		currTS = currBlock.Timestamp
	}

	utxoRecords := sbch.backend.GetRedeemingUTXOs()
	utxoInfos := make([]*sbchrpctypes.UtxoInfo, 0, len(utxoRecords))
	for _, utxoRecord := range utxoRecords {
		if forOperators && utxoRecord.ExpectedSignTime > currTS {
			continue
		}

		utxoInfo := castUtxoRecord(utxoRecord)
		amt := utxoInfo.Amount
		txid := utxoRecord.Txid[:]
		vout := utxoRecord.Index

		addr, err := bchutil.NewAddressPubKeyHash(utxoRecord.RedeemTarget[:], ccc.Net())
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
	infos := sbchrpctypes.UtxoInfos{
		Infos: utxoInfos,
	}
	infos.Signature = sbch.signUtxoInfos(infos.Infos)
	return &infos, nil
}

func (sbch sbchAPI) signUtxoInfos(infos []*sbchrpctypes.UtxoInfo) []byte {
	key := sbch.backend.GetRpcPrivateKey()
	if key != nil {
		bz, _ := json.Marshal(infos)
		hash := sha256.Sum256(bz)
		sig, _ := crypto.Sign(hash[:], key)
		return sig
	}
	return nil
}

func (sbch sbchAPI) GetToBeConvertedUtxosForMonitors() (*sbchrpctypes.UtxoInfos, error) {
	sbch.logger.Debug("sbch_getToBeConvertedUtxosForMonitors")
	return sbch.getToBeConvertedUtxos(false)
}

func (sbch sbchAPI) GetToBeConvertedUtxosForOperators() (*sbchrpctypes.UtxoInfos, error) {
	sbch.logger.Debug("sbch_getToBeConvertedUtxosForOperators")
	return sbch.getToBeConvertedUtxos(true)
}

func (sbch sbchAPI) getToBeConvertedUtxos(forOperators bool) (*sbchrpctypes.UtxoInfos, error) {
	if forOperators && sbch.backend.IsCrossChainPaused() {
		return nil, errCrossChainPaused
	}

	utxoRecords, lastCovenantAddrChangeTime := sbch.backend.GetToBeConvertedUTXOs()
	if forOperators {
		currBlock, err := sbch.backend.CurrentBlock()
		if err != nil {
			sbch.logger.Info("failed to get current block", "err", err.Error())
			return nil, err
		}

		currTS := currBlock.Timestamp
		if lastCovenantAddrChangeTime+crosschain.ExpectedConvertSignTimeDelay > currTS {
			return nil, errors.New("not match expected convert sign delay")
		}
	}

	if len(utxoRecords) == 0 {
		infos := sbchrpctypes.UtxoInfos{}
		infos.Signature = sbch.signUtxoInfos(infos.Infos)
		return &infos, nil
	}

	oldOperatorPubkeys, oldMonitorPubkeys := sbch.backend.GetOldOperatorAndMonitorPubkeys()
	newOperatorPubkeys, newMonitorPubkeys := sbch.backend.GetOperatorAndMonitorPubkeys()
	if len(oldOperatorPubkeys) == 0 || len(oldMonitorPubkeys) == 0 {
		sbch.logger.Error("no old operator or monitor pubkeys")
		infos := sbchrpctypes.UtxoInfos{}
		infos.Signature = sbch.signUtxoInfos(infos.Infos)
		return &infos, nil
	}

	ccc, err := covenant.NewDefaultCcCovenant(oldOperatorPubkeys, oldMonitorPubkeys)
	if err != nil {
		sbch.logger.Error("failed to create CcCovenant", "err", err.Error())
		return nil, err
	}

	utxoInfos := make([]*sbchrpctypes.UtxoInfo, 0, len(utxoRecords))
	for _, utxoRecord := range utxoRecords {
		utxoInfo := castUtxoRecord(utxoRecord)

		amt := utxoInfo.Amount
		txid := utxoRecord.Txid[:]
		vout := utxoRecord.Index

		_, sigHash, err := ccc.GetConvertByOperatorsTxSigHash(txid, vout, int64(amt),
			newOperatorPubkeys, newMonitorPubkeys)
		if err != nil {
			sbch.logger.Error("failed to call GetConvertByOperatorsTxSigHash", "err", err)
			continue
		}

		utxoInfo.TxSigHash = sigHash
		utxoInfos = append(utxoInfos, utxoInfo)
	}
	infos := sbchrpctypes.UtxoInfos{
		Infos: utxoInfos,
	}
	infos.Signature = sbch.signUtxoInfos(infos.Infos)
	return &infos, nil
}

func (sbch sbchAPI) GetRedeemableUtxos() *sbchrpctypes.UtxoInfos {
	sbch.logger.Debug("sbch_getRedeemableUTXOs")
	utxoRecords := sbch.backend.GetRedeemableUtxos()
	utxoInfos := castUtxoRecords(utxoRecords)
	infos := sbchrpctypes.UtxoInfos{
		Infos: utxoInfos,
	}
	infos.Signature = sbch.signUtxoInfos(infos.Infos)
	return &infos
}

func (sbch sbchAPI) GetCcUtxo(txid hexutil.Bytes, idx uint32) *sbchrpctypes.UtxoInfos {
	sbch.logger.Debug("sbch_getCcUTXO")

	var utxoId [36]byte
	copy(utxoId[:32], txid)
	binary.BigEndian.PutUint32(utxoId[32:], idx)

	utxoRecords := sbch.backend.GetUtxos([][36]byte{utxoId})
	utxoInfos := castUtxoRecords(utxoRecords)
	infos := sbchrpctypes.UtxoInfos{
		Infos: utxoInfos,
	}
	infos.Signature = sbch.signUtxoInfos(infos.Infos)
	return &infos
}

func (sbch sbchAPI) SetRpcKey(key string) error {
	sbch.logger.Debug("sbch_setRpcKey")
	ecdsaKey, _, err := ethutils.HexToPrivKey(key)
	if err != nil {
		return err
	}
	success := sbch.backend.SetRpcPrivateKey(ecdsaKey)
	if !success {
		return errors.New("already set rpc key")
	}
	return nil
}

func (sbch sbchAPI) GetRpcPubkey() (string, error) {
	sbch.logger.Debug("sbch_getRpcPubkey")
	key := sbch.backend.GetRpcPrivateKey()
	if key != nil {
		pubkey := crypto.FromECDSAPub(&key.PublicKey)
		return hex.EncodeToString(pubkey), nil
	}
	return "", errors.New("rpc pubkey not set")
}
