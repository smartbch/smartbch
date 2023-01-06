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
	cctypes "github.com/smartbch/smartbch/crosschain/types"
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
	GetLostAndFoundUtxos() *sbchrpctypes.UtxoInfos
	GetCcUtxo(txid hexutil.Bytes, idx uint32) *sbchrpctypes.UtxoInfos
	GetCcInfosForTest() *cctypes.CCInfosForTest
	SetRpcKey(key string) error
	GetRpcPubkey() (string, error)
	//todo: for injection fault test
	InjectFaultForTest(faultType hexutil.Uint64) string
}

var (
	errCrossChainPaused = errors.New("cross chain paused")
)

type sbchAPI struct {
	backend sbchapi.BackendService
	logger  log.Logger

	//todo: for fault injection test
	/*
		1. inject lostAndFound in redeemable if exist
		2. inject lostAndFound in redeeming if exist
		3. inject redeemable in lostAndFound if exist
		4. inject redeemable in redeeming if exist
		5. inject redeeming in converting
		6. inject lostAndFound in converting
		7. make first txid LSB 0 in next handleUtxos
		8. make next redeem target address fault

		tips: param_number + 20: cancel the param_number injection.
		**/

	// 1
	injectLostInRedeemable bool
	// 2
	injectLostInRedeeming bool
	// 3
	injectRedeemableInLost bool
	// 4
	injectRedeemableInRedeeming bool
	// 5
	injectRedeemingInConverting bool
	// 6
	injectLostInConverting bool
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

// InjectFaultForTest
/*
1. inject lostAndFound in redeemable if exist
2. inject lostAndFound in redeeming if exist
3. inject redeemable in lostAndFound if exist
4. inject redeemable in redeeming if exist
5. inject redeeming in converting
6. inject lostAndFound in converting
7. make first txid LSB 0 in next handleUtxos
8. make next redeem target address fault
9. make transferByBurn target address 0x12

tips: param_number + 20: cancel the param_number injection. for example: 21 cancel the 1.inject lostAndFound in redeemable if exist
**/
func (sbch sbchAPI) InjectFaultForTest(faultType hexutil.Uint64) string {
	sbch.logger.Debug("sbch_injectFaultForTest")
	switch faultType {
	case 1:
		sbch.injectLostInRedeemable = true
		return "set injectLostInRedeemable true"
	case 21:
		sbch.injectLostInRedeemable = false
		return "set injectLostInRedeemable false"
	case 2:
		sbch.injectLostInRedeeming = true
		return "set injectLostInRedeeming true"
	case 22:
		sbch.injectLostInRedeeming = false
		return "set injectLostInRedeeming false"
	case 3:
		sbch.injectRedeemableInLost = true
		return "set injectRedeemableInLost true"
	case 23:
		sbch.injectRedeemableInLost = false
		return "set injectRedeemableInLost false"
	case 4:
		sbch.injectRedeemableInRedeeming = true
		return "set injectRedeemableInRedeeming true"
	case 24:
		sbch.injectRedeemableInRedeeming = false
		return "set injectRedeemableInRedeeming false"
	case 5:
		sbch.injectRedeemingInConverting = true
		return "set injectRedeemingInConverting true"
	case 25:
		sbch.injectRedeemingInConverting = false
		return "set injectRedeemingInConverting false"
	case 6:
		sbch.injectLostInConverting = true
		return "set injectLostInConverting true"
	case 26:
		sbch.injectLostInConverting = false
		return "set injectLostInConverting false"
	case 7:
		sbch.backend.InjectHandleUtxosFault()
		return "InjectionHandleUtxosFault"
	case 8:
		sbch.backend.InjectRedeemFault()
		return "InjectionRedeemFault"
	case 9:
		sbch.backend.InjectTransferByBurnFault()
		return "InjectTransferByBurnFault"
	default:
	}
	return `
1. inject lostAndFound in redeemable if exist
2. inject lostAndFound in redeeming if exist
3. inject redeemable in lostAndFound if exist
4. inject redeemable in redeeming if exist
5. inject redeeming in converting
6. inject lostAndFound in converting
7. make first txid LSB 0 in next handleUtxos
8. make next redeem target address fault
9. make transferByBurn target address 0x12

tips: param_number + 20: cancel the param_number injection. for example: 21 cancel the 1.inject lostAndFound in redeemable if exist`
}

func (sbch sbchAPI) GetCcInfo() *sbchrpctypes.CcInfo {
	sbch.logger.Debug("sbch_getCcInfo")

	info := sbchrpctypes.CcInfo{}

	allOperatorsInfo := sbch.backend.GetAllOperatorsInfo()
	allMonitorsInfo := sbch.backend.GetAllMonitorsInfo()
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
		info.LatestEpochHandled = ctx.LatestEpochHandled
		info.CovenantAddrLastChangeTime = ctx.CovenantAddrLastChangeTime
		for _, m := range ctx.MonitorsWithPauseCommand {
			info.MonitorsWithPauseCommand = append(info.MonitorsWithPauseCommand, gethcmn.Address(m).String())
		}
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

// todo: for injection fault test
func (sbch sbchAPI) addFaultUtxosForRedeeming(utxos *sbchrpctypes.UtxoInfos) {
	if sbch.injectRedeemableInRedeeming {
		redeemableUtxos := sbch.GetRedeemableUtxos()
		if len(redeemableUtxos.Infos) != 0 {
			utxos.Infos = append(utxos.Infos, redeemableUtxos.Infos[0])
			utxos.Signature = sbch.signUtxoInfos(utxos.Infos)
		} else {
			utxos.Infos = append(utxos.Infos, &sbchrpctypes.UtxoInfo{Txid: gethcmn.HexToHash("11"), Amount: hexutil.Uint64(1)})
			utxos.Signature = sbch.signUtxoInfos(utxos.Infos)
		}
	} else if sbch.injectLostInRedeeming {
		lostUtxos := sbch.GetLostAndFoundUtxos()
		if len(lostUtxos.Infos) != 0 {
			utxos.Infos = append(utxos.Infos, lostUtxos.Infos[0])
			utxos.Signature = sbch.signUtxoInfos(utxos.Infos)
		} else {
			utxos.Infos = append(utxos.Infos, &sbchrpctypes.UtxoInfo{Txid: gethcmn.HexToHash("11"), Amount: hexutil.Uint64(1)})
			utxos.Signature = sbch.signUtxoInfos(utxos.Infos)
		}
	}
}

func (sbch sbchAPI) GetRedeemingUtxosForMonitors() (*sbchrpctypes.UtxoInfos, error) {
	sbch.logger.Debug("sbch_getRedeemingUtxosForMonitors")
	utxos, err := sbch.getRedeemingUtxos(false)
	if err != nil {
		return utxos, err
	}
	sbch.addFaultUtxosForRedeeming(utxos)
	return utxos, err
}

func (sbch sbchAPI) GetRedeemingUtxosForOperators() (*sbchrpctypes.UtxoInfos, error) {
	sbch.logger.Debug("sbch_getRedeemingUtxosForOperators")
	utxos, err := sbch.getRedeemingUtxos(true)
	if err != nil {
		return utxos, err
	}
	sbch.addFaultUtxosForRedeeming(utxos)
	return utxos, err
}

func (sbch sbchAPI) getRedeemingUtxos(forOperators bool) (*sbchrpctypes.UtxoInfos, error) {
	if forOperators && sbch.backend.IsCrossChainPaused() {
		return nil, errCrossChainPaused
	}

	utxoRecords := sbch.backend.GetRedeemingUTXOs()
	if len(utxoRecords) == 0 {
		infos := sbchrpctypes.UtxoInfos{}
		infos.Signature = sbch.signUtxoInfos(infos.Infos)
		return &infos, nil
	}

	operatorPubkeys, monitorPubkeys := sbch.backend.GetOperatorAndMonitorPubkeys()
	currCovenant, err := covenant.NewDefaultCcCovenant(operatorPubkeys, monitorPubkeys)
	if err != nil {
		sbch.logger.Error("failed to create CcCovenant", "err", err.Error())
		return nil, err
	}
	currCovenantAddr, _ := currCovenant.GetP2SHAddress20()

	oldOpPubkeys, oldMoPubkeys := sbch.backend.GetOldOperatorAndMonitorPubkeys()
	if len(oldOpPubkeys) == 0 {
		oldOpPubkeys = operatorPubkeys
	}
	if len(oldMoPubkeys) == 0 {
		oldMoPubkeys = monitorPubkeys
	}
	oldCovenant, err := covenant.NewDefaultCcCovenant(oldOpPubkeys, oldMoPubkeys)
	if err != nil {
		sbch.logger.Error("failed to create old CcCovenant", "err", err.Error())
		return nil, err
	}
	oldCovenantAddr, _ := oldCovenant.GetP2SHAddress20()

	var currTS int64
	if forOperators {
		currBlock, err := sbch.backend.CurrentBlock()
		if err != nil {
			sbch.logger.Info("failed to get current block", "err", err.Error())
			return nil, err
		}

		currTS = currBlock.Timestamp
	}

	utxoInfos := make([]*sbchrpctypes.UtxoInfo, 0, len(utxoRecords))
	for _, utxoRecord := range utxoRecords {
		if forOperators && utxoRecord.ExpectedSignTime > currTS {
			continue
		}

		utxoInfo := castUtxoRecord(utxoRecord)
		amt := utxoInfo.Amount
		txid := utxoRecord.Txid[:]
		vout := utxoRecord.Index

		addr, err := bchutil.NewAddressPubKeyHash(utxoRecord.RedeemTarget[:], currCovenant.Net())
		if err != nil {
			sbch.logger.Error("failed to derive BCH address", "err", err)
			continue
		}

		toAddr := addr.EncodeAddress()
		var sigHash []byte
		if utxoRecord.CovenantAddr == currCovenantAddr {
			_, sigHash, err = currCovenant.GetRedeemByUserTxSigHash(txid, vout, int64(amt), toAddr)
			if err != nil {
				sbch.logger.Error("failed to call GetRedeemByUserTxSigHash", "err", err)
				continue
			}
		} else if utxoRecord.CovenantAddr == oldCovenantAddr {
			_, sigHash, err = oldCovenant.GetRedeemByUserTxSigHash(txid, vout, int64(amt), toAddr)
			if err != nil {
				sbch.logger.Error("failed to call GetRedeemByUserTxSigHash", "err", err)
				continue
			}
		} else {
			sbch.logger.Error("invalid covenant address", "covenantAddr",
				hex.EncodeToString(utxoRecord.CovenantAddr[:]))
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

// todo: for injection fault test
func (sbch sbchAPI) addFaultUtxosForConverting(utxos *sbchrpctypes.UtxoInfos) {
	if sbch.injectRedeemingInConverting {
		redeemingUtxos, err := sbch.getRedeemingUtxos(false)
		if err != nil {
			return
		}
		if len(redeemingUtxos.Infos) != 0 {
			utxos.Infos = append(utxos.Infos, redeemingUtxos.Infos[0])
			utxos.Signature = sbch.signUtxoInfos(utxos.Infos)
		} else {
			utxos.Infos = append(utxos.Infos, &sbchrpctypes.UtxoInfo{Txid: gethcmn.HexToHash("11"), Amount: hexutil.Uint64(1)})
			utxos.Signature = sbch.signUtxoInfos(utxos.Infos)
		}
	} else if sbch.injectLostInConverting {
		lostUtxos := sbch.GetLostAndFoundUtxos()
		if len(lostUtxos.Infos) != 0 {
			utxos.Infos = append(utxos.Infos, lostUtxos.Infos[0])
			utxos.Signature = sbch.signUtxoInfos(utxos.Infos)
		} else {
			utxos.Infos = append(utxos.Infos, &sbchrpctypes.UtxoInfo{Txid: gethcmn.HexToHash("11"), Amount: hexutil.Uint64(1)})
			utxos.Signature = sbch.signUtxoInfos(utxos.Infos)
		}
	}
}

func (sbch sbchAPI) GetToBeConvertedUtxosForMonitors() (*sbchrpctypes.UtxoInfos, error) {
	sbch.logger.Debug("sbch_getToBeConvertedUtxosForMonitors")
	utxos, err := sbch.getToBeConvertedUtxos(false)
	if err != nil {
		return utxos, err
	}
	sbch.addFaultUtxosForConverting(utxos)
	return utxos, err
}

func (sbch sbchAPI) GetToBeConvertedUtxosForOperators() (*sbchrpctypes.UtxoInfos, error) {
	sbch.logger.Debug("sbch_getToBeConvertedUtxosForOperators")
	utxos, err := sbch.getToBeConvertedUtxos(true)
	if err != nil {
		return utxos, err
	}
	sbch.addFaultUtxosForConverting(utxos)
	return utxos, err
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
	if len(oldOperatorPubkeys) == 0 && len(oldMonitorPubkeys) == 0 {
		sbch.logger.Error("no old operator and monitor pubkeys")
		infos := sbchrpctypes.UtxoInfos{}
		infos.Signature = sbch.signUtxoInfos(infos.Infos)
		return &infos, nil
	}

	if len(oldOperatorPubkeys) == 0 {
		oldOperatorPubkeys = newOperatorPubkeys
	}
	if len(oldMonitorPubkeys) == 0 {
		oldMonitorPubkeys = newMonitorPubkeys
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

// todo: for injection fault test
func (sbch sbchAPI) addFaultUtxosForRedeemable(utxos *sbchrpctypes.UtxoInfos) {
	if sbch.injectLostInRedeemable {
		lostUtxos := sbch.GetLostAndFoundUtxos()
		if len(lostUtxos.Infos) != 0 {
			utxos.Infos = append(utxos.Infos, lostUtxos.Infos[0])
		} else {
			utxos.Infos = append(utxos.Infos, &sbchrpctypes.UtxoInfo{Txid: gethcmn.HexToHash("11"), Amount: hexutil.Uint64(1)})
		}
	}
}

func (sbch sbchAPI) GetRedeemableUtxos() *sbchrpctypes.UtxoInfos {
	sbch.logger.Debug("sbch_getRedeemableUtxos")
	utxoRecords := sbch.backend.GetRedeemableUtxos()
	utxoInfos := castUtxoRecords(utxoRecords)
	infos := sbchrpctypes.UtxoInfos{
		Infos: utxoInfos,
	}
	sbch.addFaultUtxosForRedeemable(&infos)
	infos.Signature = sbch.signUtxoInfos(infos.Infos)
	return &infos
}

func (sbch sbchAPI) GetCcInfosForTest() *cctypes.CCInfosForTest {
	sbch.logger.Debug("sbch_getCcInfosForTest")
	return sbch.backend.GetCcInfosForTest()
}

// todo: for injection fault test
func (sbch sbchAPI) addFaultUtxosForLost(utxos *sbchrpctypes.UtxoInfos) {
	if sbch.injectRedeemableInLost {
		redeemableUtxos := sbch.GetRedeemableUtxos()
		if len(redeemableUtxos.Infos) != 0 {
			utxos.Infos = append(utxos.Infos, redeemableUtxos.Infos[0])
		} else {
			utxos.Infos = append(utxos.Infos, &sbchrpctypes.UtxoInfo{Txid: gethcmn.HexToHash("11"), Amount: hexutil.Uint64(1)})
		}
	}
}

func (sbch sbchAPI) GetLostAndFoundUtxos() *sbchrpctypes.UtxoInfos {
	sbch.logger.Debug("sbch_getLostAndFoundUtxos")
	utxoRecords := sbch.backend.GetLostAndFoundUTXOs()
	utxoInfos := castUtxoRecords(utxoRecords)
	infos := sbchrpctypes.UtxoInfos{
		Infos: utxoInfos,
	}
	sbch.addFaultUtxosForLost(&infos)
	infos.Signature = sbch.signUtxoInfos(infos.Infos)
	return &infos
}

func (sbch sbchAPI) GetCcUtxo(txid hexutil.Bytes, idx uint32) *sbchrpctypes.UtxoInfos {
	sbch.logger.Debug("sbch_getCcUtxo")

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
