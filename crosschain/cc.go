package crosschain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/ebp"
	mevmtypes "github.com/smartbch/moeingevm/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/staking"
)

const (
	StatusSuccess int = 0
	StatusFailed  int = 1

	ccContractSequence uint64 = math.MaxUint64 - 3 /*uint64(-4)*/

	E18 uint64 = 1000_000_000_000_000_000
)

var (
	MaxCCAmount           uint64 = 1000
	MinCCAmount           uint64 = 1
	MinPendingBurningLeft uint64 = 10
	MatureTime            int64  = 24 * 60 * 60       // 24h
	ForceTransferTime     int64  = 6 * 30 * 24 * 3600 // 6m
)

var (
	//contract address, 10004
	//todo: transfer remain BCH to this address before working
	CCContractAddress [20]byte = [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x27, 0x14}
	// main chain burn address legacy format: 1SmartBCHBurnAddressxxxxxxy31qJGb
	BurnAddressMainChain = []byte("\x04\xdf\x9d\x9f\xed\xe3\x48\xa5\xf8\x23\x37\xce\x87\xa8\x29\xbe\x22\x00\xae\xd6")

	/*------selector------*/
	SelectorRedeem      [4]byte = [4]byte{0x04, 0x91, 0x04, 0xe5}
	SelectorStartRescan [4]byte = [4]byte{0x81, 0x3b, 0xf8, 0x3d}
	SelectorHandleUTXOs [4]byte = [4]byte{0x9c, 0x44, 0x8e, 0xfe}
	SelectorPause       [4]byte = [4]byte{0x84, 0x56, 0xcb, 0x59}
	SelectorResume      [4]byte = [4]byte{0x04, 0x6f, 0x7d, 0xa2}

	HashOfEventNewRedeemable   = crypto.Keccak256Hash([]byte("NewRedeemable(uint256,uint32,address)"))
	HashOfEventNewLostAndFound = crypto.Keccak256Hash([]byte("NewLostAndFound(uint256,uint32,address)"))
	HashOfEventRedeem          = crypto.Keccak256Hash([]byte("Redeem(uint256,uint32,address,uint8)"))
	HashOfEventChangeAddr      = crypto.Keccak256Hash([]byte("ChangeAddr(address,address)"))
	HashOfEventConvert         = crypto.Keccak256Hash([]byte("Convert(uint256,uint32,address,uint256,uint32,address)"))
	HashOfEventDeleted         = crypto.Keccak256Hash([]byte("Deleted(uint256,uint32,address,uint8)"))

	GasOfCCOp               uint64 = 400_000
	GasOfLostAndFoundRedeem uint64 = 4000_000

	UTXOHandleDelay              int64 = 20 * 60
	ExpectedRedeemSignTimeDelay  int64 = 5 * 60 // 5min
	ExpectedConvertSignTimeDelay       = ExpectedRedeemSignTimeDelay * 4

	ErrInvalidCallData         = errors.New("invalid call data")
	ErrInvalidSelector         = errors.New("invalid selector")
	ErrBalanceNotEnough        = errors.New("balance is not enough")
	ErrMustMonitor             = errors.New("only monitor")
	ErrRescanNotFinish         = errors.New("rescan not finish ")
	ErrUTXOAlreadyHandled      = errors.New("utxos in rescan already handled")
	ErrUTXONotExist            = errors.New("utxo not exist")
	ErrAmountNotMatch          = errors.New("redeem amount not match")
	ErrAlreadyRedeemed         = errors.New("already redeemed")
	ErrNotTimeToRedeem         = errors.New("not time to redeem")
	ErrNotLostAndFound         = errors.New("not lost and found utxo")
	ErrNotLoser                = errors.New("not loser")
	ErrCCPaused                = errors.New("cc paused now")
	ErrPendingBurningNotEnough = errors.New("pending burning not enough")
	ErrOutOfGas                = errors.New("out of gas")
	ErrNotCurrCovenantAddress  = errors.New("not match current covenant address")
	ErrNonPayable              = errors.New("not payable")
	ErrAlreadyPaused           = errors.New("already paused")
	ErrMustPauseFirst          = errors.New("must pause first")
)

type CcContractExecutor struct {
	Voter IVoteContract

	Lock  sync.RWMutex
	Infos []*types.CCTransferInfo

	StartUTXOCollect chan types.UTXOCollectParam
	logger           log.Logger
}

func NewCcContractExecutor(logger log.Logger, voter IVoteContract) *CcContractExecutor {
	return &CcContractExecutor{
		logger:           logger,
		Voter:            voter,
		StartUTXOCollect: make(chan types.UTXOCollectParam),
	}
}

var _ mevmtypes.SystemContractExecutor = &CcContractExecutor{}

func (c *CcContractExecutor) Init(ctx *mevmtypes.Context) {
	ccAcc := ctx.GetAccount(CCContractAddress)
	if ccAcc == nil { // only executed at genesis
		ccAcc = mevmtypes.ZeroAccountInfo()
		ccAcc.UpdateSequence(ccContractSequence)
		ctx.SetAccount(CCContractAddress, ccAcc)
	}
	ccCtx := LoadCCContext(ctx)
	if ccCtx == nil {
		address, err := c.Voter.GetCCCovenantP2SHAddr(ctx)
		if err != nil {
			panic(err)
		}
		context := types.CCContext{
			RescanTime:            math.MaxInt64,
			RescanHeight:          uint64(param.StartMainnetHeightForCC),
			LastRescannedHeight:   uint64(0),
			UTXOAlreadyHandled:    true,
			TotalBurntOnMainChain: uint256.NewInt(uint64(param.AlreadyBurntOnMainChain)).Bytes32(),
			LastCovenantAddr:      [20]byte{},
			CurrCovenantAddr:      address,
		}
		SaveCCContext(ctx, context)
		c.logger.Debug("CcContractExecutor init", "CurrCovenantAddr", address, "RescanHeight", context.RescanHeight, "TotalBurntOnMainChain", context.TotalBurntOnMainChain)
	}
}

func (_ *CcContractExecutor) IsSystemContract(addr common.Address) bool {
	return bytes.Equal(addr[:], CCContractAddress[:])
}

func (c *CcContractExecutor) Execute(ctx *mevmtypes.Context, currBlock *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	if len(tx.Data) < 4 {
		status = StatusFailed
		outData = []byte(ErrInvalidCallData.Error())
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
	case SelectorResume:
		// func resume() onlyMonitor
		return c.resume(ctx, tx)
	case SelectorHandleUTXOs:
		// func handleUTXOs()
		return c.handleUTXOs(ctx, currBlock, tx)
	default:
		status = StatusFailed
		outData = []byte(ErrInvalidSelector.Error())
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
	amount := uint256.NewInt(0).SetBytes32(tx.Value[:])
	if amount.IsZero() {
		gasUsed = GasOfLostAndFoundRedeem
	}
	if tx.Gas < gasUsed {
		outData = []byte(ErrOutOfGas.Error())
		return
	}
	callData := tx.Data[4:]
	if len(callData) < 32+32+32 {
		outData = []byte(ErrInvalidCallData.Error())
		return
	}
	context := LoadCCContext(ctx)
	if context == nil {
		panic("cc context is nil")
	}
	if isPaused(context) {
		outData = []byte(ErrCCPaused.Error())
		return
	}
	var txid [32]byte
	copy(txid[:], callData[:32])
	index := uint256.NewInt(0).SetBytes32(callData[32:64])
	var targetAddress [20]byte
	copy(targetAddress[:], callData[76:96])
	if amount.IsZero() {
		l, err := checkAndUpdateLostAndFoundTX(ctx, block, txid, uint32(index.Uint64()), tx.From, targetAddress)
		if err != nil {
			outData = []byte(err.Error())
			return
		}
		logs = append(logs, *l)
		status = StatusSuccess
		return
	}
	err := transferBch(ctx, tx.From, CCContractAddress, amount)
	if err != nil {
		outData = []byte(err.Error())
		return
	}
	l, err := checkAndUpdateRedeemTX(ctx, block, txid, uint32(index.Uint64()), amount, targetAddress, context.CurrCovenantAddr)
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
	status = StatusFailed
	gasUsed = GasOfCCOp
	if tx.Gas < GasOfCCOp {
		outData = []byte(ErrOutOfGas.Error())
		return
	}
	if !uint256.NewInt(0).SetBytes32(tx.Value[:]).IsZero() {
		outData = []byte(ErrNonPayable.Error())
		return
	}
	callData := tx.Data[4:]
	if len(callData) < 32 {
		outData = []byte(ErrInvalidCallData.Error())
		return
	}
	if !c.Voter.IsMonitor(ctx, tx.From) {
		outData = []byte(ErrMustMonitor.Error())
		return
	}
	context := LoadCCContext(ctx)
	if context == nil {
		panic("cc context is nil")
	}
	if isPaused(context) {
		outData = []byte(ErrCCPaused.Error())
		return
	}
	if context.RescanTime+UTXOHandleDelay > currBlock.Timestamp {
		outData = []byte(ErrRescanNotFinish.Error())
		return
	}
	if !context.UTXOAlreadyHandled {
		logs = append(logs, c.handleTransferInfos(ctx, currBlock, context)...)
	}
	context.LastRescannedHeight = context.RescanHeight
	context.RescanHeight = uint256.NewInt(0).SetBytes32(callData[:32]).Uint64()
	context.RescanTime = currBlock.Timestamp
	context.UTXOAlreadyHandled = false
	oldPrevCovenantAddr := context.LastCovenantAddr
	oldCurrCovenantAddr := context.CurrCovenantAddr
	logs = append(logs, c.handleOperatorOrMonitorSetChanged(ctx, currBlock, context)...)
	SaveCCContext(ctx, *context)
	c.StartUTXOCollect <- types.UTXOCollectParam{
		BeginHeight:            int64(context.LastRescannedHeight),
		EndHeight:              int64(context.RescanHeight),
		CurrentCovenantAddress: oldCurrCovenantAddr,
		PrevCovenantAddress:    oldPrevCovenantAddr,
	}
	status = StatusSuccess
	return
}

func RestartUTXOCollect(ctx *mevmtypes.Context, collectChannel chan types.UTXOCollectParam) {
	ccCTx := LoadCCContext(ctx)
	if ccCTx.UTXOAlreadyHandled {
		return
	}
	p := types.UTXOCollectParam{
		BeginHeight:            int64(ccCTx.LastRescannedHeight),
		EndHeight:              int64(ccCTx.RescanHeight),
		CurrentCovenantAddress: ccCTx.CurrCovenantAddr,
		PrevCovenantAddress:    ccCTx.LastCovenantAddr,
	}
	collectChannel <- p
}

// pause() onlyMonitor
func (c *CcContractExecutor) pause(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	if tx.Gas < GasOfCCOp {
		outData = []byte(ErrOutOfGas.Error())
		return
	}
	if !uint256.NewInt(0).SetBytes32(tx.Value[:]).IsZero() {
		outData = []byte(ErrNonPayable.Error())
		return
	}
	if !c.Voter.IsMonitor(ctx, tx.From) {
		outData = []byte(ErrMustMonitor.Error())
		return
	}
	context := LoadCCContext(ctx)
	if context == nil {
		panic("cc context is nil")
	}
	for _, m := range context.MonitorsWithPauseCommand {
		if m == tx.From {
			outData = []byte(ErrAlreadyPaused.Error())
			return
		}
	}
	context.MonitorsWithPauseCommand = append(context.MonitorsWithPauseCommand, tx.From)
	SaveCCContext(ctx, *context)
	status = StatusSuccess
	return
}

// resume() onlyMonitor
func (c *CcContractExecutor) resume(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	if tx.Gas < GasOfCCOp {
		outData = []byte(ErrOutOfGas.Error())
		return
	}
	if !uint256.NewInt(0).SetBytes32(tx.Value[:]).IsZero() {
		outData = []byte(ErrNonPayable.Error())
		return
	}
	if !c.Voter.IsMonitor(ctx, tx.From) {
		outData = []byte(ErrMustMonitor.Error())
		return
	}
	context := LoadCCContext(ctx)
	if context == nil {
		panic("cc context is nil")
	}
	var newMonitors [][20]byte
	var pauseBefore bool
	for _, m := range context.MonitorsWithPauseCommand {
		if m == tx.From {
			pauseBefore = true
		} else {
			newMonitors = append(newMonitors, tx.From)
		}
	}
	if !pauseBefore {
		outData = []byte(ErrMustPauseFirst.Error())
		return
	}
	context.MonitorsWithPauseCommand = newMonitors
	SaveCCContext(ctx, *context)
	status = StatusSuccess
	return
}

// handleUTXOs()
func (c *CcContractExecutor) handleUTXOs(ctx *mevmtypes.Context, currBlock *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	if tx.Gas < GasOfCCOp {
		outData = []byte(ErrOutOfGas.Error())
		return
	}
	if !uint256.NewInt(0).SetBytes32(tx.Value[:]).IsZero() {
		outData = []byte(ErrNonPayable.Error())
		return
	}
	context := LoadCCContext(ctx)
	if context == nil {
		panic("cc context is nil")
	}
	if isPaused(context) {
		outData = []byte(ErrCCPaused.Error())
		return
	}
	if context.RescanTime+UTXOHandleDelay > currBlock.Timestamp {
		outData = []byte(ErrRescanNotFinish.Error())
		return
	}
	if context.UTXOAlreadyHandled {
		outData = []byte(ErrUTXOAlreadyHandled.Error())
		return
	}
	logs = append(logs, c.handleTransferInfos(ctx, currBlock, context)...)
	SaveCCContext(ctx, *context)
	status = StatusSuccess
	return
}

func isPaused(context *types.CCContext) bool {
	return len(context.MonitorsWithPauseCommand) != 0
}

func (c *CcContractExecutor) handleTransferInfos(ctx *mevmtypes.Context, block *mevmtypes.BlockInfo, context *types.CCContext) (logs []mevmtypes.EvmLog) {
	context.UTXOAlreadyHandled = true
	c.Lock.RLock()
	defer c.Lock.RUnlock()
	for _, info := range c.Infos {
		switch info.Type {
		case types.TransferType:
			logs = append(logs, handleTransferTypeUTXO(ctx, context, block, info)...)
		case types.ConvertType:
			logs = append(logs, handleConvertTypeUTXO(ctx, context, info)...)
		case types.RedeemOrLostAndFoundType:
			logs = append(logs, handleRedeemOrLostAndFoundTypeUTXO(ctx, context, info)...)
		default:
		}
	}
	return logs
}

func handleTransferTypeUTXO(ctx *mevmtypes.Context, context *types.CCContext, block *mevmtypes.BlockInfo, info *types.CCTransferInfo) []mevmtypes.EvmLog {
	r := types.UTXORecord{
		Txid:         info.UTXO.TxID,
		Index:        info.UTXO.Index,
		Amount:       info.UTXO.Amount,
		CovenantAddr: info.CovenantAddress,
	}
	if info.CovenantAddress == context.LastCovenantAddr {
		r.OwnerOfLost = info.Receiver
		SaveUTXORecord(ctx, r)
		return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}
	}
	amount := uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:])
	maxAmount := uint256.NewInt(0).Mul(uint256.NewInt(MaxCCAmount), uint256.NewInt(E18))
	minAmount := uint256.NewInt(0).Mul(uint256.NewInt(MinCCAmount), uint256.NewInt(E18))
	if amount.Gt(maxAmount) {
		r.OwnerOfLost = info.Receiver
		SaveUTXORecord(ctx, r)
		return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}
	} else if amount.Lt(minAmount) {
		pendingBurning, _, totalBurntOnMain := getBurningRelativeData(ctx, context)
		minPendingBurningLeft := uint256.NewInt(0).Mul(uint256.NewInt(MinPendingBurningLeft), uint256.NewInt(E18))
		if pendingBurning.Lt(uint256.NewInt(0).Add(minPendingBurningLeft, amount)) {
			r.OwnerOfLost = info.Receiver
			SaveUTXORecord(ctx, r)
			return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}
		}
		// add total burnt on main chain here, not waiting tx minted on main chain
		totalBurntOnMain.Add(totalBurntOnMain, amount)
		context.TotalBurntOnMainChain = totalBurntOnMain.Bytes32()
		r.IsRedeemed = true
		copy(r.RedeemTarget[:], BurnAddressMainChain)
		r.ExpectedSignTime = block.Timestamp + ExpectedRedeemSignTimeDelay
		SaveUTXORecord(ctx, r)
		err := transferBch(ctx, CCContractAddress, info.Receiver, amount)
		if err != nil {
			panic(err)
		}
		return []mevmtypes.EvmLog{buildRedeemLog(r.Txid, r.Index, context.CurrCovenantAddr, types.FromBurnRedeem)}
	}
	r.BornTime = block.Timestamp
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
		panic("wrong amount in convert utxo")
	}
	//deduct miner fee used for utxo convert
	pendingBurning, totalMinerFeeForConvertTx, _ := getBurningRelativeData(ctx, context)
	minerFee := originAmount.Sub(originAmount, newAmount)
	if pendingBurning.Lt(minerFee) {
		panic(ErrPendingBurningNotEnough.Error())
	}
	// add miner fee on TotalMinerFeeForConvertTx
	totalMinerFeeForConvertTx.Add(totalMinerFeeForConvertTx, minerFee)
	context.TotalMinerFeeForConvertTx = totalMinerFeeForConvertTx.Bytes32()
	newR := types.UTXORecord{
		Txid:         info.UTXO.TxID,
		Index:        info.UTXO.Index,
		Amount:       newAmount.Bytes32(),
		CovenantAddr: info.CovenantAddress,
	}
	newR.BornTime = r.BornTime
	SaveUTXORecord(ctx, newR)
	DeleteUTXORecord(ctx, info.PrevUTXO.TxID, info.PrevUTXO.Index)
	return []mevmtypes.EvmLog{buildConvertLog(r.Txid, r.Index, r.CovenantAddr, newR.Txid, newR.Index, newR.CovenantAddr)}
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

func getBurningRelativeData(ctx *mevmtypes.Context, context *types.CCContext) (pendingBurning *uint256.Int, totalMinerFeeForConvertTx *uint256.Int, totalBurntOnMainChain *uint256.Int) {
	blockHoleBalance := ebp.GetBlackHoleBalance(ctx)
	totalBurntOnMainChain = uint256.NewInt(0).SetBytes32(context.TotalBurntOnMainChain[:])
	totalMinerFeeForConvertTx = uint256.NewInt(0).SetBytes32(context.TotalMinerFeeForConvertTx[:])
	totalConsumedOnMainChain := uint256.NewInt(0).Add(totalMinerFeeForConvertTx, totalBurntOnMainChain)
	if !blockHoleBalance.Gt(totalConsumedOnMainChain) {
		panic(ErrPendingBurningNotEnough.Error())
	}
	pendingBurning = uint256.NewInt(0).Sub(blockHoleBalance, totalConsumedOnMainChain)
	return
}

func (c *CcContractExecutor) handleOperatorOrMonitorSetChanged(ctx *mevmtypes.Context, currBlock *mevmtypes.BlockInfo, context *types.CCContext) (logs []mevmtypes.EvmLog) {
	stakingInfo := staking.LoadStakingInfo(ctx)
	currEpochNum := stakingInfo.CurrEpochNum
	if currEpochNum == context.LatestEpochHandled {
		return nil
	}
	if (currEpochNum-param.StartEpochNumberForCC+1)%param.OperatorElectionEpochs == 0 {
		ElectOperators(ctx, currBlock.Timestamp, c.logger)
		context.LatestEpochHandled = currEpochNum
	}
	// todo: monitor should call startRescan at least once in every epoch to trigger this
	if (currEpochNum-param.StartEpochNumberForCC+1)%param.MonitorElectionEpochs == 0 {
		var infos = make([]*types.MonitorVoteInfo, 0, param.MonitorElectionEpochs)
		for i := currEpochNum - param.MonitorElectionEpochs + 1; i <= currEpochNum; i++ {
			infos = append(infos, LoadMonitorVoteInfo(ctx, i))
		}
		HandleMonitorVoteInfos(ctx, currBlock.Timestamp, infos, c.logger)
	}
	changed, newAddress := c.Voter.IsOperatorOrMonitorChanged(ctx, context.CurrCovenantAddr)
	if !changed && !(context.CovenantAddrLastChangeTime != 0 && currBlock.Timestamp > context.CovenantAddrLastChangeTime+ForceTransferTime) {
		return nil
	}
	context.CovenantAddrLastChangeTime = currBlock.Timestamp
	context.LastCovenantAddr = context.CurrCovenantAddr
	context.CurrCovenantAddr = newAddress
	logs = append(logs, buildChangeAddrLog(context.LastCovenantAddr, context.CurrCovenantAddr))
	return
}

func checkAndUpdateRedeemTX(ctx *mevmtypes.Context, block *mevmtypes.BlockInfo, txid [32]byte, index uint32, amount *uint256.Int, targetAddress, currCovenantAddr [20]byte) (*mevmtypes.EvmLog, error) {
	r := LoadUTXORecord(ctx, txid, index)
	if r == nil {
		return nil, ErrUTXONotExist
	}
	if !uint256.NewInt(0).SetBytes32(r.Amount[:]).Eq(amount) {
		return nil, ErrAmountNotMatch
	}
	if r.IsRedeemed {
		return nil, ErrAlreadyRedeemed
	}
	if r.CovenantAddr != currCovenantAddr {
		return nil, ErrNotCurrCovenantAddress
	}
	if r.BornTime == 0 || r.BornTime+MatureTime >= block.Timestamp {
		return nil, ErrNotTimeToRedeem
	}
	r.IsRedeemed = true
	r.RedeemTarget = targetAddress
	r.ExpectedSignTime = block.Timestamp + ExpectedRedeemSignTimeDelay
	SaveUTXORecord(ctx, *r)
	l := buildRedeemLog(r.Txid, r.Index, r.CovenantAddr, types.FromRedeemable)
	return &l, nil
}

func checkAndUpdateLostAndFoundTX(ctx *mevmtypes.Context, block *mevmtypes.BlockInfo, txid [32]byte, index uint32, sender common.Address, targetAddress [20]byte) (*mevmtypes.EvmLog, error) {
	r := LoadUTXORecord(ctx, txid, index)
	if r == nil {
		return nil, ErrUTXONotExist
	}
	if r.OwnerOfLost == [20]byte{} {
		return nil, ErrNotLostAndFound
	}
	if sender != r.OwnerOfLost {
		return nil, ErrNotLoser
	}
	if r.IsRedeemed {
		return nil, ErrAlreadyRedeemed
	}
	r.IsRedeemed = true
	r.RedeemTarget = targetAddress
	r.ExpectedSignTime = block.Timestamp + ExpectedRedeemSignTimeDelay
	SaveUTXORecord(ctx, *r)
	l := buildRedeemLog(r.Txid, index, r.CovenantAddr, types.FromLostAndFound)
	return &l, nil
}

func transferBch(ctx *mevmtypes.Context, sender, receiver common.Address, value *uint256.Int) error {
	senderAcc := ctx.GetAccount(sender)
	balance := senderAcc.Balance()
	if balance.Lt(value) {
		return ErrBalanceNotEnough
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

func CollectOpList(mdbBlock *modbtypes.Block) modbtypes.OpListsForCcUtxo {
	opList := modbtypes.OpListsForCcUtxo{}
	events := mevmtypes.BlockToChainEvent(mdbBlock)
	for _, l := range events.Logs {
		if l.Address != CCContractAddress {
			continue
		}
		var eventHash [32]byte
		copy(eventHash[:], l.Topics[0].Bytes())
		switch common.Hash(eventHash) {
		case HashOfEventRedeem:
			redeemOp := modbtypes.RedeemOp{}
			copy(redeemOp.UtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(redeemOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			redeemOp.CovenantAddr = common.BytesToAddress(l.Topics[3][:])
			redeemOp.SourceType = byte(uint256.NewInt(0).SetBytes32(l.Data).Uint64())
			opList.RedeemOps = append(opList.RedeemOps, redeemOp)
		case HashOfEventNewRedeemable:
			newRedeemableOp := modbtypes.NewRedeemableOp{}
			copy(newRedeemableOp.UtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(newRedeemableOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			newRedeemableOp.CovenantAddr = common.BytesToAddress(l.Topics[3][:])
			opList.NewRedeemableOps = append(opList.NewRedeemableOps, newRedeemableOp)
		case HashOfEventNewLostAndFound:
			newLostAndFoundOp := modbtypes.NewLostAndFoundOp{}
			copy(newLostAndFoundOp.UtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(newLostAndFoundOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			newLostAndFoundOp.CovenantAddr = common.BytesToAddress(l.Topics[3][:])
			opList.NewLostAndFoundOps = append(opList.NewLostAndFoundOps, newLostAndFoundOp)
		case HashOfEventConvert:
			newConvertedOp := modbtypes.ConvertedOp{}
			copy(newConvertedOp.PrevUtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(newConvertedOp.PrevUtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			newConvertedOp.OldCovenantAddr = common.BytesToAddress(l.Topics[3][:])
			copy(newConvertedOp.UtxoId[:32], l.Data[:32])
			binary.BigEndian.PutUint32(newConvertedOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Data[32:64]).Uint64()))
			newConvertedOp.NewCovenantAddr = common.BytesToAddress(l.Data[64:])
			opList.ConvertedOps = append(opList.ConvertedOps, newConvertedOp)
		case HashOfEventDeleted:
			newDeleteOp := modbtypes.DeletedOp{}
			copy(newDeleteOp.UtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(newDeleteOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			newDeleteOp.CovenantAddr = common.BytesToAddress(l.Topics[3][:])
			newDeleteOp.SourceType = byte(uint256.NewInt(0).SetBytes32(l.Data).Uint64())
			opList.DeletedOps = append(opList.DeletedOps, newDeleteOp)
		default:
			continue
		}
	}
	return opList
}

type IVoteContract interface {
	IsMonitor(ctx *mevmtypes.Context, address common.Address) bool
	IsOperatorOrMonitorChanged(ctx *mevmtypes.Context, currentAddress [20]byte) (bool, common.Address)
	GetCCCovenantP2SHAddr(ctx *mevmtypes.Context) ([20]byte, error)
}

type VoteContract struct{}

func (v VoteContract) IsMonitor(ctx *mevmtypes.Context, address common.Address) bool {
	monitors := ReadMonitorInfos(ctx, param.MonitorsGovSequence)
	for _, monitor := range monitors {
		if monitor.ElectedTime.Uint64() > 0 && monitor.Addr == address {
			return true
		}
	}
	return false
}

func (v VoteContract) IsOperatorOrMonitorChanged(ctx *mevmtypes.Context, currAddress [20]byte) (bool, common.Address) {
	newAddr, err := GetCCCovenantP2SHAddr(ctx)
	if err != nil {
		panic(err)
	}
	return currAddress != newAddr, newAddr
}

func (v VoteContract) GetCCCovenantP2SHAddr(ctx *mevmtypes.Context) ([20]byte, error) {
	return GetCCCovenantP2SHAddr(ctx)
}

type MockVoteContract struct {
	IsM        bool
	IsChanged  bool
	NewAddress common.Address
}

func (m *MockVoteContract) IsMonitor(ctx *mevmtypes.Context, address common.Address) bool {
	return m.IsM
}

func (m *MockVoteContract) IsOperatorOrMonitorChanged(ctx *mevmtypes.Context, currAddress [20]byte) (bool, common.Address) {
	return m.IsChanged, m.NewAddress
}

func (m *MockVoteContract) GetCCCovenantP2SHAddr(ctx *mevmtypes.Context) ([20]byte, error) {
	return m.NewAddress, nil
}

var _ IVoteContract = &MockVoteContract{}
