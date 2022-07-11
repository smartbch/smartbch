package crosschain

import (
	"bytes"
	"errors"
	"math"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/libs/log"

	mevmtypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/crosschain/types"
)

const (
	StatusSuccess      int    = 0
	StatusFailed       int    = 1
	ccContractSequence uint64 = math.MaxUint64 - 4 /*uint64(-4)*/
	CcDelay            int64  = 20 * 60
)

var (
	//contract address, 9999
	//todo: transfer remain BCH to this address before working
	CCContractAddress [20]byte = [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x27, 0x09}

	/*------selector------*/
	SelectorRedeem      [4]byte = [4]byte{0x18, 0x92, 0xa8, 0xb3} // todo: modify it
	SelectorStartRescan [4]byte = [4]byte{0x18, 0x92, 0xa8, 0xb3} // todo: modify it
	SelectorHandelUTXOs [4]byte = [4]byte{0x18, 0x92, 0xa8, 0xb3} // todo: modify it

	HashOfEventTransferToBch [32]byte = common.HexToHash("0x4a9f09be1e2df144675144ec10cb5fe6c05504a84262275b62028189c1d410c1")
	HashOfEventBurn          [32]byte = common.HexToHash("0xeae299b236fc8161793d044c8260b3dc7f8c20b5b3b577eb7f075e4a9c3bf48d")

	GasOfCCOp uint64 = 400_000

	InvalidCallData   = errors.New("invalid call data")
	InvalidSelector   = errors.New("invalid selector")
	BchAmountNotMatch = errors.New("value not match bch amount in utxo")
	BalanceNotEnough  = errors.New("balance is not enough")
	MustMonitor       = errors.New("only monitor")
	RescanNotFinish   = errors.New("rescan not finish ")
	UTXOAlreadyHandle = errors.New("utxos in rescan already handled")
	UTXONotExist      = errors.New("utxo not exist")
	AmountNotMatch    = errors.New("redeem amount not match")
	AlreadyRedeemed   = errors.New("already redeemed")
	CCPaused          = errors.New("cc paused now")
)

var (
	SlotCCInfo             string = strings.Repeat(string([]byte{0}), 32)
	SlotScriptInfo         string = strings.Repeat(string([]byte{0}), 31) + string([]byte{2})
	SlotUnsignedConvertTxs string = strings.Repeat(string([]byte{0}), 31) + string([]byte{4})
)

type CcContractExecutor struct {
	Infos           []*types.CCTransferInfo
	UTXOCollectDone chan bool
	logger          log.Logger
}

func NewCcContractExecutor(logger log.Logger) *CcContractExecutor {
	return &CcContractExecutor{
		logger: logger,
	}
}

var _ mevmtypes.SystemContractExecutor = &CcContractExecutor{}

func (_ *CcContractExecutor) Init(ctx *mevmtypes.Context) {
	ccAcc := ctx.GetAccount(CCContractAddress)
	if ccAcc == nil { // only executed at genesis
		ccAcc = mevmtypes.ZeroAccountInfo()
		ccAcc.UpdateSequence(ccContractSequence)
		ctx.SetAccount(CCContractAddress, ccAcc)
	}
}

func (_ *CcContractExecutor) IsSystemContract(addr common.Address) bool {
	return bytes.Equal(addr[:], CCContractAddress[:])
}

func (c *CcContractExecutor) Execute(ctx *mevmtypes.Context, currBlock *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	if len(tx.Data) < 4 {
		status = StatusFailed
		outData = []byte(InvalidCallData.Error())
		return
	}
	var selector [4]byte
	copy(selector[:], tx.Data[:4])
	switch selector {
	case SelectorRedeem:
		return redeem(ctx, tx)
	case SelectorStartRescan:
		return startRescan(ctx, currBlock, tx)
	case SelectorHandelUTXOs:
		return handleUTXOs(ctx, c, currBlock, tx)
	default:
		status = StatusFailed
		outData = []byte(InvalidSelector.Error())
		return
	}
}

func (_ *CcContractExecutor) RequiredGas(_ []byte) uint64 {
	return GasOfCCOp
}

func (_ *CcContractExecutor) Run(_ []byte) ([]byte, error) {
	return nil, nil
}

//func buildTransferToMainnetEvmLog(utxo [36]byte, from common.Address, value *uint256.Int) mevmtypes.EvmLog {
//	evmLog := mevmtypes.EvmLog{
//		Address: CCContractAddress,
//		Topics:  make([]common.Hash, 0, 4),
//	}
//
//	evmLog.Topics = append(evmLog.Topics, HashOfEventTransferToBch)
//	txId := common.Hash{}
//	txId.SetBytes(utxo[:32])
//	evmLog.Topics = append(evmLog.Topics, txId)
//	vOutIndex := common.Hash{}
//	vOutIndex.SetBytes(utxo[32:])
//	evmLog.Topics = append(evmLog.Topics, vOutIndex)
//	sender := common.Hash{}
//	sender.SetBytes(from[:])
//	evmLog.Topics = append(evmLog.Topics, sender)
//
//	data := value.Bytes32()
//	evmLog.Data = append(evmLog.Data, data[:]...)
//	return evmLog
//}

// function redeem(txid bytes32, index uint256, targetPubkey bytes32) external // amount is tx.value
func redeem(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	callData := tx.Data[4:]
	if len(callData) < 32+32+32 {
		outData = []byte(InvalidCallData.Error())
		return
	}
	context := LoadCCContext(ctx)
	if context == nil {
		panic("cc context is nil")
	}
	if context.IsPaused {
		outData = []byte(CCPaused.Error())
		return
	}
	var txid [32]byte
	copy(txid[:], callData[:32])
	index := uint256.NewInt(0).SetBytes32(callData[32:64])
	amount := uint256.NewInt(0).SetBytes32(tx.Value[:])
	err := transferBch(ctx, tx.From, CCContractAddress, uint256.NewInt(0).SetBytes32(tx.Value[:]))
	if err != nil {
		outData = []byte(err.Error())
		return
	}
	var targetPubkey [32]byte
	copy(targetPubkey[:], callData[64:96])
	if err, ok := checkAndUpdateRedeemTX(ctx, txid, uint32(index.Uint64()), amount, targetPubkey); !ok {
		outData = []byte(err.Error())
		return
	}

	//serializedTx, _ := buildUnsignedTx(utxo, nil, [20]byte{})
	////build log including serializedTx
	//fmt.Printf(serializedTx)
	status = StatusSuccess
	return
}

// todo: vote contract offer this
func isMonitor(address common.Address) bool {
	return true
}

// startRescan([32]byte mainFinalizedBlockHash) onlyMonitor
func startRescan(ctx *mevmtypes.Context, currBlock *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	callData := tx.Data[4:]
	if len(callData) < 32 {
		outData = []byte(InvalidCallData.Error())
		return
	}
	if !isMonitor(tx.From) {
		outData = []byte(MustMonitor.Error())
		return
	}
	context := LoadCCContext(ctx)
	if context == nil {
		panic("cc context is nil")
	}
	context.LastRescannedHint = context.RescanHint
	copy(context.RescanHint[:], callData[:32])
	context.RescanTime = currBlock.Timestamp
	context.UTXOAlreadyHandle = false
	SaveCCContext(ctx, *context)
	status = StatusSuccess
	return
}

// handleUTXOs()
func handleUTXOs(ctx *mevmtypes.Context, executor *CcContractExecutor, currBlock *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	context := LoadCCContext(ctx)
	if context == nil {
		panic("cc context is nil")
	}
	if context.RescanTime+CcDelay > currBlock.Timestamp {
		outData = []byte(RescanNotFinish.Error())
		return
	}
	if context.UTXOAlreadyHandle {
		outData = []byte(UTXOAlreadyHandle.Error())
		return
	}
	<-executor.UTXOCollectDone
	context.UTXOAlreadyHandle = true
	handleTransferInfos(ctx, context, executor)
	status = StatusSuccess
	return
}

func handleTransferInfos(ctx *mevmtypes.Context, context *types.CCContext, executor *CcContractExecutor) {

}

func checkAndUpdateRedeemTX(ctx *mevmtypes.Context, txid [32]byte, index uint32, amount *uint256.Int, targetPubkey [32]byte) (error, bool) {
	r := LoadUTXORecord(ctx, txid, index)
	if r == nil {
		return UTXONotExist, false
	}
	if !bytes.Equal(r.Amount[:], amount.Bytes()) {
		return AmountNotMatch, false
	}
	if r.IsRedeemed {
		return AlreadyRedeemed, false
	}
	r.IsRedeemed = true
	r.RedeemTarget = targetPubkey
	SaveUTXORecord(ctx, *r)
	return nil, true
}

func transferBch(ctx *mevmtypes.Context, sender, receiver common.Address, value *uint256.Int) error {
	senderAcc := ctx.GetAccount(sender)
	balance := senderAcc.Balance()
	if balance.Lt(value) {
		return BalanceNotEnough
	}
	if !value.IsZero() {
		balance.Sub(balance, value)
		senderAcc.UpdateBalance(balance)
		ctx.SetAccount(sender, senderAcc)

		receiverAcc := ctx.GetAccount(receiver)
		if receiverAcc == nil {
			receiverAcc = mevmtypes.ZeroAccountInfo()
		}
		receiverAccBalance := receiverAcc.Balance()
		receiverAccBalance.Add(receiverAccBalance, value)
		receiverAcc.UpdateBalance(receiverAccBalance)
		ctx.SetAccount(receiver, receiverAcc)
	}
	return nil
}

var fixedMainnetFee = int64(10)

func isOperatorChanged() bool {
	return false
}
