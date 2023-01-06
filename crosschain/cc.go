package crosschain

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

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
	E17 uint64 = 100_000_000_000_000_000
	E14 uint64 = 100_000_000_000_000
)

var (
	MaxCCAmount           uint64 = 100              // 0.01BCH
	MinCCAmount           uint64 = 10               // 0.001BCH
	MinPendingBurningLeft uint64 = 1                // 0.0001BCH
	MatureTime            int64  = 1                // 24h
	ForceTransferTime     int64  = 60 * 60 * 24 * 3 // 3days
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

	UTXOHandleDelay              int64  = 3    // For test
	ExpectedRedeemSignTimeDelay  int64  = 1    // For test
	ExpectedConvertSignTimeDelay int64  = 3    // For test
	MaxRescanBlockInterval       uint64 = 1000 // mainnet blocks

	ErrInvalidCallData         = errors.New("invalid call data")
	ErrInvalidSelector         = errors.New("invalid selector")
	ErrBalanceNotEnough        = errors.New("balance is not enough")
	ErrMustMonitor             = errors.New("only monitor")
	ErrRescanNotFinish         = errors.New("rescan not finish ")
	ErrRescanHeightTooSmall    = errors.New("rescan height too small")
	ErrRescanHeightTooBig      = errors.New("rescan height too big")
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

	Lock               sync.RWMutex
	Infos              []*types.CCTransferInfo
	LastEndRescanBlock uint64

	UTXOInitCollectDoneChan chan bool
	logger                  log.Logger

	//todo: for injection fault test
	HandleUtxosInject    bool
	RedeemInject         bool
	TransferByBurnInject bool
}

func NewCcContractExecutor(logger log.Logger, voter IVoteContract) *CcContractExecutor {
	return &CcContractExecutor{
		logger:                  logger,
		Voter:                   voter,
		UTXOInitCollectDoneChan: make(chan bool),
	}
}

var _ mevmtypes.SystemContractExecutor = &CcContractExecutor{}

func (c *CcContractExecutor) Init(ctx *mevmtypes.Context) {
	ccAcc := ctx.GetAccount(CCContractAddress)
	if ccAcc == nil { // only executed at genesis
		ccAcc = mevmtypes.ZeroAccountInfo()
		ccAcc.UpdateSequence(ccContractSequence)
		// init balance of cc address is 1000000sBCH
		ccAcc.UpdateBalance(uint256.NewInt(0).Mul(uint256.NewInt(0).Mul(uint256.NewInt(1000_000), uint256.NewInt(1e8)), uint256.NewInt(1e10)))
		ctx.SetAccount(CCContractAddress, ccAcc)
	}
	ccCtx := LoadCCContext(ctx)
	if ccCtx == nil {
		context := types.CCContext{
			RescanTime:            math.MaxInt64,
			RescanHeight:          uint64(param.StartMainnetHeightForCC),
			LastRescannedHeight:   uint64(0),
			UTXOAlreadyHandled:    true,
			TotalBurntOnMainChain: uint256.NewInt(uint64(param.AlreadyBurntOnMainChain)).Bytes32(),
			CurrCovenantAddr:      common.HexToAddress(param.GenesisCovenantAddress),
			LatestEpochHandled:    -1,
		}
		SaveCCContext(ctx, context)
		c.logger.Debug("CcContractExecutor init", "CurrCovenantAddr", context.CurrCovenantAddr, "RescanHeight", context.RescanHeight, "TotalBurntOnMainChain", context.TotalBurntOnMainChain)
	}
}

func (_ *CcContractExecutor) IsSystemContract(addr common.Address) bool {
	return bytes.Equal(addr[:], CCContractAddress[:])
}

func (c *CcContractExecutor) Execute(ctx *mevmtypes.Context, currBlock *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	if len(tx.Data) < 4 {
		status = StatusFailed
		gasUsed = tx.Gas
		outData = []byte(ErrInvalidCallData.Error())
		return
	}
	var selector [4]byte
	copy(selector[:], tx.Data[:4])
	switch selector {
	case SelectorRedeem:
		// func redeem(txid bytes32, index uint256, targetAddress address) external
		return redeem(ctx, currBlock, tx, c)
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
		gasUsed = tx.Gas
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
func redeem(ctx *mevmtypes.Context, block *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun, c *CcContractExecutor) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	amount := uint256.NewInt(0).SetBytes32(tx.Value[:])
	if amount.IsZero() {
		gasUsed = GasOfLostAndFoundRedeem
	}
	if tx.Gas < gasUsed {
		outData = []byte(ErrOutOfGas.Error())
		gasUsed = tx.Gas
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
	l, err := checkAndUpdateRedeemTX(ctx, block, txid, uint32(index.Uint64()), amount, targetAddress, context.CurrCovenantAddr, c)
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
		fmt.Printf("startrescan bas gas:%d\n", tx.Gas)
		outData = []byte(ErrOutOfGas.Error())
		gasUsed = tx.Gas
		return
	}
	if !uint256.NewInt(0).SetBytes32(tx.Value[:]).IsZero() {
		fmt.Printf("startrescan not zero value\n")
		outData = []byte(ErrNonPayable.Error())
		return
	}
	callData := tx.Data[4:]
	if len(callData) < 32 {
		fmt.Printf("startrescan bad calldata\n")
		outData = []byte(ErrInvalidCallData.Error())
		return
	}
	if !c.Voter.IsMonitor(ctx, tx.From) {
		fmt.Printf("startrescan not monitor\n")
		outData = []byte(ErrMustMonitor.Error())
		return
	}
	context := LoadCCContext(ctx)
	if context == nil {
		panic("cc context is nil")
	}
	if isPaused(context) {
		fmt.Printf("startrescan paused\n")
		outData = []byte(ErrCCPaused.Error())
		return
	}
	if context.RescanTime+UTXOHandleDelay > currBlock.Timestamp {
		fmt.Printf("startrescan context.RescanTime+UTXOHandleDelay > currBlock.Timestamp, rescantime:%d\n", context.RescanTime)
		outData = []byte(ErrRescanNotFinish.Error())
		return
	}
	rescanHeight := uint256.NewInt(0).SetBytes32(callData[:32]).Uint64()
	// todo: hardcode, delete this when merge branch
	//if context.RescanHeight == 1526600 {
	//	context.RescanHeight = 1529538
	//}
	if rescanHeight <= context.RescanHeight {
		c.logger.Debug("rescanHeight <= context.RescanHeight", "rescanHeight", rescanHeight, "context.RescanHeight", context.RescanHeight)
		outData = []byte(ErrRescanHeightTooSmall.Error())
		return
	}
	if rescanHeight >= context.RescanHeight+MaxRescanBlockInterval {
		c.logger.Debug("rescanHeight >= context.RescanHeight + MaxRescanBlockInterval", "rescanHeight", rescanHeight, "context.RescanHeight", context.RescanHeight)
		outData = []byte(ErrRescanHeightTooBig.Error())
		return
	}
	if !context.UTXOAlreadyHandled {
		fmt.Printf("context.UTXOAlreadyHandled is false\n")
		logs = append(logs, c.handleTransferInfos(ctx, currBlock, context)...)
	}
	context.LastRescannedHeight = context.RescanHeight
	context.RescanHeight = rescanHeight
	context.RescanTime = currBlock.Timestamp
	context.UTXOAlreadyHandled = false
	oldPrevCovenantAddr := context.LastCovenantAddr
	oldCurrCovenantAddr := context.CurrCovenantAddr
	fmt.Printf("startRescan prevCovenantAddress:%s,CurrentCovenantAddress:%s\n", common.BytesToAddress(oldPrevCovenantAddr[:]), common.BytesToAddress(oldCurrCovenantAddr[:]))
	logs = append(logs, c.handleOperatorOrMonitorSetChanged(ctx, currBlock, context)...)
	SaveCCContext(ctx, *context)
	c.logger.Debug("startRescan", "lastRescanHeight", context.LastRescannedHeight, "rescanHeight", context.RescanHeight)
	status = StatusSuccess
	return
}

func WaitUTXOCollectDone(ctx *mevmtypes.Context, collectDoneChannel chan bool) {
	ccCTx := LoadCCContext(ctx)
	if ccCTx.UTXOAlreadyHandled {
		return
	}
	<-collectDoneChannel
}

// pause() onlyMonitor
func (c *CcContractExecutor) pause(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	if tx.Gas < GasOfCCOp {
		outData = []byte(ErrOutOfGas.Error())
		gasUsed = tx.Gas
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
		gasUsed = tx.Gas
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
			newMonitors = append(newMonitors, m)
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
		gasUsed = tx.Gas
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
	fmt.Println("enter handle utxos")
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
	for {
		c.Lock.RLock()
		if c.LastEndRescanBlock == context.RescanHeight {
			break
		}
		fmt.Printf("cc want handle RescanHeight:%d, but watcher now is %d\n", context.RescanHeight, c.LastEndRescanBlock)
		c.Lock.RUnlock()
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Printf("handleTransferInfos inofs:%d\n", len(c.Infos))
	for _, info := range c.Infos {
		fmt.Println("txid:", hex.EncodeToString(info.UTXO.TxID[:]))
		fmt.Println("vout:", info.UTXO.Index)
		fmt.Println("amount:", hex.EncodeToString(info.UTXO.Amount[:]))
		fmt.Println("type:", info.Type)
		fmt.Println("receiver:", hex.EncodeToString(info.Receiver[:]))
		fmt.Println("covenantAddress:", hex.EncodeToString(info.CovenantAddress[:]))
	}
	for _, info := range c.Infos {
		switch info.Type {
		case types.TransferType:
			logs = append(logs, handleTransferTypeUTXO(ctx, context, block, info, c)...)
		case types.ConvertType:
			logs = append(logs, handleConvertTypeUTXO(ctx, context, info)...)
		case types.RedeemOrLostAndFoundType:
			logs = append(logs, handleRedeemOrLostAndFoundTypeUTXO(ctx, context, info)...)
		default:
		}
	}
	c.Lock.RUnlock()
	return logs
}

func handleTransferTypeUTXO(ctx *mevmtypes.Context, context *types.CCContext, block *mevmtypes.BlockInfo, info *types.CCTransferInfo, c *CcContractExecutor) []mevmtypes.EvmLog {
	r := types.UTXORecord{
		Txid:         info.UTXO.TxID,
		Index:        info.UTXO.Index,
		Amount:       info.UTXO.Amount,
		CovenantAddr: info.CovenantAddress,
	}
	if c.HandleUtxosInject {
		if r.Txid[31] != 0 {
			r.Txid[31] = 0
			c.HandleUtxosInject = false
			fmt.Printf("handleTransferTypeUTXO inject fault make txid:%s to %s\n", common.BytesToHash(info.UTXO.TxID[:]).String(), common.BytesToHash(r.Txid[:]).String())
		}
	}
	if info.CovenantAddress == context.LastCovenantAddr {
		r.OwnerOfLost = info.Receiver
		SaveUTXORecord(ctx, r)
		fmt.Printf("handleTransferTypeUTXO info.CovenantAddress == context.LastCovenantAddr\n")
		// todo: for test
		infos := LoadInternalInfoForTest(ctx)
		infos.TotalTransferAmountM2S = uint256.NewInt(0).Add(uint256.NewInt(0).SetBytes32(infos.TotalTransferAmountM2S[:]), uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:])).Bytes32()
		infos.TotalTransferNumsM2S++
		SaveInternalInfoForTest(ctx, *infos)
		return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}
	}
	amount := uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:])
	//maxAmount := uint256.NewInt(0).Mul(uint256.NewInt(MaxCCAmount), uint256.NewInt(E18))
	//minAmount := uint256.NewInt(0).Mul(uint256.NewInt(MinCCAmount), uint256.NewInt(E18))
	maxAmount := uint256.NewInt(0).Mul(uint256.NewInt(MaxCCAmount), uint256.NewInt(E14))
	minAmount := uint256.NewInt(0).Mul(uint256.NewInt(MinCCAmount), uint256.NewInt(E14))
	if amount.Gt(maxAmount) {
		r.OwnerOfLost = info.Receiver
		SaveUTXORecord(ctx, r)
		fmt.Printf("handleTransferTypeUTXO amount.Gt(maxAmount)\n")
		// todo: for test
		infos := LoadInternalInfoForTest(ctx)
		infos.TotalTransferAmountM2S = uint256.NewInt(0).Add(uint256.NewInt(0).SetBytes32(infos.TotalTransferAmountM2S[:]), uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:])).Bytes32()
		infos.TotalTransferNumsM2S++
		SaveInternalInfoForTest(ctx, *infos)
		return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}
	} else if amount.Lt(minAmount) {
		pendingBurning, _, totalBurntOnMain, _ := GetBurningRelativeData(ctx, context)
		//todo: change for test
		//minPendingBurningLeft := uint256.NewInt(0).Mul(uint256.NewInt(MinPendingBurningLeft), uint256.NewInt(E17))
		minPendingBurningLeft := uint256.NewInt(0).Mul(uint256.NewInt(MinPendingBurningLeft), uint256.NewInt(E14))
		//todo: hack for test
		//fmt.Printf("pending burning: %s, hack it to 0.0005BCH, and transfer amount should in (0.0006, 0.001)\n", pendingBurning.String())
		//pendingBurning = uint256.NewInt(0).Mul(uint256.NewInt(5), uint256.NewInt(1e14))
		if pendingBurning.Lt(uint256.NewInt(0).Add(minPendingBurningLeft, amount)) {
			r.OwnerOfLost = info.Receiver
			SaveUTXORecord(ctx, r)
			fmt.Printf("handleTransferTypeUTXO amount.Lt(minAmount) and lost\n")
			// todo: for test
			infos := LoadInternalInfoForTest(ctx)
			infos.TotalTransferAmountM2S = uint256.NewInt(0).Add(uint256.NewInt(0).SetBytes32(infos.TotalTransferAmountM2S[:]), uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:])).Bytes32()
			infos.TotalTransferNumsM2S++
			SaveInternalInfoForTest(ctx, *infos)
			return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}
		}
		// add total burnt on main chain here, not waiting tx minted on main chain
		totalBurntOnMain.Add(totalBurntOnMain, amount)
		context.TotalBurntOnMainChain = totalBurntOnMain.Bytes32()
		r.IsRedeemed = true
		copy(r.RedeemTarget[:], BurnAddressMainChain)
		if c.TransferByBurnInject {
			r.RedeemTarget = common.HexToAddress("0x12")
			fmt.Printf("handleTransferTypeUTXO inject fault change transfer by burn target address 0x12, txid:%s\n", common.BytesToHash(info.UTXO.TxID[:]).String())
		}
		r.ExpectedSignTime = block.Timestamp + ExpectedRedeemSignTimeDelay
		SaveUTXORecord(ctx, r)
		err := transferBch(ctx, CCContractAddress, info.Receiver, amount)
		if err != nil {
			panic(err)
		}
		fmt.Printf("handleTransferTypeUTXO amount.Lt(minAmount) and not lost\n")
		//todo: for test
		infos := LoadInternalInfoForTest(ctx)
		infos.TotalTransferByBurnAmount = uint256.NewInt(0).Add(uint256.NewInt(0).SetBytes32(infos.TotalTransferByBurnAmount[:]), amount).Bytes32()
		infos.TotalTransferByBurnNums++
		SaveInternalInfoForTest(ctx, *infos)
		return []mevmtypes.EvmLog{buildRedeemLog(r.Txid, r.Index, context.CurrCovenantAddr, types.FromBurnRedeem)}
	}
	r.BornTime = block.Timestamp
	SaveUTXORecord(ctx, r)
	err := transferBch(ctx, CCContractAddress, info.Receiver, amount)
	if err != nil {
		panic(err)
	}
	fmt.Printf("handleTransferTypeUTXO normal\n")
	// todo: for test
	infos := LoadInternalInfoForTest(ctx)
	infos.TotalTransferAmountM2S = uint256.NewInt(0).Add(uint256.NewInt(0).SetBytes32(infos.TotalTransferAmountM2S[:]), uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:])).Bytes32()
	infos.TotalTransferNumsM2S++
	SaveInternalInfoForTest(ctx, *infos)
	return []mevmtypes.EvmLog{buildNewRedeemable(r.Txid, r.Index, context.CurrCovenantAddr)}
}

func handleConvertTypeUTXO(ctx *mevmtypes.Context, context *types.CCContext, info *types.CCTransferInfo) []mevmtypes.EvmLog {
	fmt.Println("handle convert type utxo")
	r := LoadUTXORecord(ctx, info.PrevUTXO.TxID, info.PrevUTXO.Index)
	if r == nil {
		fmt.Println("no record in handle convert utxo")
		return nil
	}
	originAmount := uint256.NewInt(0).SetBytes(r.Amount[:])
	newAmount := uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:])
	if originAmount.Lt(newAmount) || originAmount.Eq(newAmount) {
		panic("wrong amount in convert utxo")
	}
	//deduct miner fee used for utxo convert
	pendingBurning, totalMinerFeeForConvertTx, _, _ := GetBurningRelativeData(ctx, context)
	minerFee := uint256.NewInt(0).Sub(originAmount, newAmount)
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
	fmt.Printf("handleConvertTypeUTXO, totalMinerFeeForConvertTx:%s,prevTxid:%s,txid:%s,newAmount:%s,covenantAddress:%s\n", totalMinerFeeForConvertTx.String(),
		common.BytesToHash(info.PrevUTXO.TxID[:]).String(), common.BytesToHash(info.UTXO.TxID[:]).String(), newAmount.String(), common.BytesToAddress(info.CovenantAddress[:]).String())
	return []mevmtypes.EvmLog{buildConvertLog(r.Txid, r.Index, r.CovenantAddr, newR.Txid, newR.Index, newR.CovenantAddr)}
}

func handleRedeemOrLostAndFoundTypeUTXO(ctx *mevmtypes.Context, context *types.CCContext, info *types.CCTransferInfo) []mevmtypes.EvmLog {
	r := LoadUTXORecord(ctx, info.PrevUTXO.TxID, info.PrevUTXO.Index)
	fmt.Printf("handleRedeemOrLostAndFoundTypeUTXO, PrevUTXO.TxID:%s, PrevUTXO.Index:%d\n", common.BytesToHash(info.PrevUTXO.TxID[:]).String(), info.PrevUTXO.Index)
	if r == nil {
		fmt.Println("cannot load utxo record in handleRedeemOrLostAndFoundTypeUTXO")
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

func GetBurningRelativeData(ctx *mevmtypes.Context, context *types.CCContext) (pendingBurning, totalMinerFeeForConvertTx, totalBurntOnMainChain, totalConsumedOnMainChain *uint256.Int) {
	blockHoleBalance := ebp.GetBlackHoleBalance(ctx)
	totalBurntOnMainChain = uint256.NewInt(0).SetBytes32(context.TotalBurntOnMainChain[:])
	totalMinerFeeForConvertTx = uint256.NewInt(0).SetBytes32(context.TotalMinerFeeForConvertTx[:])
	totalConsumedOnMainChain = uint256.NewInt(0).Add(totalMinerFeeForConvertTx, totalBurntOnMainChain)
	if !blockHoleBalance.Gt(totalConsumedOnMainChain) {
		panic(ErrPendingBurningNotEnough.Error())
	}
	pendingBurning = uint256.NewInt(0).Sub(blockHoleBalance, totalConsumedOnMainChain)
	return
}

func (c *CcContractExecutor) handleOperatorOrMonitorSetChanged(ctx *mevmtypes.Context, currBlock *mevmtypes.BlockInfo, context *types.CCContext) (logs []mevmtypes.EvmLog) {
	stakingInfo := staking.LoadStakingInfo(ctx)
	currEpochNum := stakingInfo.CurrEpochNum
	//if currEpochNum == 0 {
	//	fmt.Println("handleOperatorOrMonitorSetChanged currEpochNum is zero")
	//	return nil
	//}
	if currEpochNum == context.LatestEpochHandled {
		fmt.Println("handleOperatorOrMonitorSetChanged same epoch")
		return nil
	}
	if (currEpochNum-param.StartEpochNumberForCC+1)%param.OperatorElectionEpochs == 0 {
		ElectOperators(ctx, currBlock.Timestamp, c.logger)
		context.LatestEpochHandled = currEpochNum
		fmt.Printf("elect operators, current epoch number:%d\n", currEpochNum)
	}
	// todo: monitor should call startRescan at least once in every epoch to trigger this
	if (currEpochNum-param.StartEpochNumberForCC+1)%param.MonitorElectionEpochs == 0 {
		voteInfos := loadMonitorVotes(ctx, currEpochNum)
		voteMaps := monitorVotesToMap(voteInfos)
		ElectMonitors(ctx, voteMaps, currBlock.Timestamp, c.logger)
		fmt.Printf("elect monitors, current epoch number:%d\n", currEpochNum)
	}
	changed, newAddress := c.Voter.IsOperatorOrMonitorChanged(ctx, context.CurrCovenantAddr)
	if !changed && !(context.CovenantAddrLastChangeTime != 0 && currBlock.Timestamp > context.CovenantAddrLastChangeTime+ForceTransferTime) {
		return nil
	}
	context.CovenantAddrLastChangeTime = currBlock.Timestamp
	context.LastCovenantAddr = context.CurrCovenantAddr
	context.CurrCovenantAddr = newAddress
	fmt.Printf("handleOperatorOrMonitorSetChanged changed:%v,lastCovenantAddr%s,CurrCovenantAddr:%s\n", changed, common.BytesToAddress(context.LastCovenantAddr[:]).String(), common.BytesToAddress(context.CurrCovenantAddr[:]).String())
	logs = append(logs, buildChangeAddrLog(context.LastCovenantAddr, context.CurrCovenantAddr))
	return
}

func loadMonitorVotes(ctx *mevmtypes.Context, currEpochNum int64) []*types.MonitorVoteInfo {
	var infos = make([]*types.MonitorVoteInfo, 0, param.MonitorElectionEpochs)
	for i := currEpochNum - param.MonitorElectionEpochs + 1; i <= currEpochNum; i++ {
		if i == 0 {
			continue
		}
		infos = append(infos, LoadMonitorVoteInfo(ctx, i))
	}
	return infos
}
func monitorVotesToMap(infos []*types.MonitorVoteInfo) map[[33]byte]int64 {
	var pubkeyVoteMap = make(map[[33]byte]int64)
	for _, info := range infos {
		for _, n := range info.Nominations {
			if _, ok := pubkeyVoteMap[n.Pubkey]; !ok {
				pubkeyVoteMap[n.Pubkey] = n.NominatedCount
				continue
			}
			pubkeyVoteMap[n.Pubkey] += n.NominatedCount
		}
	}
	return pubkeyVoteMap
}

func checkAndUpdateRedeemTX(ctx *mevmtypes.Context, block *mevmtypes.BlockInfo, txid [32]byte, index uint32, amount *uint256.Int, targetAddress, currCovenantAddr [20]byte, c *CcContractExecutor) (*mevmtypes.EvmLog, error) {
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
	fmt.Printf("checkAndUpdateRedeemTX passed\n")
	r.IsRedeemed = true
	r.RedeemTarget = targetAddress
	if c != nil {
		if c.RedeemInject {
			r.RedeemTarget = common.HexToAddress("11")
			fmt.Printf("redeem inject fault, txid:%s, change target to 0x11\n", common.BytesToHash(r.Txid[:]).String())
			c.RedeemInject = false
		}
	}
	r.ExpectedSignTime = block.Timestamp + ExpectedRedeemSignTimeDelay
	SaveUTXORecord(ctx, *r)
	l := buildRedeemLog(r.Txid, r.Index, r.CovenantAddr, types.FromRedeemable)
	//todo: for test
	infos := LoadInternalInfoForTest(ctx)
	infos.TotalRedeemAmountS2M = uint256.NewInt(0).Add(uint256.NewInt(0).SetBytes32(infos.TotalRedeemAmountS2M[:]), amount).Bytes32()
	infos.TotalRedeemNumsS2M++
	SaveInternalInfoForTest(ctx, *infos)
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
	fmt.Printf("checkAndUpdateLostAndFoundTX passed\n")
	r.IsRedeemed = true
	r.RedeemTarget = targetAddress
	r.ExpectedSignTime = block.Timestamp + ExpectedRedeemSignTimeDelay
	SaveUTXORecord(ctx, *r)
	l := buildRedeemLog(r.Txid, index, r.CovenantAddr, types.FromLostAndFound)
	//todo: for test
	infos := LoadInternalInfoForTest(ctx)
	infos.TotalLostAndFoundAmountS2M = uint256.NewInt(0).Add(uint256.NewInt(0).SetBytes32(infos.TotalLostAndFoundAmountS2M[:]), uint256.NewInt(0).SetBytes32(r.Amount[:])).Bytes32()
	infos.TotalLostAndFoundNumsS2M++
	SaveInternalInfoForTest(ctx, *infos)
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
			fmt.Printf("CollectOpList: HashOfEventRedeem, txid: %s\n", common.BytesToHash(redeemOp.UtxoId[:32]).String())
		case HashOfEventNewRedeemable:
			newRedeemableOp := modbtypes.NewRedeemableOp{}
			copy(newRedeemableOp.UtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(newRedeemableOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			newRedeemableOp.CovenantAddr = common.BytesToAddress(l.Topics[3][:])
			opList.NewRedeemableOps = append(opList.NewRedeemableOps, newRedeemableOp)
			fmt.Printf("CollectOpList: HashOfEventNewRedeemable, txid: %s\n", common.BytesToHash(newRedeemableOp.UtxoId[:32]).String())
		case HashOfEventNewLostAndFound:
			newLostAndFoundOp := modbtypes.NewLostAndFoundOp{}
			copy(newLostAndFoundOp.UtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(newLostAndFoundOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			newLostAndFoundOp.CovenantAddr = common.BytesToAddress(l.Topics[3][:])
			opList.NewLostAndFoundOps = append(opList.NewLostAndFoundOps, newLostAndFoundOp)
			fmt.Printf("CollectOpList: HashOfEventNewLostAndFound, txid: %s\n", common.BytesToHash(newLostAndFoundOp.UtxoId[:32]).String())
		case HashOfEventConvert:
			newConvertedOp := modbtypes.ConvertedOp{}
			copy(newConvertedOp.PrevUtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(newConvertedOp.PrevUtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			newConvertedOp.OldCovenantAddr = common.BytesToAddress(l.Topics[3][:])
			copy(newConvertedOp.UtxoId[:32], l.Data[:32])
			binary.BigEndian.PutUint32(newConvertedOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Data[32:64]).Uint64()))
			newConvertedOp.NewCovenantAddr = common.BytesToAddress(l.Data[64:])
			opList.ConvertedOps = append(opList.ConvertedOps, newConvertedOp)
			fmt.Printf("CollectOpList: convertOp, prevTxid:%s, txid: %s\n", common.BytesToHash(newConvertedOp.PrevUtxoId[:32]).String(), common.BytesToHash(newConvertedOp.UtxoId[:32]).String())
		case HashOfEventDeleted:
			newDeleteOp := modbtypes.DeletedOp{}
			copy(newDeleteOp.UtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(newDeleteOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			newDeleteOp.CovenantAddr = common.BytesToAddress(l.Topics[3][:])
			newDeleteOp.SourceType = byte(uint256.NewInt(0).SetBytes32(l.Data).Uint64())
			opList.DeletedOps = append(opList.DeletedOps, newDeleteOp)
			fmt.Printf("CollectOpList: newDeleteOp, txid: %s\n", common.BytesToHash(newDeleteOp.UtxoId[:32]).String())
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

// todo:
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
