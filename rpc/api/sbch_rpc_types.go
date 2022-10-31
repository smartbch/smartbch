package api

import (
	"bytes"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	"github.com/smartbch/moeingevm/ebp"
	motypes "github.com/smartbch/moeingevm/types"
	sbchapi "github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/crosschain"
	cctypes "github.com/smartbch/smartbch/crosschain/types"
	sbchrpctypes "github.com/smartbch/smartbch/rpc/types"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

// StakingEpoch

type StakingEpoch struct {
	Number      hexutil.Uint64 `json:"number"`
	StartHeight hexutil.Uint64 `json:"startHeight"`
	EndTime     int64          `json:"endTime"`
	Nominations []*Nomination  `json:"nominations"`
	PosVotes    []*PosVote     `json:"posVotes"`
}
type Nomination struct {
	Pubkey         gethcmn.Hash `json:"pubkey"`
	NominatedCount int64        `json:"nominatedCount"`
}
type PosVote struct {
	Pubkey       gethcmn.Hash `json:"pubkey"`
	CoinDaysSlot *hexutil.Big `json:"coinDaysSlot"`
	CoinDays     float64      `json:"coinDays"`
}

func castStakingEpochs(epochs []*stakingtypes.Epoch) []*StakingEpoch {
	rpcEpochs := make([]*StakingEpoch, len(epochs))
	for i, epoch := range epochs {
		rpcEpochs[i] = castStakingEpoch(epoch)
	}
	return rpcEpochs
}
func castStakingEpoch(epoch *stakingtypes.Epoch) *StakingEpoch {
	return &StakingEpoch{
		Number:      hexutil.Uint64(epoch.Number),
		StartHeight: hexutil.Uint64(epoch.StartHeight),
		EndTime:     epoch.EndTime,
		Nominations: castNominations(epoch.Nominations),
	}
}
func castNominations(nominations []*stakingtypes.Nomination) []*Nomination {
	rpcNominations := make([]*Nomination, len(nominations))
	for i, nomination := range nominations {
		rpcNominations[i] = &Nomination{
			Pubkey:         nomination.Pubkey,
			NominatedCount: nomination.NominatedCount,
		}
	}
	return rpcNominations
}

type CCTransferInfo struct {
	UTXO         hexutil.Bytes  `json:"utxo"`
	Amount       hexutil.Uint64 `json:"amount"`
	SenderPubkey hexutil.Bytes  `json:"senderPubkey"`
}

//func castTransferInfos(ccTransferInfos []*cctypes.CCTransferInfo) []*CCTransferInfo {
//	rpcTransferInfos := make([]*CCTransferInfo, len(ccTransferInfos))
//	for i, ccTransferInfo := range ccTransferInfos {
//		rpcTransferInfos[i] = &CCTransferInfo{
//			UTXO:         ccTransferInfo.UTXO[:],
//			Amount:       hexutil.Uint64(ccTransferInfo.Amount),
//			SenderPubkey: ccTransferInfo.SenderPubkey[:],
//		}
//	}
//	return rpcTransferInfos
//}

// CallDetail

type CallDetail struct {
	Status                 int             `json:"status"`
	GasUsed                hexutil.Uint64  `json:"gasUsed"`
	OutData                hexutil.Bytes   `json:"returnData"`
	Logs                   []*CallLog      `json:"logs"`
	CreatedContractAddress gethcmn.Address `json:"contractAddress"`
	InternalTxs            []*InternalTx   `json:"internalTransactions"`
	RwLists                *RWLists        `json:"rwLists"`
}
type CallLog struct {
	Address gethcmn.Address `json:"address"`
	Topics  []gethcmn.Hash  `json:"topics"`
	Data    hexutil.Bytes   `json:"data"`
}
type RWLists struct {
	CreationCounterRList []CreationCounterRWOp `json:"creationCounterRList"`
	CreationCounterWList []CreationCounterRWOp `json:"creationCounterWList"`
	AccountRList         []AccountRWOp         `json:"accountRList"`
	AccountWList         []AccountRWOp         `json:"accountWList"`
	BytecodeRList        []BytecodeRWOp        `json:"bytecodeRList"`
	BytecodeWList        []BytecodeRWOp        `json:"bytecodeWList"`
	StorageRList         []StorageRWOp         `json:"storageRList"`
	StorageWList         []StorageRWOp         `json:"storageWList"`
	BlockHashList        []BlockHashOp         `json:"blockHashList"`
}
type CreationCounterRWOp struct {
	Lsb     uint8  `json:"lsb"`
	Counter uint64 `json:"counter"`
}
type AccountRWOp struct {
	Addr    gethcmn.Address `json:"address"`
	Nonce   hexutil.Uint64  `json:"nonce"`
	Balance *uint256.Int    `json:"balance"`
}
type BytecodeRWOp struct {
	Addr     gethcmn.Address `json:"address"`
	Bytecode hexutil.Bytes   `json:"bytecode"`
}
type StorageRWOp struct {
	Seq   hexutil.Uint64 `json:"seq"`
	Key   hexutil.Bytes  `json:"key"`
	Value hexutil.Bytes  `json:"value"`
}
type BlockHashOp struct {
	Height hexutil.Uint64 `json:"height"`
	Hash   gethcmn.Hash   `json:"hash"`
}

func toRpcCallDetail(detail *sbchapi.CallDetail) *CallDetail {
	callDetail := &CallDetail{
		Status:                 1, // success
		GasUsed:                hexutil.Uint64(detail.GasUsed),
		OutData:                detail.OutData,
		Logs:                   castMoEvmLogs(detail.Logs),
		CreatedContractAddress: detail.CreatedContractAddress,
		InternalTxs:            buildInternalCallList(detail.InternalTxCalls, detail.InternalTxReturns),
		RwLists:                castRWLists(detail.RwLists),
	}
	if ebp.StatusIsFailure(detail.Status) {
		callDetail.Status = 0 // failure
	}
	return callDetail
}

func castMoEvmLogs(evmLogs []motypes.EvmLog) []*CallLog {
	callLogs := make([]*CallLog, len(evmLogs))
	for i, evmLog := range evmLogs {
		callLogs[i] = &CallLog{
			Address: evmLog.Address,
			Topics:  evmLog.Topics,
			Data:    evmLog.Data,
		}
		if evmLog.Topics == nil {
			callLogs[i].Topics = make([]gethcmn.Hash, 0)
		}
	}
	return callLogs
}

func castRWLists(rwLists *motypes.ReadWriteLists) *RWLists {
	if rwLists == nil {
		return &RWLists{}
	}
	return &RWLists{
		CreationCounterRList: castCreationCounterOps(rwLists.CreationCounterRList),
		CreationCounterWList: castCreationCounterOps(rwLists.CreationCounterWList),
		AccountRList:         castAccountOps(rwLists.AccountRList),
		AccountWList:         castAccountOps(rwLists.AccountWList),
		BytecodeRList:        castBytecodeOps(rwLists.BytecodeRList),
		BytecodeWList:        castBytecodeOps(rwLists.BytecodeWList),
		StorageRList:         castStorageOps(rwLists.StorageRList),
		StorageWList:         castStorageOps(rwLists.StorageWList),
		BlockHashList:        castBlockHashOps(rwLists.BlockHashList),
	}
}
func castCreationCounterOps(ops []motypes.CreationCounterRWOp) []CreationCounterRWOp {
	rpcOps := make([]CreationCounterRWOp, len(ops))
	for i, op := range ops {
		rpcOps[i] = CreationCounterRWOp{
			Lsb:     op.Lsb,
			Counter: op.Counter,
		}
	}
	return rpcOps
}
func castAccountOps(ops []motypes.AccountRWOp) []AccountRWOp {
	rpcOps := make([]AccountRWOp, len(ops))
	for i, op := range ops {
		rpcOp := AccountRWOp{Addr: op.Addr}
		if len(op.Account) > 0 {
			accInfo := motypes.NewAccountInfo(op.Account)
			rpcOp.Nonce = hexutil.Uint64(accInfo.Nonce())
			rpcOp.Balance = accInfo.Balance()
		}
		rpcOps[i] = rpcOp
	}
	return rpcOps
}
func castBytecodeOps(ops []motypes.BytecodeRWOp) []BytecodeRWOp {
	rpcOps := make([]BytecodeRWOp, len(ops))
	for i, op := range ops {
		rpcOps[i] = BytecodeRWOp{
			Addr:     op.Addr,
			Bytecode: op.Bytecode,
		}
	}
	return rpcOps
}
func castStorageOps(ops []motypes.StorageRWOp) []StorageRWOp {
	rpcOps := make([]StorageRWOp, len(ops))
	for i, op := range ops {
		rpcOps[i] = StorageRWOp{
			Seq:   hexutil.Uint64(op.Seq),
			Key:   []byte(op.Key),
			Value: op.Value,
		}
	}
	return rpcOps
}
func castBlockHashOps(ops []motypes.BlockHashOp) []BlockHashOp {
	rpcOps := make([]BlockHashOp, len(ops))
	for i, op := range ops {
		rpcOps[i] = BlockHashOp{
			Height: hexutil.Uint64(op.Height),
			Hash:   op.Hash,
		}
	}
	return rpcOps
}

func TxToRpcCallDetail(tx *motypes.Transaction) *CallDetail {
	return &CallDetail{
		Status:                 int(tx.Status),
		GasUsed:                hexutil.Uint64(tx.GasUsed),
		OutData:                tx.OutData,
		Logs:                   castMoLogs(tx.Logs),
		CreatedContractAddress: tx.ContractAddress,
		InternalTxs:            buildInternalCallList(tx.InternalTxCalls, tx.InternalTxReturns),
		RwLists:                castRWLists(tx.RwLists),
	}
}
func castMoLogs(moLogs []motypes.Log) []*CallLog {
	callLogs := make([]*CallLog, len(moLogs))
	for i, moLog := range moLogs {
		callLogs[i] = &CallLog{
			Address: moLog.Address,
			Topics:  motypes.ToGethHashes(moLog.Topics),
			Data:    moLog.Data,
		}
	}
	return callLogs
}

// Cross Chain

func castOperatorInfo(ccOperatorInfo *crosschain.OperatorInfo) *sbchrpctypes.OperatorInfo {
	return &sbchrpctypes.OperatorInfo{
		Address: ccOperatorInfo.Addr,
		Pubkey:  ccOperatorInfo.Pubkey,
		RpcUrl:  string(bytes.TrimLeft(ccOperatorInfo.RpcUrl, string([]byte{0}))),
		Intro:   string(bytes.TrimLeft(ccOperatorInfo.Intro, string([]byte{0}))),
	}
}
func castMonitorInfo(ccMonitorInfo *crosschain.MonitorInfo) *sbchrpctypes.MonitorInfo {
	return &sbchrpctypes.MonitorInfo{
		Address: ccMonitorInfo.Addr,
		Pubkey:  ccMonitorInfo.Pubkey,
		Intro:   string(bytes.TrimLeft(ccMonitorInfo.Intro, string([]byte{0}))),
	}
}

func castUtxoRecords(utxoRecords []*cctypes.UTXORecord) []*sbchrpctypes.UtxoInfo {
	infos := make([]*sbchrpctypes.UtxoInfo, len(utxoRecords))
	for i, record := range utxoRecords {
		infos[i] = castUtxoRecord(record)
	}
	return infos
}

func castUtxoRecord(utxoRecord *cctypes.UTXORecord) *sbchrpctypes.UtxoInfo {
	return &sbchrpctypes.UtxoInfo{
		OwnerOfLost:      utxoRecord.OwnerOfLost,
		CovenantAddr:     utxoRecord.CovenantAddr,
		IsRedeemed:       utxoRecord.IsRedeemed,
		RedeemTarget:     utxoRecord.RedeemTarget,
		ExpectedSignTime: utxoRecord.ExpectedSignTime,
		Txid:             utxoRecord.Txid,
		Index:            utxoRecord.Index,
		Amount:           hexutil.Uint64(getUtxoAmtInSatoshi(utxoRecord)),
	}
}

func getUtxoAmtInSatoshi(utxoRecord *cctypes.UTXORecord) uint64 {
	amtWei := uint256.NewInt(0).SetBytes32(utxoRecord.Amount[:])
	return amtWei.Div(amtWei, uint256.NewInt(1e10)).Uint64()
}
