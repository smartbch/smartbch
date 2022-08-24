package crosschain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/libs/log"

	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/ebp"
	mevmtypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/param"
)

const (
	StatusSuccess int = 0
	StatusFailed  int = 1

	ccContractSequence uint64 = math.MaxUint64 - 4 /*uint64(-4)*/

	E18 uint64 = 1000_000_000_000_000_000
)

var (
	MaxCCAmount           uint64 = 1000
	MinCCAmount           uint64 = 1
	MinPendingBurningLeft uint64 = 10
)

var (
	//contract address, 9999
	//todo: transfer remain BCH to this address before working
	CCContractAddress [20]byte = [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x27, 0x09}

	BurnAddressMainChain = common.HexToAddress("04df9d9fede348a5f82337ce87a829be2200aed6")

	/*------selector------*/
	SelectorRedeem      [4]byte = [4]byte{0x04, 0x91, 0x04, 0xe5}
	SelectorStartRescan [4]byte = [4]byte{0x81, 0x3b, 0xf8, 0x3d}
	SelectorHandleUTXOs [4]byte = [4]byte{0x9c, 0x44, 0x8e, 0xfe}
	SelectorPause       [4]byte = [4]byte{0x84, 0x56, 0xcb, 0x59}

	//event NewRedeemable(uint256 txid, uint32 vout, address covenantAddr);
	HashOfEventNewRedeemable [32]byte = common.HexToHash("0x15bab6fd59710de61ff75fa11875274a47fc2179068400add57ba8fb8bb4c5f1")
	//event Redeem(uint256 txid, uint32 vout, address covenantAddr, uint8 sourceType)
	HashOfEventRedeem [32]byte = common.HexToHash("0x8a9c454bba797fa0dfd6fb9d59687e2e0d5e4828de1f91ffdcf4719e1163aec0")
	//event NewLostAndFound(uint256 txid, uint32 vout, address covenantAddr);
	HashOfEventNewLostAndFound [32]byte = common.HexToHash("0x5097ba403df8e5415e49ecafe3a1610dce19fdae7df003d29d07d4f0833542ee")
	//event Deleted(uint256 txid, uint32 vout, address covenantAddr, uint8 sourceType);
	HashOfEventDeleted [32]byte = common.HexToHash("0x88efadfda2430f2d2ac267ce7158a19f80c4faef7beef319a98ba853e3ebed6f")
	//event ChangeAddr(address oldCovenantAddr, address newCovenantAddr);
	HashOfEventChangeAddr [32]byte = common.HexToHash("0x5029ef2b1891e99a0ef410ffbd2219535c135111c548ab27c4db353800fc6df6")
	//event Convert(uint256 prevTxid, uint32 prevVout, address newCovenantAddr, uint256 txid, uint32 vout)
	HashOfEventConvert [32]byte = common.HexToHash("0xb6e0dd42526bc069e197a4c86eceb9fd67d88b8c7100f1d7be12acd9b780a5fa")

	GasOfCCOp               uint64 = 400_000
	GasOfLostAndFoundRedeem uint64 = 4000_000
	FixedMainnetFee         int64  = 10

	UTXOHandleDelay       int64 = 20 * 60
	ExpectedSignTimeDelay int64 = 5 * 60 // 5min

	InvalidCallData         = errors.New("invalid call data")
	InvalidSelector         = errors.New("invalid selector")
	BalanceNotEnough        = errors.New("balance is not enough")
	MustMonitor             = errors.New("only monitor")
	RescanNotFinish         = errors.New("rescan not finish ")
	UTXOAlreadyHandled      = errors.New("utxos in rescan already handled")
	UTXONotExist            = errors.New("utxo not exist")
	AmountNotMatch          = errors.New("redeem amount not match")
	AlreadyRedeemed         = errors.New("already redeemed")
	NotLostAndFound         = errors.New("not lost and found utxo")
	NotLoser                = errors.New("not loser")
	CCPaused                = errors.New("cc paused now")
	PendingBurningNotEnough = errors.New("pending burning not enough")
)

type CcContractExecutor struct {
	Voter            IVoteContract
	UTXOCollectDone  chan []*types.CCTransferInfo
	StartUTXOCollect chan types.UTXOCollectParam
	logger           log.Logger
}

func NewCcContractExecutor(logger log.Logger, voter IVoteContract) *CcContractExecutor {
	return &CcContractExecutor{
		logger:           logger,
		Voter:            voter,
		UTXOCollectDone:  make(chan []*types.CCTransferInfo),
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
	ccCtx := LoadCCContext(ctx)
	if ccCtx == nil {
		context := types.CCContext{
			IsPaused:              false,
			RescanTime:            math.MaxInt64,
			RescanHeight:          uint64(param.EpochStartHeightForCC),
			LastRescannedHeight:   uint64(0),
			UTXOAlreadyHandled:    true,
			TotalBurntOnMainChain: uint256.NewInt(uint64(param.GenesisBCHAlreadyMintedInMainChain)).Bytes32(),
			LastCovenantAddr:      [20]byte{},
			CurrCovenantAddr:      common.HexToAddress(param.GenesisCovenantAddress),
		}
		SaveCCContext(ctx, context)
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
	context.UTXOAlreadyHandled = false
	logs = append(logs, c.handleOperatorOrMonitorSetChanged(ctx, context)...)
	SaveCCContext(ctx, *context)
	c.StartUTXOCollect <- types.UTXOCollectParam{
		BeginHeight: int64(context.LastRescannedHeight),
		EndHeight:   int64(context.RescanHeight),
	}
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
	if context.UTXOAlreadyHandled {
		outData = []byte(UTXOAlreadyHandled.Error())
		return
	}
	newLogs, err := c.handleTransferInfos(ctx, context)
	if err != nil {
		outData = []byte(err.Error())
		return
	}
	logs = append(logs, newLogs...)
	SaveCCContext(ctx, *context)
	status = StatusSuccess
	return
}

func (c *CcContractExecutor) handleTransferInfos(ctx *mevmtypes.Context, context *types.CCContext) (logs []mevmtypes.EvmLog, err error) {
	context.UTXOAlreadyHandled = true
	for _, info := range <-c.UTXOCollectDone {
		switch info.Type {
		case types.TransferType:
			transferLogs, err := handleTransferTypeUTXO(ctx, context, info)
			if err != nil {
				return nil, err
			}
			logs = append(logs, transferLogs...)
		case types.ConvertType:
			convertLogs, err := handleConvertTypeUTXO(ctx, context, info)
			if err != nil {
				return nil, err
			}
			logs = append(logs, convertLogs...)
		case types.RedeemOrLostAndFoundType:
			logs = append(logs, handleRedeemOrLostAndFoundTypeUTXO(ctx, context, info)...)
		default:
		}
	}
	return logs, nil
}

func handleTransferTypeUTXO(ctx *mevmtypes.Context, context *types.CCContext, info *types.CCTransferInfo) ([]mevmtypes.EvmLog, error) {
	r := types.UTXORecord{
		Txid:   info.UTXO.TxID,
		Index:  info.UTXO.Index,
		Amount: info.UTXO.Amount,
	}
	if info.CovenantAddress == context.LastCovenantAddr {
		r.OwnerOfLost = info.Receiver
		SaveUTXORecord(ctx, r)
		return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}, nil
	}
	amount := uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:])
	maxAmount := uint256.NewInt(0).Mul(uint256.NewInt(MaxCCAmount), uint256.NewInt(E18))
	minAmount := uint256.NewInt(0).Mul(uint256.NewInt(MinCCAmount), uint256.NewInt(E18))
	if amount.Gt(maxAmount) {
		r.OwnerOfLost = info.Receiver
		SaveUTXORecord(ctx, r)
		return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}, nil
	} else if amount.Lt(minAmount) {
		blackHoleBalance := ebp.GetBlackHoleBalance(ctx)
		totalBurntOnMain := uint256.NewInt(0).SetBytes32(context.TotalBurntOnMainChain[:])
		if !blackHoleBalance.Gt(totalBurntOnMain) {
			return nil, PendingBurningNotEnough
		}
		pendingBurning := blackHoleBalance.Sub(blackHoleBalance, totalBurntOnMain)
		minPendingBurningLeft := uint256.NewInt(0).Mul(uint256.NewInt(MinPendingBurningLeft), uint256.NewInt(E18))
		if pendingBurning.Lt(uint256.NewInt(0).Add(minPendingBurningLeft, amount)) {
			r.OwnerOfLost = info.Receiver
			SaveUTXORecord(ctx, r)
			return []mevmtypes.EvmLog{buildNewLostAndFound(r.Txid, r.Index, r.CovenantAddr)}, nil
		}
		// add total burnt on main chain here, not waiting tx minted on main chain
		totalBurnt := uint256.NewInt(0).SetBytes32(context.TotalBurntOnMainChain[:])
		totalBurnt.Add(totalBurnt, amount)
		context.TotalBurntOnMainChain = totalBurnt.Bytes32()

		r.IsRedeemed = true
		r.RedeemTarget = BurnAddressMainChain
		SaveUTXORecord(ctx, r)
		err := transferBch(ctx, CCContractAddress, info.Receiver, amount)
		if err != nil {
			panic(err)
		}
		return []mevmtypes.EvmLog{buildRedeemLog(r.Txid, r.Index, context.CurrCovenantAddr, types.FromBurnRedeem)}, nil
	}
	SaveUTXORecord(ctx, r)
	err := transferBch(ctx, CCContractAddress, info.Receiver, amount)
	if err != nil {
		panic(err)
	}
	return []mevmtypes.EvmLog{buildNewRedeemable(r.Txid, r.Index, context.CurrCovenantAddr)}, nil
}

func handleConvertTypeUTXO(ctx *mevmtypes.Context, context *types.CCContext, info *types.CCTransferInfo) ([]mevmtypes.EvmLog, error) {
	r := LoadUTXORecord(ctx, info.PrevUTXO.TxID, info.PrevUTXO.Index)
	if r == nil {
		return nil, nil
	}
	originAmount := uint256.NewInt(0).SetBytes(r.Amount[:])
	newAmount := uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:])
	if originAmount.Lt(newAmount) || originAmount.Eq(newAmount) {
		return nil, nil
	}
	//deduct gas fee used for utxo convert
	blockHoleBalance := ebp.GetBlackHoleBalance(ctx)
	totalBurntOnMainChain := uint256.NewInt(0).SetBytes32(context.TotalBurntOnMainChain[:])
	if !blockHoleBalance.Gt(totalBurntOnMainChain) {
		return nil, PendingBurningNotEnough
	}
	pendingBurning := blockHoleBalance.Sub(blockHoleBalance, totalBurntOnMainChain)
	gasFee := originAmount.Sub(originAmount, newAmount)
	if pendingBurning.Lt(gasFee) {
		return nil, PendingBurningNotEnough
	}
	// add gas fee on totalBurntOnMainChain, maybe should record this on specific field
	totalBurntOnMainChain.Add(totalBurntOnMainChain, gasFee)
	context.TotalBurntOnMainChain = totalBurntOnMainChain.Bytes32()
	newR := types.UTXORecord{
		Txid:   info.UTXO.TxID,
		Index:  info.UTXO.Index,
		Amount: newAmount.Bytes32(),
	}
	SaveUTXORecord(ctx, newR)
	DeleteUTXORecord(ctx, info.PrevUTXO.TxID, info.PrevUTXO.Index)
	return []mevmtypes.EvmLog{buildConvertLog(r.Txid, r.Index, newR.CovenantAddr, newR.Txid, newR.Index)}, nil
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
	context.LastCovenantAddr = context.CurrCovenantAddr
	context.CurrCovenantAddr = newAddress
	logs = append(logs, buildChangeAddrLog(context.LastCovenantAddr, context.CurrCovenantAddr))
	return
}

func checkAndUpdateRedeemTX(ctx *mevmtypes.Context, block *mevmtypes.BlockInfo, txid [32]byte, index uint32, amount *uint256.Int, targetAddress [20]byte) (*mevmtypes.EvmLog, error) {
	r := LoadUTXORecord(ctx, txid, index)
	if r == nil {
		return nil, UTXONotExist
	}
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

func CollectOpList(mdbBlock *modbtypes.Block) modbtypes.OpListsForCcUtxo {
	opList := modbtypes.OpListsForCcUtxo{}
	events := mevmtypes.BlockToChainEvent(mdbBlock)
	for _, l := range events.Logs {
		if l.Address != CCContractAddress {
			continue
		}
		var eventHash [32]byte
		copy(eventHash[:], l.Topics[0].Bytes())
		switch eventHash {
		case HashOfEventRedeem:
			redeemOp := modbtypes.RedeemOp{
				CovenantAddr: [20]byte{},
				SourceType:   0,
			}
			copy(redeemOp.UtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(redeemOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			redeemOp.CovenantAddr = common.BytesToAddress(l.Topics[2][:])
			redeemOp.SourceType = byte(uint256.NewInt(0).SetBytes32(l.Data).Uint64())
			opList.RedeemOps = append(opList.RedeemOps, redeemOp)
		case HashOfEventNewRedeemable:
			newRedeemableOp := modbtypes.NewRedeemableOp{
				UtxoId:       [36]byte{},
				CovenantAddr: [20]byte{},
			}
			copy(newRedeemableOp.UtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(newRedeemableOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			newRedeemableOp.CovenantAddr = common.BytesToAddress(l.Topics[2][:])
			opList.NewRedeemableOps = append(opList.NewRedeemableOps, newRedeemableOp)
		case HashOfEventNewLostAndFound:
			newLostAndFoundOp := modbtypes.NewLostAndFoundOp{
				UtxoId:       [36]byte{},
				CovenantAddr: [20]byte{},
			}
			copy(newLostAndFoundOp.UtxoId[:32], l.Topics[1][:])
			binary.BigEndian.PutUint32(newLostAndFoundOp.UtxoId[32:], uint32(uint256.NewInt(0).SetBytes32(l.Topics[2][:]).Uint64()))
			newLostAndFoundOp.CovenantAddr = common.BytesToAddress(l.Topics[2][:])
			opList.NewLostAndFoundOps = append(opList.NewLostAndFoundOps, newLostAndFoundOp)
		case HashOfEventConvert:
		case HashOfEventChangeAddr:
			newChangeAddrOp := modbtypes.ChangeAddrOp{
				PrevUtxoId:      [36]byte{},
				UtxoId:          [36]byte{},
				OldCovenantAddr: [20]byte{},
				NewCovenantAddr: [20]byte{},
			}
			opList.ChangeAddrOps = append(opList.ChangeAddrOps, newChangeAddrOp)
		case HashOfEventDeleted:
			newDeleteOp := modbtypes.DeletedOp{
				UtxoId:       [36]byte{},
				CovenantAddr: [20]byte{},
				SourceType:   0,
			}
			opList.DeletedOps = append(opList.DeletedOps, newDeleteOp)
		default:
			continue
		}
	}
	return opList
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

type MockVoteContract struct {
	IsM        bool
	IsChanged  bool
	NewAddress common.Address
}

func (m *MockVoteContract) IsMonitor(ctx *mevmtypes.Context, address common.Address) bool {
	return m.IsM
}

func (m *MockVoteContract) IsOperatorOrMonitorChanged(ctx *mevmtypes.Context, ccCtx *types.CCContext) bool {
	return m.IsChanged
}

func (m *MockVoteContract) GetNewCovenantAddress(ctx *mevmtypes.Context) common.Address {
	return m.NewAddress
}

var _ IVoteContract = &MockVoteContract{}
