package crosschain

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/smartbch/moeingevm/ebp"
	mevmtypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/crosschain/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	"math"
	"strings"
)

const (
	StatusSuccess      int    = 0
	StatusFailed       int    = 1
	ccContractSequence uint64 = math.MaxUint64 - 4 /*uint64(-4)*/
)

var (
	//contract address, 9999
	//todo: transfer remain BCH to this address before working
	ccContractAddress [20]byte = [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x27, 0x09}
	/*------selector------*/
	/*interface CC {
	    function transferBCHToMainnet(utxo bytes, amount uint256) external;
	    function burnBCH(utxo bytes, amount uint256) external;

	    event TransferToMainnet(bytes32 indexed mainnetTxId, bytes4 indexed vOutIndex, address indexed from, uint256 value);
	    event Burn(bytes32 indexed mainnetTxId, bytes4 indexed vOutIndex, uint256 value);
	}*/
	SelectorTransferBchToMainnet [4]byte = [4]byte{0x24, 0xd1, 0xed, 0x5d}
	SelectorBurnBCH              [4]byte = [4]byte{0xa4, 0x87, 0x4d, 0x77}

	HashOfEventTransferToBch [32]byte = [32]byte{}
	HashOfEventBurn          [32]byte = [32]byte{}

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
	ccAcc := ctx.GetAccount(ccContractAddress)
	if ccAcc == nil { // only executed at genesis
		ccAcc = mevmtypes.ZeroAccountInfo()
		ccAcc.UpdateSequence(ccContractSequence)
		ctx.SetAccount(ccContractAddress, ccAcc)
	}
}

func (_ *CcContractExecutor) IsSystemContract(addr common.Address) bool {
	return bytes.Equal(addr[:], ccContractAddress[:])
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
	if len(callData) < 96 /*[36]byte abi encode length*/ {
		outData = []byte(InvalidCallData.Error())
		return
	}
	// First argument: utxo
	var utxo [36]byte
	copy(utxo[:], callData[32:68])

	value := uint256.NewInt().SetBytes32(tx.Value[:])
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
		Address: ccContractAddress,
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
	if len(callData) < 96+32 /*[36]byte* + uint256*/ {
		outData = []byte(InvalidCallData.Error())
		return
	}
	var utxo [36]byte
	copy(utxo[:], callData[32:68])

	value := uint256.NewInt().SetBytes32(tx.Value[:])
	err := consumeUTXO(ctx, utxo, value)
	if err != nil {
		outData = []byte(err.Error())
		return
	}
	err = UpdateBchBurnt(ctx, value)
	if err != nil {
		outData = []byte(err.Error())
		return
	}
	logs = append(logs, buildBurnEvmLog(utxo, value))
	status = StatusSuccess
	return
}

func buildBurnEvmLog(utxo [36]byte, value *uint256.Int) mevmtypes.EvmLog {
	evmLog := mevmtypes.EvmLog{
		Address: ccContractAddress,
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
	SaveUTXO(ctx, utxo, uint256.NewInt())
	return nil
}

func LoadUTXO(ctx *mevmtypes.Context, utxo [36]byte) *uint256.Int {
	var bz []byte
	key := sha256.Sum256(utxo[:])
	bz = ctx.GetStorageAt(ccContractSequence, string(key[:]))
	return uint256.NewInt().SetBytes(bz)
}

func SaveUTXO(ctx *mevmtypes.Context, utxo [36]byte, amount *uint256.Int) {
	key := sha256.Sum256(utxo[:])
	ctx.SetStorageAt(ccContractSequence, string(key[:]), amount.Bytes())
}

func LoadBchMainnetBurnt(ctx *mevmtypes.Context) *uint256.Int {
	var bz []byte
	bz = ctx.GetStorageAt(ccContractSequence, SlotBCHAlreadyBurnt)
	return uint256.NewInt().SetBytes(bz)
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
	err := transferBch(ctx, tx.From, ccContractAddress, uint256.NewInt().SetBytes(tx.Value[:]))
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
		receiverAccBalance := receiverAcc.Balance()
		receiverAccBalance.Add(receiverAccBalance, value)
		receiverAcc.UpdateBalance(receiverAccBalance)
		ctx.SetAccount(receiver, receiverAcc)
	}
	return nil
}

func SwitchCCEpoch(ctx *mevmtypes.Context, epoch *types.CCEpoch) {
	for _, info := range epoch.TransferInfos {
		value := uint256.NewInt().SetUint64(info.Amount)
		SaveUTXO(ctx, info.UTXO, value)
		var sender common.Address
		sender.SetBytes(ed25519.PubKey(info.SenderPubkey[:]).Address().Bytes())
		err := transferBch(ctx, ccContractAddress, sender, value)
		if err != nil {
			//log it, never in here if mainnet is honest
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
