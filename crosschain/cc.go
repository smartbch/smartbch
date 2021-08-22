package crosschain

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingevm/ebp"
	mevmtypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/crosschain/types"
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
	/*interface CC {
	    function transferBCHToMainnet(bytes utxo) external;
	    function burnBCH(bytes utxo) external;

	    event TransferToMainnet(bytes32 indexed mainnetTxId, bytes4 indexed vOutIndex, address indexed from, uint256 value);
	    event Burn(bytes32 indexed mainnetTxId, bytes4 indexed vOutIndex, uint256 value);
	}*/
	SelectorTransferBchToMainnet [4]byte = [4]byte{0xa0, 0x4f, 0x3b, 0x01}
	SelectorBurnBCH              [4]byte = [4]byte{0x18, 0x92, 0xa8, 0xb3}

	HashOfEventTransferToBch [32]byte = common.HexToHash("0x4a9f09be1e2df144675144ec10cb5fe6c05504a84262275b62028189c1d410c1")
	HashOfEventBurn          [32]byte = common.HexToHash("0xeae299b236fc8161793d044c8260b3dc7f8c20b5b3b577eb7f075e4a9c3bf48d")

	GasOfCCOp uint64 = 400_000

	InvalidCallData   = errors.New("invalid call data")
	InvalidSelector   = errors.New("invalid selector")
	BchAmountNotMatch = errors.New("value not match bch amount in utxo")
	BalanceNotEnough  = errors.New("balance is not enough")
	BurnToMuch        = errors.New("burn more bch than reality")

	SlotCCInfo          string = strings.Repeat(string([]byte{0}), 32)
	SlotBCHAlreadyBurnt string = strings.Repeat(string([]byte{0}), 31) + string([]byte{1})
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
	case SelectorTransferBchToMainnet:
		return transferBchToMainnet(ctx, tx)
	case SelectorBurnBCH:
		return burnBch(ctx, tx)
	default:
		status = StatusFailed
		outData = []byte(InvalidSelector.Error())
		return
	}
}

// this functions is called when other contract calls
func (_ *CcContractExecutor) RequiredGas(_ []byte) uint64 {
	return GasOfCCOp
}

func (_ *CcContractExecutor) Run(_ []byte) ([]byte, error) {
	return nil, nil
}

// function transferBCHToMainnet(bytes utxo) external;
func transferBchToMainnet(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed //default status is failed
	gasUsed = GasOfCCOp
	callData := tx.Data[4:]
	if len(callData) < 32+32+64 /*[36]byte abi encode length*/ {
		outData = []byte(InvalidCallData.Error())
		return
	}
	// First argument: utxo
	var utxo [36]byte
	copy(utxo[:], callData[64:64+36])

	value := uint256.NewInt(0).SetBytes32(tx.Value[:])
	err := consumeUTXO(ctx, utxo, value)
	if err != nil {
		outData = []byte(err.Error())
		return
	}
	status, outData = transferBchFromTx(ctx, tx)
	if status == StatusSuccess {
		// emit event, convert BCH from 18bit to 8bit
		logs = append(logs, buildTransferToMainnetEvmLog(utxo, tx.From, value))
	}
	return
}

func buildTransferToMainnetEvmLog(utxo [36]byte, from common.Address, value *uint256.Int) mevmtypes.EvmLog {
	evmLog := mevmtypes.EvmLog{
		Address: CCContractAddress,
		Topics:  make([]common.Hash, 0, 4),
	}

	evmLog.Topics = append(evmLog.Topics, HashOfEventTransferToBch)
	txId := common.Hash{}
	txId.SetBytes(utxo[:32])
	evmLog.Topics = append(evmLog.Topics, txId)
	vOutIndex := common.Hash{}
	vOutIndex.SetBytes(utxo[32:])
	evmLog.Topics = append(evmLog.Topics, vOutIndex)
	sender := common.Hash{}
	sender.SetBytes(from[:])
	evmLog.Topics = append(evmLog.Topics, sender)

	data := value.Bytes32()
	evmLog.Data = append(evmLog.Data, data[:]...)
	return evmLog
}

// function burnBCH(utxo bytes, amount uint256) external
func burnBch(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfCCOp
	callData := tx.Data[4:]
	if len(callData) < 32+32+64 /*[36]byte*/ {
		outData = []byte(InvalidCallData.Error())
		return
	}
	var utxo [36]byte
	copy(utxo[:], callData[64:64+36])

	value := LoadUTXO(ctx, utxo)
	err := UpdateBchBurnt(ctx, value)
	if err != nil {
		outData = []byte(err.Error())
		return
	}
	deleteUTXO(ctx, utxo)
	logs = append(logs, buildBurnEvmLog(utxo, value))
	status = StatusSuccess
	return
}

func buildBurnEvmLog(utxo [36]byte, value *uint256.Int) mevmtypes.EvmLog {
	evmLog := mevmtypes.EvmLog{
		Address: CCContractAddress,
		Topics:  make([]common.Hash, 0, 4),
	}

	evmLog.Topics = append(evmLog.Topics, HashOfEventBurn)
	txId := common.Hash{}
	txId.SetBytes(utxo[:32])
	evmLog.Topics = append(evmLog.Topics, txId)
	vOutIndex := common.Hash{}
	vOutIndex.SetBytes(utxo[32:])
	evmLog.Topics = append(evmLog.Topics, vOutIndex)

	data := value.Bytes32()
	evmLog.Data = append(evmLog.Data, data[:]...)
	return evmLog
}

func consumeUTXO(ctx *mevmtypes.Context, utxo [36]byte, value *uint256.Int) error {
	originAmount := LoadUTXO(ctx, utxo)
	if originAmount.Cmp(value) != 0 {
		return BchAmountNotMatch
	}
	deleteUTXO(ctx, utxo)
	return nil
}

func LoadUTXO(ctx *mevmtypes.Context, utxo [36]byte) *uint256.Int {
	var bz []byte
	key := sha256.Sum256(utxo[:])
	bz = ctx.GetStorageAt(ccContractSequence, string(key[:]))
	return uint256.NewInt(0).SetBytes(bz)
}

func SaveUTXO(ctx *mevmtypes.Context, utxo [36]byte, amount *uint256.Int) {
	key := sha256.Sum256(utxo[:])
	ctx.SetStorageAt(ccContractSequence, string(key[:]), amount.Bytes())
}

func deleteUTXO(ctx *mevmtypes.Context, utxo [36]byte) {
	key := sha256.Sum256(utxo[:])
	ctx.DeleteStorageAt(ccContractSequence, string(key[:]))
}

func LoadBchMainnetBurnt(ctx *mevmtypes.Context) *uint256.Int {
	var bz = ctx.GetStorageAt(ccContractSequence, SlotBCHAlreadyBurnt)
	return uint256.NewInt(0).SetBytes(bz)
}

func SaveBchMainnetBurnt(ctx *mevmtypes.Context, amount *uint256.Int) {
	ctx.SetStorageAt(ccContractSequence, SlotBCHAlreadyBurnt, amount.Bytes())
}

func UpdateBchBurnt(ctx *mevmtypes.Context, amount *uint256.Int) error {
	mainnetBurnt := LoadBchMainnetBurnt(ctx)
	burnt := ebp.GetBlackHoleBalance(ctx)
	if burnt.Cmp(mainnetBurnt.Add(mainnetBurnt, amount)) < 0 {
		return BurnToMuch
	}
	SaveBchMainnetBurnt(ctx, mainnetBurnt)
	return nil
}

func transferBchFromTx(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, outData []byte) {
	status = StatusFailed
	err := transferBch(ctx, tx.From, CCContractAddress, uint256.NewInt(0).SetBytes(tx.Value[:]))
	if err != nil {
		outData = []byte(err.Error())
		return
	}
	status = StatusSuccess
	return
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

func SwitchCCEpoch(ctx *mevmtypes.Context, epoch *types.CCEpoch) {
	for _, info := range epoch.TransferInfos {
		value := uint256.NewInt(0).SetUint64(info.Amount)
		fmt.Printf("switch epoch: info.pubkey:%v, info.amount:%d, info.utxo:%s\n",
			info.SenderPubkey, info.Amount, info.UTXO)
		SaveUTXO(ctx, info.UTXO, value)
		var sender common.Address
		sender.SetBytes(secp256k1.PubKey(info.SenderPubkey[:]).Address().Bytes())
		err := transferBch(ctx, CCContractAddress, sender, value)
		if err != nil {
			//log it, never in here if mainnet is honest
			fmt.Println(err.Error())
		}
	}
	ccInfo := LoadCCInfo(ctx)
	ccInfo.CurrEpochNum++
	SaveCCInfo(ctx, ccInfo)
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
