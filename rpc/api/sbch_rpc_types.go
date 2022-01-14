package api

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	motypes "github.com/smartbch/moeingevm/types"
	sbchapi "github.com/smartbch/smartbch/api"
	cctypes "github.com/smartbch/smartbch/crosschain/types"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

// StakingEpoch

type StakingEpoch struct {
	Number      hexutil.Uint64 `json:"number"`
	StartHeight hexutil.Uint64 `json:"startHeight"`
	EndTime     int64          `json:"endTime"`
	Nominations []*Nomination  `json:"nominations"`
}
type Nomination struct {
	Pubkey         gethcmn.Hash `json:"pubkey"`
	NominatedCount int64        `json:"nominatedCount"`
}

func castStakingEpochs(epochs []*stakingtypes.Epoch) []*StakingEpoch {
	rpcEpochs := make([]*StakingEpoch, len(epochs))
	for i, epoch := range epochs {
		rpcEpochs[i] = &StakingEpoch{
			Number:      hexutil.Uint64(epoch.Number),
			StartHeight: hexutil.Uint64(epoch.StartHeight),
			EndTime:     epoch.EndTime,
			Nominations: castNominations(epoch.Nominations),
		}
	}
	return rpcEpochs
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

// CCEpoch

type CCEpoch struct {
	Number        hexutil.Uint64    `json:"number"`
	StartHeight   hexutil.Uint64    `json:"startHeight"`
	EndTime       int64             `json:"endTime"`
	TransferInfos []*CCTransferInfo `json:"transferInfos"`
}
type CCTransferInfo struct {
	UTXO         hexutil.Bytes  `json:"utxo"`
	Amount       hexutil.Uint64 `json:"amount"`
	SenderPubkey hexutil.Bytes  `json:"senderPubkey"`
}

func castCCEpochs(ccEpochs []*cctypes.CCEpoch) []*CCEpoch {
	rpcEpochs := make([]*CCEpoch, len(ccEpochs))
	for i, ccEpoch := range ccEpochs {
		rpcEpochs[i] = &CCEpoch{
			Number:        hexutil.Uint64(ccEpoch.Number),
			StartHeight:   hexutil.Uint64(ccEpoch.StartHeight),
			EndTime:       ccEpoch.EndTime,
			TransferInfos: castTransferInfos(ccEpoch.TransferInfos),
		}
	}
	return rpcEpochs
}
func castTransferInfos(ccTransferInfos []*cctypes.CCTransferInfo) []*CCTransferInfo {
	rpcTransferInfos := make([]*CCTransferInfo, len(ccTransferInfos))
	for i, ccTransferInfo := range ccTransferInfos {
		rpcTransferInfos[i] = &CCTransferInfo{
			UTXO:         ccTransferInfo.UTXO[:],
			Amount:       hexutil.Uint64(ccTransferInfo.Amount),
			SenderPubkey: ccTransferInfo.SenderPubkey[:],
		}
	}
	return rpcTransferInfos
}

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
	return &CallDetail{
		Status:                 detail.Status,
		GasUsed:                hexutil.Uint64(detail.GasUsed),
		OutData:                detail.OutData,
		Logs:                   toRpcLogs(detail.Logs),
		CreatedContractAddress: detail.CreatedContractAddress,
		InternalTxs:            buildInternalCallList(detail.InternalTxCalls, detail.InternalTxReturns),
		RwLists:                toRpcRWLists(detail.RwLists),
	}
}

func toRpcLogs(evmLogs []motypes.EvmLog) []*CallLog {
	callLogs := make([]*CallLog, len(evmLogs))
	for i, evmLog := range evmLogs {
		callLogs[i] = &CallLog{
			Address: evmLog.Address,
			Topics:  evmLog.Topics,
			Data:    evmLog.Data,
		}
	}
	return callLogs
}

func toRpcRWLists(rwLists *motypes.ReadWriteLists) *RWLists {
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
		accInfo := motypes.NewAccountInfo(op.Account)
		rpcOps[i] = AccountRWOp{
			Addr:    op.Addr,
			Nonce:   hexutil.Uint64(accInfo.Nonce()),
			Balance: accInfo.Balance(),
		}
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
