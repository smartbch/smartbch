package crosschain

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/libs/log"

	mevmtypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/param"
)

const (
	StatusSuccess      int    = 0
	StatusFailed       int    = 1
	ccContractSequence uint64 = math.MaxUint64 - 4 /*uint64(-4)*/
)

var (
	//contract address, 9999
	//todo: transfer remain BCH to this address before working
	CCContractAddress [20]byte = [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x27, 0x09}
	/*------selector------*/
	SelectorRedeem [4]byte = [4]byte{0x18, 0x92, 0xa8, 0xb3} // todo: modify it

	HashOfEventTransferToBch [32]byte = common.HexToHash("0x4a9f09be1e2df144675144ec10cb5fe6c05504a84262275b62028189c1d410c1")
	HashOfEventBurn          [32]byte = common.HexToHash("0xeae299b236fc8161793d044c8260b3dc7f8c20b5b3b577eb7f075e4a9c3bf48d")

	GasOfCCOp uint64 = 400_000

	InvalidCallData   = errors.New("invalid call data")
	InvalidSelector   = errors.New("invalid selector")
	BchAmountNotMatch = errors.New("value not match bch amount in utxo")
	BalanceNotEnough  = errors.New("balance is not enough")
)

var (
	SlotCCInfo             string = strings.Repeat(string([]byte{0}), 32)
	SlotScriptInfo         string = strings.Repeat(string([]byte{0}), 31) + string([]byte{2})
	SlotSwitchInfo         string = strings.Repeat(string([]byte{0}), 31) + string([]byte{3})
	SlotUnsignedConvertTxs string = strings.Repeat(string([]byte{0}), 31) + string([]byte{4})
)

type CcContractExecutor struct {
	logger log.Logger
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

func (s *CcContractExecutor) Execute(ctx *mevmtypes.Context, currBlock *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
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

// function redeem(txid bytes32, index uint32, amount uint256) external
func redeem(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	callData := tx.Data[4:]
	if len(callData) < 32+32+32 {
		outData = []byte(InvalidCallData.Error())
		return
	}
	var utxo types.UTXO
	copy(utxo.TxID[:], callData[:32])
	index := uint256.NewInt(0).SetBytes32(callData[32:64])
	amount := uint256.NewInt(0).SetBytes32(callData[64:96])
	utxo.Index = uint32(index.Uint64())
	utxo.Amount = int64(amount.Uint64())
	if !checkAndDeleteUtxo(ctx, utxo) {
		status = StatusFailed
		return
	}
	err := transferBch(ctx, tx.From, CCContractAddress, uint256.NewInt(0).SetBytes32(tx.Value[:]))
	if err != nil {
		outData = []byte(err.Error())
		return
	}
	serializedTx, _ := buildUnsignedTx(utxo, nil, [20]byte{})
	//build log including serializedTx
	fmt.Printf(serializedTx)
	status = StatusSuccess
	return
}

func checkAndDeleteUtxo(ctx *mevmtypes.Context, utxo types.UTXO) bool {
	info := LoaUTXOInfos(ctx, types.TransferType, types.AllUTXO)
	index := -1
	for i, u := range info.UtxoSet {
		if utxo.TxID == u.TxID && utxo.Index == u.Index && utxo.Amount == u.Amount {
			index = i
			break
		}
	}
	var newUTXOes []types.UTXO
	if index == len(info.UtxoSet)-1 {
		newUTXOes = info.UtxoSet[:len(info.UtxoSet)]
	} else if index != -1 {
		newUTXOes = append(info.UtxoSet[:index], info.UtxoSet[index+1:]...)
	} else {
		newUTXOes = info.UtxoSet
		return false
	}
	info.UtxoSet = newUTXOes
	SaveUTXOInfos(ctx, types.TransferType, types.AllUTXO, info)
	return true
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

func LoadCCInfo(ctx *mevmtypes.Context) (info types.CCInfo) {
	bz := ctx.GetStorageAt(ccContractSequence, SlotCCInfo)
	if bz == nil {
		return types.CCInfo{}
	}
	_, err := info.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	return
}

func SaveCCInfo(ctx *mevmtypes.Context, info types.CCInfo) {
	bz, err := info.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(ccContractSequence, SlotCCInfo, bz)
}

// get a slot number to store an epoch's validators
func getSlotForCCEpoch(epochNum int64) string {
	var buf [32]byte
	buf[23] = 1
	binary.BigEndian.PutUint64(buf[24:], uint64(epochNum))
	return string(buf[:])
}

func SaveCCEpoch(ctx *mevmtypes.Context, epochNum int64, epoch *types.CCEpoch) {
	bz, err := epoch.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(ccContractSequence, getSlotForCCEpoch(epochNum), bz)
}

func LoadCCEpoch(ctx *mevmtypes.Context, epochNum int64) (epoch types.CCEpoch, ok bool) {
	bz := ctx.GetStorageAt(ccContractSequence, getSlotForCCEpoch(epochNum))
	if bz == nil {
		return
	}
	_, err := epoch.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	ok = true
	return
}

var fixedMainnetFee = int64(10)

// build tx

func buildUnsignedTx(utxo types.UTXO, redeemScript []byte, p2shHash [20]byte) (string, error) {
	tx := wire.NewMsgTx(wire.TxVersion)
	// 1. build tx output
	pkScript, err := buildP2shPubkeyScript(p2shHash[:])
	if err != nil {
		panic(err)
	}
	tx.AddTxOut(wire.NewTxOut(utxo.Amount-fixedMainnetFee, pkScript))
	// 2. build tx input
	in := wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  utxo.TxID,
			Index: utxo.Index,
		},
		SignatureScript: redeemScript, //store redeem script here
		Sequence:        0xffffffff,
	}
	tx.AddTxIn(&in)
	return txSerialize2Hex(tx), nil
}

func buildP2shPubkeyScript(scriptHash []byte) ([]byte, error) {
	return txscript.NewScriptBuilder().AddOp(txscript.OP_HASH160).AddData(scriptHash).AddOp(txscript.OP_EQUAL).Script()
}

func buildMultiSigRedeemScript(pubkeys []*bchutil.AddressPubKey, n int) ([]byte, error) {
	return txscript.MultiSigScript(pubkeys, n)
}

func txSerialize2Hex(tx *wire.MsgTx) string {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf.Bytes())
}

// storage related
func AddUTXOInfos(ctx *mevmtypes.Context, param types.UTXOParam, utxo types.UTXO) {
	infos := LoaUTXOInfos(ctx, utxo.Type, param)
	infos.UtxoSet = append(infos.UtxoSet, utxo)
	SaveUTXOInfos(ctx, utxo.Type, param, infos)
}

func SubUTXOInfos(ctx *mevmtypes.Context, param types.UTXOParam, utxo types.UTXO) {
	infos := LoaUTXOInfos(ctx, utxo.Type, param)
	newSet := make([]types.UTXO, 0, len(infos.UtxoSet)-1)
	m := make(map[[32]byte]*types.UTXO)
	for _, info := range infos.UtxoSet {
		m[info.TxID] = &info
	}
	if u := m[utxo.TxID]; u != nil {
		if u.Index == utxo.Index {
			delete(m, u.TxID)
		}
	}
	for _, u := range m {
		newSet = append(newSet, *u)
	}
	infos.UtxoSet = newSet
	SaveUTXOInfos(ctx, utxo.Type, param, infos)
}

func LoaUTXOInfos(ctx *mevmtypes.Context, utxoType types.UTXOType, param types.UTXOParam) types.UTXOInfos {
	return types.UTXOInfos{}
}

func SaveUTXOInfos(ctx *mevmtypes.Context, utxoType types.UTXOType, param types.UTXOParam, infos types.UTXOInfos) {
}

func LoaSwitchInfo(ctx *mevmtypes.Context) bool {
	var sw bool
	bz := ctx.GetStorageAt(ccContractSequence, SlotSwitchInfo)
	if bz == nil {
		return false
	}
	_ = json.Unmarshal(bz, &sw)
	return sw
}

func SaveSwitchInfo(ctx *mevmtypes.Context, sw bool) {
	bz, err := json.Marshal(sw)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(ccContractSequence, SlotSwitchInfo, bz)
}

func LoadScriptInfo(ctx *mevmtypes.Context) (info types.ScriptInfo) {
	bz := ctx.GetStorageAt(ccContractSequence, SlotScriptInfo)
	if bz == nil {
		return types.ScriptInfo{}
	}
	_, err := info.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	return
}

func SaveScriptInfo(ctx *mevmtypes.Context, info types.ScriptInfo) {
	bz, err := info.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(ccContractSequence, SlotScriptInfo, bz)
}

func SwitchCCEpoch(ctx *mevmtypes.Context, epoch *types.CCEpoch) {
	ccInfo := LoadCCInfo(ctx)
	//when open epoch speedup, staking epoch is in strict accordance with the mainnet height,
	//but cc epoch which already switched in recent staking epoch can be fetch repeat,
	//so filter these cc epochs here
	if epoch.StartHeight < ccInfo.GenesisMainnetBlockHeight+ccInfo.CurrEpochNum*param.BlocksInCCEpoch {
		fmt.Printf("skip crosschain epoch, its start height:%d as of ccepoch speed up\n", epoch.StartHeight)
		return
	}
	for _, info := range epoch.TransferInfos {
		value := uint256.NewInt(0).SetUint64(uint64(info.UTXO.Amount))
		fmt.Printf("switch epoch: info.receiver:%v, info.amount:%d, info.utxo:%v\n",
			info.Receiver, info.UTXO.Amount, info.UTXO)
		err := transferBch(ctx, CCContractAddress, info.Receiver, value)
		if err != nil {
			panic(err.Error())
		}
	}
	UpdateUtxoSet(ctx, epoch.TransferInfos)
	ccInfo.CurrEpochNum++
	SaveCCInfo(ctx, ccInfo)
	epoch.Number = ccInfo.CurrEpochNum
	SaveCCEpoch(ctx, epoch.Number, epoch)
}

func UpdateUtxoSet(ctx *mevmtypes.Context, infos []*types.CCTransferInfo) {
	for _, info := range infos {
		switch info.UTXO.Type {
		//
		case types.TransferType:
			AddUTXOInfos(ctx, types.AllUTXO, info.UTXO)
		case types.ConvertType:
			SubUTXOInfos(ctx, types.AllUTXO, info.PrevUTXO)
			AddUTXOInfos(ctx, types.AllUTXO, info.UTXO)
		case types.MonitorCancelRedeemType:
		case types.MonitorCancelConvertType:
		default:
		}
	}
	if isOperatorChanged() {
		BuildSaveUTXOConvertUnsignedTxs(ctx)
	}
}

func isOperatorChanged() bool {
	return false
}

func BuildSaveUTXOConvertUnsignedTxs(ctx *mevmtypes.Context) {
	all := LoaUTXOInfos(ctx, types.TransferType, types.AllUTXO)
	allTxs := make([]string, 0, len(all.UtxoSet))
	for i, utxo := range all.UtxoSet {
		s, _ := buildUnsignedTx(utxo, nil, [20]byte{})
		allTxs[i] = s
	}
	SaveUTXOConvertUnsignedTxs(ctx, allTxs)
}

func SaveUTXOConvertUnsignedTxs(ctx *mevmtypes.Context, txs []string) {
	bz, err := json.Marshal(txs)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(ccContractSequence, SlotUnsignedConvertTxs, bz)
}
