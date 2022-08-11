package crosschain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/libs/log"

	mevmtypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/param"
)

const (
	StatusSuccess int = 0
	StatusFailed  int = 1

	ccContractSequence uint64 = math.MaxUint64 - 4 /*uint64(-4)*/
)

var (
	//contract address, 9999
	//todo: transfer remain BCH to this address before working
	CCContractAddress [20]byte = [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x27, 0x09}

	BurnAddressMainChain = common.HexToAddress("04df9d9fede348a5f82337ce87a829be2200aed6")

	/*------selector------*/
	SelectorRedeem      [4]byte = [4]byte{0x18, 0x92, 0xa8, 0xb3} // todo: modify it
	SelectorStartRescan [4]byte = [4]byte{0x18, 0x92, 0xa8, 0xb3} // todo: modify it
	SelectorHandleUTXOs [4]byte = [4]byte{0x18, 0x92, 0xa8, 0xb3} // todo: modify it
	SelectorPause       [4]byte = [4]byte{0x18, 0x92, 0xa8, 0xb3} // todo: modify it

	HashOfEventNewRedeemable   [32]byte = common.HexToHash("0x4a9f09be1e2df144675144ec10cb5fe6c05504a84262275b62028189c1d410c1")
	HashOfEventRedeem          [32]byte = common.HexToHash("0xeae299b236fc8161793d044c8260b3dc7f8c20b5b3b577eb7f075e4a9c3bf48d")
	HashOfEventNewLostAndFound [32]byte = common.HexToHash("0xeae299b236fc8161793d044c8260b3dc7f8c20b5b3b577eb7f075e4a9c3bf48d")
	HashOfEventDeleted         [32]byte = common.HexToHash("0xeae299b236fc8161793d044c8260b3dc7f8c20b5b3b577eb7f075e4a9c3bf48d")
	HashOfEventConvert         [32]byte = common.HexToHash("0xeae299b236fc8161793d044c8260b3dc7f8c20b5b3b577eb7f075e4a9c3bf48d")
	HashOfEventChangeAddr      [32]byte = common.HexToHash("0xeae299b236fc8161793d044c8260b3dc7f8c20b5b3b577eb7f075e4a9c3bf48d")

	GasOfCCOp               uint64 = 400_000
	GasOfLostAndFoundRedeem uint64 = 4000_000
	FixedMainnetFee                = int64(10)

	UTXOHandleDelay       int64 = 20 * 60
	ExpectedSignTimeDelay int64 = 5 * 60 // 5min

	InvalidCallData   = errors.New("invalid call data")
	InvalidSelector   = errors.New("invalid selector")
	BalanceNotEnough  = errors.New("balance is not enough")
	MustMonitor       = errors.New("only monitor")
	RescanNotFinish   = errors.New("rescan not finish ")
	UTXOAlreadyHandle = errors.New("utxos in rescan already handled")
	UTXONotExist      = errors.New("utxo not exist")
	AmountNotMatch    = errors.New("redeem amount not match")
	AlreadyRedeemed   = errors.New("already redeemed")
	NotLostAndFound   = errors.New("not lost and found utxo")
	NotLoser          = errors.New("not loser")
	CCPaused          = errors.New("cc paused now")
)

type CcContractExecutor struct {
	Infos            []*types.CCTransferInfo
	Voter            IVoteContract
	UTXOCollectDone  chan bool
	StartUTXOCollect chan types.UTXOCollectParam
	logger           log.Logger
}

func NewCcContractExecutor(logger log.Logger) *CcContractExecutor {
	return &CcContractExecutor{
		logger:           logger,
		Voter:            VoteContract{},
		UTXOCollectDone:  make(chan bool),
		StartUTXOCollect: make(chan types.UTXOCollectParam),
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
		// func redeem(txid bytes32, index uint256, targetAddress address) external
		return redeem(ctx, currBlock, tx)
	case SelectorStartRescan:
		// func startRescan(uint mainFinalizedBlockHeight) onlyMonitor
		return c.startRescan(ctx, currBlock, tx)
	case SelectorPause:
		// func pause() onlyMonitor
		return c.pause(ctx, tx)
	case SelectorHandleUTXOs:
		// func handleUTXOs()
		return c.handleUTXOs(ctx, currBlock, tx)
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

// function redeem(txid bytes32, index uint256, targetAddress address) external // amount is tx.value
func redeem(ctx *mevmtypes.Context, block *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
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
	var targetAddress [20]byte
	copy(targetAddress[:], callData[76:96])
	amount := uint256.NewInt(0).SetBytes32(tx.Value[:])
	if amount.IsZero() {
		gasUsed = GasOfLostAndFoundRedeem
		if err, ok := checkAndUpdateLostAndFoundTX(ctx, block, txid, uint32(index.Uint64()), tx.From, targetAddress, logs); !ok {
			outData = []byte(err.Error())
			return
		}
		status = StatusSuccess
		return
	}
	err := transferBch(ctx, tx.From, CCContractAddress, amount)
	if err != nil {
		outData = []byte(err.Error())
		return
	}
	l, err := checkAndUpdateRedeemTX(ctx, block, txid, uint32(index.Uint64()), amount, targetAddress)
	if err != nil {
		outData = []byte(err.Error())
		return
	}
	logs = append(logs, *l)
	status = StatusSuccess
	return
}

// startRescan(uint mainFinalizedBlockHeight) onlyMonitor
func (c *CcContractExecutor) startRescan(ctx *mevmtypes.Context, currBlock *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	//todo: add protect scheme to avoid monitor wrong call
	status = StatusFailed
	gasUsed = GasOfCCOp
	callData := tx.Data[4:]
	if len(callData) < 32 {
		outData = []byte(InvalidCallData.Error())
		return
	}
	if !c.Voter.IsMonitor(ctx, tx.From) {
		outData = []byte(MustMonitor.Error())
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
	context.LastRescannedHeight = context.RescanHeight
	context.RescanHeight = uint256.NewInt(0).SetBytes32(callData[:32]).Uint64()
	context.RescanTime = currBlock.Timestamp
	context.UTXOAlreadyHandle = false
	SaveCCContext(ctx, *context)
	c.StartUTXOCollect <- struct {
		BeginHeight int64
		EndHeight   int64
	}{BeginHeight: int64(context.LastRescannedHeight), EndHeight: int64(context.RescanHeight)}
	status = StatusSuccess
	return
}

// pause() onlyMonitor
func (c *CcContractExecutor) pause(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	if !c.Voter.IsMonitor(ctx, tx.From) {
		outData = []byte(MustMonitor.Error())
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
	context.IsPaused = true
	SaveCCContext(ctx, *context)
	status = StatusSuccess
	return
}

// handleUTXOs()
func (c *CcContractExecutor) handleUTXOs(ctx *mevmtypes.Context, currBlock *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	context := LoadCCContext(ctx)
	if context == nil {
		panic("cc context is nil")
	}
	if context.IsPaused {
		outData = []byte(CCPaused.Error())
		return
	}
	if context.RescanTime+UTXOHandleDelay > currBlock.Timestamp {
		outData = []byte(RescanNotFinish.Error())
		return
	}
	if context.UTXOAlreadyHandle {
		outData = []byte(UTXOAlreadyHandle.Error())
		return
	}
	logs = append(logs, c.handleTransferInfos(ctx, context)...)
	logs = append(logs, c.handleOperatorOrMonitorSetChanged(ctx, context)...)
	SaveCCContext(ctx, *context)
	status = StatusSuccess
	return
}

func (c *CcContractExecutor) handleTransferInfos(ctx *mevmtypes.Context, context *types.CCContext) (logs []mevmtypes.EvmLog) {
	<-c.UTXOCollectDone
	context.UTXOAlreadyHandle = true
	for _, info := range c.Infos {
		switch info.Type {
		case types.TransferType:
			logs = append(logs, handleTransferTypeUTXO(ctx, context, info)...)
		case types.ConvertType:
			logs = append(logs, handleConvertTypeUTXO(ctx, context, info)...)
		case types.RedeemOrLostAndFoundType:
			logs = append(logs, handleRedeemOrLostAndFoundTypeUTXO(ctx, context, info)...)
		default:
		}
	}
	return logs
}

func handleTransferTypeUTXO(ctx *mevmtypes.Context, context *types.CCContext, info *types.CCTransferInfo) []mevmtypes.EvmLog {
	r := types.UTXORecord{
		Txid:   info.UTXO.TxID,
		Index:  info.UTXO.Index,
		Amount: info.UTXO.Amount,
	}
	if info.CovenantAddress == context.LastCovenantAddr {
		r.OwnerOfLost = info.Receiver
		SaveUTXORecord(ctx, r)
		return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}
	}
	amount := uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:])
	maxAmount := uint256.NewInt(0).Mul(uint256.NewInt(param.MaxCCAmount), uint256.NewInt(1e18))
	minAmount := uint256.NewInt(0).Mul(uint256.NewInt(param.MinCCAmount), uint256.NewInt(1e18))
	if amount.Gt(maxAmount) {
		r.OwnerOfLost = info.Receiver
		SaveUTXORecord(ctx, r)
		return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}
	} else if amount.Lt(minAmount) {
		pendingBurning := uint256.NewInt(0).SetBytes32(context.PendingBurning[:])
		if pendingBurning.Lt(uint256.NewInt(1e18)) || amount.Gt(uint256.NewInt(0).Sub(pendingBurning, uint256.NewInt(1e18))) {
			r.OwnerOfLost = info.Receiver
			SaveUTXORecord(ctx, r)
			return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}
		}
		r.IsRedeemed = true
		r.RedeemTarget = BurnAddressMainChain
		SaveUTXORecord(ctx, r)
		err := transferBch(ctx, CCContractAddress, info.Receiver, amount)
		if err != nil {
			panic(err)
		}
		return nil
	}
	SaveUTXORecord(ctx, r)
	err := transferBch(ctx, CCContractAddress, info.Receiver, amount)
	if err != nil {
		panic(err)
	}
	return []mevmtypes.EvmLog{buildNewRedeemable(r.Txid, r.Index, context.CurrCovenantAddr)}
}

func handleConvertTypeUTXO(ctx *mevmtypes.Context, context *types.CCContext, info *types.CCTransferInfo) []mevmtypes.EvmLog {
	r := LoadUTXORecord(ctx, info.PrevUTXO.TxID, info.PrevUTXO.Index)
	if r == nil {
		return nil
	}
	originAmount := uint256.NewInt(0).SetBytes(r.Amount[:])
	newAmount := uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:])
	if originAmount.Lt(newAmount) || originAmount.Eq(newAmount) {
		return nil
	}
	//deduct gas fee used for utxo convert
	pendingBurning := uint256.NewInt(0).SetBytes(context.PendingBurning[:])
	gasFee := originAmount.Sub(originAmount, newAmount)
	if pendingBurning.Lt(gasFee) {
		//todo:
		panic("not cover gas fee used for utxo convert")
	}
	pendingBurning.Sub(pendingBurning, gasFee)
	context.PendingBurning = pendingBurning.Bytes32()
	newR := types.UTXORecord{
		Txid:   info.UTXO.TxID,
		Index:  info.UTXO.Index,
		Amount: newAmount.Bytes32(),
	}
	SaveUTXORecord(ctx, newR)
	DeleteUTXORecord(ctx, info.PrevUTXO.TxID, info.PrevUTXO.Index)
	return []mevmtypes.EvmLog{buildChangeAddrLog(r.Txid, r.Index, newR.CovenantAddr, newR.Txid, newR.Index)}
}

func handleRedeemOrLostAndFoundTypeUTXO(ctx *mevmtypes.Context, context *types.CCContext, info *types.CCTransferInfo) []mevmtypes.EvmLog {
	r := LoadUTXORecord(ctx, info.PrevUTXO.TxID, info.PrevUTXO.Index)
	if r == nil {
		return nil
	}
	if !r.IsRedeemed {
		panic("utxo should be redeemed")
	}
	DeleteUTXORecord(ctx, info.PrevUTXO.TxID, info.PrevUTXO.Index)
	//not check if send to correct receiver or not, monitor do this
	if r.OwnerOfLost != [20]byte{} {
		return []mevmtypes.EvmLog{buildDeletedLog(r.Txid, r.Index, r.CovenantAddr, types.FromLostAndFound)}
	} else {
		return []mevmtypes.EvmLog{buildDeletedLog(r.Txid, r.Index, r.CovenantAddr, types.FromRedeeming)}
	}
}

func (c *CcContractExecutor) handleOperatorOrMonitorSetChanged(ctx *mevmtypes.Context, context *types.CCContext) (logs []mevmtypes.EvmLog) {
	if !c.Voter.IsOperatorOrMonitorChanged(ctx, context) {
		return nil
	}
	newAddress := c.Voter.GetNewCovenantAddress(ctx)
	RedeemableUTXOs := ctx.Db.GetRedeemableUtxoIds()
	for _, utxo := range RedeemableUTXOs {
		var prevTxid [32]byte
		copy(prevTxid[:], utxo[:32])
		logs = append(logs, buildConvert(prevTxid, binary.BigEndian.Uint32(utxo[32:]), newAddress))
	}
	return
	//todo: lostAndFound tx convert?
}

func checkAndUpdateRedeemTX(ctx *mevmtypes.Context, block *mevmtypes.BlockInfo, txid [32]byte, index uint32, amount *uint256.Int, targetAddress [20]byte) (*mevmtypes.EvmLog, error) {
	r := LoadUTXORecord(ctx, txid, index)
	if r == nil {
		return nil, UTXONotExist
	}
	fmt.Printf("redeem: txid:%v, vout:%d, target:%v, amount:%s, record amount:%s\n", txid, index, targetAddress, amount.String(), uint256.NewInt(0).SetBytes32(r.Amount[:]).String())
	if !uint256.NewInt(0).SetBytes32(r.Amount[:]).Eq(amount) {
		return nil, AmountNotMatch
	}
	if r.IsRedeemed {
		return nil, AlreadyRedeemed
	}
	r.IsRedeemed = true
	r.RedeemTarget = targetAddress
	r.ExpectedSignTime = block.Timestamp + ExpectedSignTimeDelay
	SaveUTXORecord(ctx, *r)
	l := buildRedeemLog(r.Txid, r.Index, r.CovenantAddr, types.FromRedeemable)
	return &l, nil
}

func checkAndUpdateLostAndFoundTX(ctx *mevmtypes.Context, block *mevmtypes.BlockInfo, txid [32]byte, index uint32, sender common.Address, targetAddress [20]byte, logs []mevmtypes.EvmLog) (error, bool) {
	r := LoadUTXORecord(ctx, txid, index)
	if r == nil {
		return UTXONotExist, false
	}
	if r.OwnerOfLost == [20]byte{} {
		return NotLostAndFound, false
	}
	if sender != r.OwnerOfLost {
		return NotLoser, false
	}
	if r.IsRedeemed {
		return AlreadyRedeemed, false
	}
	r.IsRedeemed = true
	r.RedeemTarget = targetAddress
	r.ExpectedSignTime = block.Timestamp + ExpectedSignTimeDelay
	SaveUTXORecord(ctx, *r)
	logs = append(logs, buildRedeemLog(r.Txid, index, r.CovenantAddr, types.FromLostAndFound))
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

type IVoteContract interface {
	IsMonitor(ctx *mevmtypes.Context, address common.Address) bool
	IsOperatorOrMonitorChanged(ctx *mevmtypes.Context, ccCtx *types.CCContext) bool
	GetNewCovenantAddress(ctx *mevmtypes.Context) common.Address
}

type VoteContract struct{}

func (v VoteContract) IsMonitor(ctx *mevmtypes.Context, address common.Address) bool {
	monitors := ReadMonitorInfos(ctx, MonitorsGovSeq)
	for _, monitor := range monitors {
		if monitor.ElectedTime.Uint64() > 0 && monitor.Addr == address {
			return true
		}
	}
	return false
}

func (v VoteContract) IsOperatorOrMonitorChanged(ctx *mevmtypes.Context, ccCtx *types.CCContext) bool {
	newAddr, err := GetCCCovenantP2SHAddr(ctx)
	if err != nil {
		return false // TODO: panic
	}
	oldAddr := ccCtx.LastCovenantAddr
	return oldAddr != newAddr
}

func (v VoteContract) GetNewCovenantAddress(ctx *mevmtypes.Context) common.Address {
	addr, _ := GetCCCovenantP2SHAddr(ctx)
	// TODO: panic(err)
	return addr
}
