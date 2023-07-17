package staking

import (
	"bytes"
	"container/heap"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingevm/ebp"
	mevmtypes "github.com/smartbch/moeingevm/types"

	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/staking/types"
)

const (
	SumVotingPowerGasPerByte uint64 = 25
	SumVotingPowerBaseGas    uint64 = 10000

	StatusSuccess int = 0 // because EVMC_SUCCESS = 0,
	StatusFailed  int = 1 // because EVMC_FAILURE = 1,

	StakingContractSequence uint64 = math.MaxUint64 - 2 /*uint64(-3)*/
	Uint64_1e18             uint64 = 1000_000_000_000_000_000
)

var (
	//contract address, 10000
	StakingContractAddress = [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x27, 0x10}
	/*------selector------*/
	/*interface Staking {
		// following methods can only be called by EOA
		//0x24d1ed5d
		function createValidator(address rewardTo, bytes32 introduction, bytes32 pubkey) external;
		//0x9dc159b6
		function editValidator(address rewardTo, bytes32 introduction) external;
		//0xa4874d77
		function retire() external;
		//0xf2016e8e
		function increaseMinGasPrice() external;
		//0x696e6ad2
		function decreaseMinGasPrice() external;
		//30326c17
		function proposal(uint target) external;
		//0121b93f
		function vote(uint target) external;
		//373058b8
		function executeProposal() external;
		//8d337b81
		function getVote(address validator) external view returns (uint)

		// sumVotingPower can only be called by other smart contracts
		//9ce06909
		function sumVotingPower(address[] calldata addrList) external override returns (uint summedPower, uint totalPower)
	}*/
	SelectorCreateValidator     = [4]byte{0x24, 0xd1, 0xed, 0x5d}
	SelectorEditValidator       = [4]byte{0x9d, 0xc1, 0x59, 0xb6}
	SelectorRetire              = [4]byte{0xa4, 0x87, 0x4d, 0x77}
	SelectorIncreaseMinGasPrice = [4]byte{0xf2, 0x01, 0x6e, 0x8e}
	SelectorDecreaseMinGasPrice = [4]byte{0x69, 0x6e, 0x6a, 0xd2}
	SelectorProposal            = [4]byte{0x30, 0x32, 0x6c, 0x17}
	SelectorVote                = [4]byte{0x01, 0x21, 0xb9, 0x3f}
	SelectorExecuteProposal     = [4]byte{0x37, 0x30, 0x58, 0xb8}
	SelectorGetVote             = [4]byte{0x8d, 0x33, 0x7b, 0x81}
	SelectorSumVotingPower      = [4]byte{0x9c, 0xe0, 0x69, 0x09}

	//slot
	SlotStakingInfo               = strings.Repeat(string([]byte{0}), 32)
	SlotAllBurnt                  = strings.Repeat(string([]byte{0}), 31) + string([]byte{1})
	SlotMinGasPriceInBlock        = strings.Repeat(string([]byte{0}), 31) + string([]byte{2})
	SlotLastMinGasPrice           = strings.Repeat(string([]byte{0}), 31) + string([]byte{3})
	SlotMinGasPriceProposalTarget = strings.Repeat(string([]byte{0}), 31) + string([]byte{4})
	SlotVoters                    = strings.Repeat(string([]byte{0}), 31) + string([]byte{5})
	SlotOnlineInfo                = strings.Repeat(string([]byte{0}), 31) + string([]byte{6})
	SlotWatchInfo                 = strings.Repeat(string([]byte{0}), 31) + string([]byte{7})

	// slot in hex
	SlotMinGasPriceHex = hex.EncodeToString([]byte(SlotLastMinGasPrice))

	votingSlotHashPrefix = [4]byte{'v', 'o', 't', 'e'}
)

var (
	/*------param------*/
	//staking
	InitialStakingAmount                 = uint256.NewInt(0)
	MinimumStakingAmount                 = uint256.NewInt(0)
	MinimumStakingAmountAfterStakingFork = uint256.NewInt(0).Mul(uint256.NewInt(32), uint256.NewInt(Uint64_1e18))

	GasOfValidatorOp   uint64 = 400_000
	GasOfMinGasPriceOp uint64 = 50_000

	//minGasPrice
	DefaultMinGasPrice          uint64 = param.DefaultMinGasPrice //10gwei
	MinGasPriceDeltaRateInBlock uint64 = 16
	MinGasPriceDeltaRate        uint64 = 5               //gas delta rate every proposal can change
	MinGasPriceUpperBound       uint64 = 500_000_000_000 //500gwei
	MinGasPriceLowerBoundOld    uint64 = 1_000_000_000   //1gwei
	MinGasPriceLowerBound       uint64 = 10_000_000      //0.01gwei
	DefaultProposalDuration     uint64 = 60 * 60 * 24    //24hour
)

var (
	/*------error info------*/
	InvalidCallData                   = errors.New("invalid call data")
	InvalidSelector                   = errors.New("invalid selector")
	BalanceNotEnough                  = errors.New("balance is not enough")
	NoSuchValidator                   = errors.New("no such validator")
	ValidatorInRetiring               = errors.New("validator in retiring")
	ValidatorNotActive                = errors.New("validator not active")
	OperatorNotValidator              = errors.New("minGasPrice operator not validator or its rewardTo")
	MinGasPriceTooBig                 = errors.New("minGasPrice bigger than allowed highest value")
	MinGasPriceTooSmall               = errors.New("minGasPrice smaller than allowed lowest value")
	InvalidArgument                   = errors.New("invalid argument")
	CreateValidatorCoinLtInitAmount   = errors.New("validator's staking coin less than init amount")
	MinGasPriceExceedBlockChangeDelta = errors.New("the amount of variation in minGasPrice exceeds the allowable range")
	StillInProposal                   = errors.New("still in proposal")
	NotInProposal                     = errors.New("not in proposal")
	TargetExceedChangeDelta           = errors.New("minGasPrice target exceeds the allowable range")
	ProposalHasFinished               = errors.New("proposal has finished")
	ProposalNotFinished               = errors.New("proposal not finished")
	ErrOutOfGas                       = errors.New("out of gas")
)

var readonlyStakingInfo *types.StakingInfo // for sumVotingPower

type StakingContractExecutor struct {
	logger log.Logger
}

func NewStakingContractExecutor(logger log.Logger) *StakingContractExecutor {
	return &StakingContractExecutor{
		logger: logger,
	}
}

var _ mevmtypes.SystemContractExecutor = &StakingContractExecutor{}

func (_ *StakingContractExecutor) Init(ctx *mevmtypes.Context) {
	stakingAcc := ctx.GetAccount(StakingContractAddress)
	if stakingAcc == nil { // only executed at genesis
		stakingAcc = mevmtypes.ZeroAccountInfo()
		stakingAcc.UpdateSequence(StakingContractSequence)
		ctx.SetAccount(StakingContractAddress, stakingAcc)
	}
	info := LoadStakingInfo(ctx)
	readonlyStakingInfo = &info
}

func (_ *StakingContractExecutor) IsSystemContract(addr common.Address) bool {
	return bytes.Equal(addr[:], StakingContractAddress[:])
}

// Staking functions which can be invoked by EOA through smart contract calls
// The extra gas fee is distributed to the validators, not refunded
func (s *StakingContractExecutor) Execute(ctx *mevmtypes.Context, currBlock *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	if len(tx.Data) < 4 {
		status = StatusFailed
		gasUsed = tx.Gas
		outData = []byte(InvalidCallData.Error())
		return
	}
	var selector [4]byte
	copy(selector[:], tx.Data[:4])
	switch selector {
	case SelectorCreateValidator:
		//createValidator(address rewardTo, bytes32 introduction, bytes32 pubkey)
		return createValidator(ctx, tx)
	case SelectorEditValidator:
		//editValidator(address rewardTo, bytes32 introduction)
		return editValidator(ctx, tx)
	case SelectorRetire:
		//retire()
		return retire(ctx, tx)
	case SelectorIncreaseMinGasPrice:
		//function increaseMinGasPrice() external;
		return handleMinGasPrice(ctx, tx, true, s.logger)
	case SelectorDecreaseMinGasPrice:
		//function decreaseMinGasPrice() external;
		return handleMinGasPrice(ctx, tx, false, s.logger)
	case SelectorProposal:
		if ctx.IsXHedgeFork() {
			return createProposal(ctx, uint64(currBlock.Timestamp), tx)
		} else {
			return handleInvalidSelector(tx)
		}
	case SelectorVote:
		if ctx.IsXHedgeFork() {
			return vote(ctx, uint64(currBlock.Timestamp), tx)
		} else {
			return handleInvalidSelector(tx)
		}
	case SelectorExecuteProposal:
		if ctx.IsXHedgeFork() {
			return executeProposal(ctx, uint64(currBlock.Timestamp), tx)
		} else {
			return handleInvalidSelector(tx)
		}
	case SelectorGetVote:
		if ctx.IsXHedgeFork() {
			return getVote(ctx, tx)
		} else {
			return handleInvalidSelector(tx)
		}
	default:
		return handleInvalidSelector(tx)
	}
}

// this functions is called when other contract calls sumVotingPower
func (_ *StakingContractExecutor) RequiredGas(input []byte) uint64 {
	return uint64(len(input))*SumVotingPowerGasPerByte + SumVotingPowerBaseGas
}

// function sumVotingPower(address[] calldata addrList) external override returns (uint summedPower, uint totalPower)
func (_ *StakingContractExecutor) Run(input []byte) ([]byte, error) {
	if len(input) < 4+32*2 || !bytes.Equal(input[:4], SelectorSumVotingPower[:]) {
		return nil, InvalidArgument
	}
	input = input[4+32*2:] // ignore selector, offset, and length
	addrSet := make(map[[20]byte]struct{}, len(input)/32)
	for i := 0; i+32 <= len(input); i += 32 {
		var addr [20]byte
		copy(addr[:], input[i+12:i+32])
		addrSet[addr] = struct{}{}
	}
	summedPower := int64(0)
	totalPower := int64(0)
	validators := []*types.Validator{}
	if readonlyStakingInfo != nil {
		validators = readonlyStakingInfo.Validators
	}
	countedAddrs := make(map[[20]byte]struct{}, len(input)/32)
	for _, val := range validators {
		_, hasValidator := addrSet[val.Address]
		_, hasRewardTo := addrSet[val.RewardTo]
		if hasValidator || hasRewardTo {
			if _, ok := countedAddrs[val.Address]; !ok { // a validator cannot be counted twice
				summedPower += val.VotingPower
				countedAddrs[val.Address] = struct{}{}
			}
		}
		totalPower += val.VotingPower
	}
	var result [64]byte
	uint256.NewInt(uint64(summedPower)).WriteToSlice(result[:32])
	uint256.NewInt(uint64(totalPower)).WriteToSlice(result[32:])
	return result[:], nil
}

// create a new validator with rewardTo, intro and pubkey fields, and stake it with some coins
func createValidator(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed //default status is failed
	gasUsed = GasOfValidatorOp
	if tx.Gas < gasUsed {
		outData = []byte(ErrOutOfGas.Error())
		gasUsed = tx.Gas
		return
	}
	callData := tx.Data[4:]
	if len(callData) < 96 {
		outData = []byte(InvalidCallData.Error())
		return
	}
	// First argument: rewardTo
	var rewardTo [20]byte
	copy(rewardTo[:], callData[12:32])
	// Second argument: introduction, byte32, limited to 32 byte
	intro := stringFromBytes(callData[32:64])
	// Third argument: pubkey (only createValidator has it)
	var pubkey [32]byte
	copy(pubkey[:], callData[64:])

	initialAmount := InitialStakingAmount
	if ctx.IsStakingFork() {
		initialAmount = MinimumStakingAmountAfterStakingFork
	}
	// need tx.value > 32bch
	if uint256.NewInt(0).SetBytes(tx.Value[:]).Cmp(initialAmount) <= 0 {
		outData = []byte(CreateValidatorCoinLtInitAmount.Error())
		return
	}

	stakingAcc, info := LoadStakingAccAndInfo(ctx)
	err := info.AddValidator(tx.From, pubkey, intro, tx.Value, rewardTo)
	if err != nil {
		outData = []byte(err.Error())
		return
	}

	//Now let's update the states, readonlyStakingInfo is unchanged because voting powers are unchanged.
	SaveStakingInfo(ctx, info)

	status, outData = transferStakedCoins(ctx, tx, stakingAcc)
	return
}

// edit a new validator's rewardTo and intro fields (pubkey cannot change), and stake it with some more coins
func editValidator(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed //default status is failed
	gasUsed = GasOfValidatorOp
	if tx.Gas < gasUsed {
		outData = []byte(ErrOutOfGas.Error())
		gasUsed = tx.Gas
		return
	}
	callData := tx.Data[4:]
	if len(callData) < 64 {
		outData = []byte(InvalidCallData.Error())
		return
	}
	// First argument: rewardTo
	var rewardTo [20]byte
	copy(rewardTo[:], callData[12:32])
	// Second argument: introduction, byte32, limited to 32 byte
	intro := stringFromBytes(callData[32:64])
	if ctx.IsXHedgeFork() {
		intro = strings.TrimRight(string(callData[32:64]), string([]byte{0}))
	}

	stakingAcc, info := LoadStakingAccAndInfo(ctx)

	val := info.GetValidatorByAddr(tx.From)
	if val == nil {
		outData = []byte(NoSuchValidator.Error())
		return
	}
	if val.IsRetiring {
		outData = []byte(ValidatorInRetiring.Error())
		return
	}
	var bz20Zero [20]byte
	if !bytes.Equal(rewardTo[:], bz20Zero[:]) {
		val.RewardTo = rewardTo
	}
	if len(intro) != 0 {
		val.Introduction = intro
	}
	coins4staking := uint256.NewInt(0).SetBytes32(tx.Value[:])
	if !coins4staking.IsZero() {
		stakedCoins := uint256.NewInt(0).SetBytes32(val.StakedCoins[:])
		stakedCoins.Add(stakedCoins, coins4staking)
		val.StakedCoins = stakedCoins.Bytes32()
	}

	//Now let's update the states, readonlyStakingInfo is unchanged because voting powers are unchanged.
	SaveStakingInfo(ctx, info)

	status, outData = transferStakedCoins(ctx, tx, stakingAcc)
	return
}

// a string stored in bz with one or more ending '\0' characters
func stringFromBytes(bz []byte) string {
	for i := len(bz) - 1; i >= 0; i-- {
		if bz[i] != 0 {
			return string(bz[:i+1])
		}
	}
	return string(bz)
}

func transferStakedCoins(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun, stakingAcc *mevmtypes.AccountInfo) (status int, outData []byte) {
	status = StatusFailed //default status is failed
	sender := ctx.GetAccount(tx.From)
	balance := sender.Balance()
	coins4staking := uint256.NewInt(0).SetBytes32(tx.Value[:])
	if balance.Lt(coins4staking) {
		outData = []byte(BalanceNotEnough.Error())
		return
	}

	if !coins4staking.IsZero() {
		balance.Sub(balance, coins4staking)
		sender.UpdateBalance(balance)
		stakingAccBalance := stakingAcc.Balance()
		stakingAccBalance.Add(stakingAccBalance, coins4staking)
		stakingAcc.UpdateBalance(stakingAccBalance)
		ctx.SetAccount(tx.From, sender)
		ctx.SetAccount(StakingContractAddress, stakingAcc)
	}
	status = StatusSuccess
	return
}

// a validator marks itself as "retiring", then at the next epoch it will not be elected as a validator
func retire(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed //default status is failed
	gasUsed = GasOfValidatorOp
	if tx.Gas < gasUsed {
		outData = []byte(ErrOutOfGas.Error())
		gasUsed = tx.Gas
		return
	}
	info := LoadStakingInfo(ctx)
	val := info.GetValidatorByAddr(tx.From)
	if val == nil {
		outData = []byte(NoSuchValidator.Error())
		return
	}
	val.IsRetiring = true
	//Now let's update the states, readonlyStakingInfo is unchanged because voting powers are unchanged.
	SaveStakingInfo(ctx, info)
	status = StatusSuccess
	return
}

func createProposal(ctx *mevmtypes.Context, now uint64, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfMinGasPriceOp
	if tx.Gas < gasUsed {
		outData = []byte(ErrOutOfGas.Error())
		gasUsed = tx.Gas
		return
	}
	info := LoadStakingInfo(ctx)
	val := info.GetValidatorByAddr(tx.From)
	if val == nil {
		outData = []byte(NoSuchValidator.Error())
		return
	}
	if val.VotingPower == 0 {
		outData = []byte(ValidatorNotActive.Error())
		return
	}
	target, _ := LoadProposal(ctx)
	if target != 0 {
		outData = []byte(StillInProposal.Error())
		return
	}
	callData := tx.Data[4:]
	if len(callData) != 32 {
		outData = []byte(InvalidCallData.Error())
		return
	}
	target = binary.BigEndian.Uint64(callData[24:])
	lastMinGasPrice := LoadMinGasPrice(ctx, true)
	if err := checkTarget(lastMinGasPrice, target); err != nil {
		outData = []byte(err.Error())
		return
	}

	SaveProposal(ctx, target, now+DefaultProposalDuration)
	SaveVote(ctx, tx.From, target, uint64(val.VotingPower))
	AddVoters(ctx, tx.From)
	status = StatusSuccess
	return
}

func vote(ctx *mevmtypes.Context, now uint64, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfMinGasPriceOp
	if tx.Gas < gasUsed {
		outData = []byte(ErrOutOfGas.Error())
		gasUsed = tx.Gas
		return
	}
	info := LoadStakingInfo(ctx)
	val := info.GetValidatorByAddr(tx.From)
	if val == nil {
		outData = []byte(NoSuchValidator.Error())
		return
	}
	if val.VotingPower == 0 {
		outData = []byte(ValidatorNotActive.Error())
		return
	}
	target, deadline := LoadProposal(ctx)
	if target == 0 {
		outData = []byte(NotInProposal.Error())
		return
	}
	if now >= deadline {
		outData = []byte(ProposalHasFinished.Error())
		return
	}
	callData := tx.Data[4:]
	if len(callData) != 32 {
		outData = []byte(InvalidCallData.Error())
		return
	}
	target = binary.BigEndian.Uint64(callData[24:])
	lastMinGasPrice := LoadMinGasPrice(ctx, true)
	if target == 0 {
		target = lastMinGasPrice
	}
	if err := checkTarget(lastMinGasPrice, target); err != nil {
		outData = []byte(err.Error())
		return
	}
	SaveVote(ctx, tx.From, target, uint64(val.VotingPower))
	AddVoters(ctx, tx.From)
	status = StatusSuccess
	return
}

func checkTarget(lastMinGasPrice, target uint64) error {
	if lastMinGasPrice != 0 && (lastMinGasPrice*MinGasPriceDeltaRate < target ||
		lastMinGasPrice > target*MinGasPriceDeltaRate) {
		return TargetExceedChangeDelta
	}
	if target > MinGasPriceUpperBound {
		return MinGasPriceTooBig
	}
	if target < MinGasPriceLowerBound {
		return MinGasPriceTooSmall
	}
	return nil
}

func executeProposal(ctx *mevmtypes.Context, now uint64, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfMinGasPriceOp
	if tx.Gas < gasUsed {
		outData = []byte(ErrOutOfGas.Error())
		gasUsed = tx.Gas
		return
	}
	target, deadline := LoadProposal(ctx)
	if target == 0 {
		outData = []byte(NotInProposal.Error())
		return
	}
	if now < deadline {
		outData = []byte(ProposalNotFinished.Error())
		return
	}
	voters := GetVoters(ctx)
	target = CalculateTarget(ctx, voters)
	SaveMinGasPrice(ctx, target, false)
	DeleteProposalInfos(ctx, voters)
	status = StatusSuccess
	return
}

func CalculateTarget(ctx *mevmtypes.Context, voters []common.Address) uint64 {
	var totalPower uint64
	var totalTarget uint64
	var targets []uint64
	for _, voter := range voters {
		target, votingPower := LoadVote(ctx, voter)
		targets = append(targets, target)
		totalPower += votingPower
		totalTarget += votingPower * target
	}
	t1 := CalcMedian(targets)
	t2 := totalTarget / totalPower
	return (t1 + t2) / 2
}

func CalcMedian(nums []uint64) uint64 {
	sort.Slice(nums, func(i, j int) bool {
		return nums[i] < nums[j]
	})
	index := len(nums) / 2
	if 2*index == len(nums) {
		return (nums[index-1] + nums[index]) / 2
	}
	return nums[index]
}

func getVote(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = GasOfMinGasPriceOp
	if tx.Gas < gasUsed {
		outData = []byte(ErrOutOfGas.Error())
		gasUsed = tx.Gas
		return
	}
	callData := tx.Data[4:]
	if len(callData) != 32 {
		outData = []byte(InvalidCallData.Error())
		return
	}
	var from common.Address
	from.SetBytes(callData[12:])
	target, _ := LoadVote(ctx, from)

	outData = make([]byte, 32)
	uint256.NewInt(target).WriteToSlice(outData)

	status = StatusSuccess
	return
}

func handleMinGasPrice(ctx *mevmtypes.Context, tx *mevmtypes.TxToRun, isIncrease bool, logger log.Logger) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	if ctx.IsXHedgeFork() {
		gasUsed = GasOfMinGasPriceOp
		if tx.Gas < gasUsed {
			outData = []byte(ErrOutOfGas.Error())
			gasUsed = tx.Gas
			status = StatusFailed
			return
		}
		status = StatusSuccess
		return
	} else {
		status = StatusFailed //default status is failed
		gasUsed = GasOfMinGasPriceOp
		if tx.Gas < gasUsed {
			outData = []byte(ErrOutOfGas.Error())
			gasUsed = tx.Gas
			return
		}
		sender := tx.From
		mGP := LoadMinGasPrice(ctx, false)
		lastMGP := LoadMinGasPrice(ctx, true) // this variable only updates at endblock
		info := LoadStakingInfo(ctx)
		isValidatorOrRewardTo := false
		activeValidators := GetActiveValidators(ctx, info.Validators)
		for _, v := range activeValidators {
			if v.Address == sender || v.RewardTo == sender {
				isValidatorOrRewardTo = true
				break
			}
		}
		if !isValidatorOrRewardTo {
			logger.Debug("sender is not active validator or its rewardTo", "sender", sender.String())
			outData = []byte(OperatorNotValidator.Error())
			return
		}
		if isIncrease {
			mGP += MinGasPriceDeltaRate * mGP / 100
		} else {
			mGP -= MinGasPriceDeltaRate * mGP / 100
		}
		logger.Debug(fmt.Sprintf("mGP(%d),lastMGP(%d),increase(%v)", mGP, lastMGP, isIncrease))

		if mGP < MinGasPriceLowerBoundOld {
			outData = []byte(MinGasPriceTooSmall.Error())
			return
		}
		if mGP > MinGasPriceUpperBound {
			outData = []byte(MinGasPriceTooBig.Error())
			return
		}
		if (mGP > lastMGP && 100*(mGP-lastMGP) > MinGasPriceDeltaRateInBlock*lastMGP) ||
			(mGP < lastMGP && 100*(lastMGP-mGP) > MinGasPriceDeltaRateInBlock*lastMGP) {
			outData = []byte(MinGasPriceExceedBlockChangeDelta.Error())
			return
		}
		SaveMinGasPrice(ctx, mGP, false)
		status = StatusSuccess
	}
	return
}

func handleInvalidSelector(tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = StatusFailed
	gasUsed = tx.Gas
	outData = []byte(InvalidSelector.Error())
	return
}

func LoadStakingAccAndInfo(ctx *mevmtypes.Context) (stakingAcc *mevmtypes.AccountInfo, info types.StakingInfo) {
	stakingAcc = ctx.GetAccount(StakingContractAddress)
	if stakingAcc == nil {
		panic("Cannot find staking contract")
	}
	info = LoadStakingInfo(ctx)
	return
}

func LoadStakingInfo(ctx *mevmtypes.Context) (info types.StakingInfo) {
	bz := ctx.GetStorageAt(StakingContractSequence, SlotStakingInfo)
	if bz == nil {
		return types.StakingInfo{}
	}
	_, err := info.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	return
}

func AddGenesisValidatorsIntoStakingInfo(ctx *mevmtypes.Context, genesisValidators []*types.Validator) {
	info := LoadStakingInfo(ctx)
	info.Validators = genesisValidators
	info.PendingRewards = make([]*types.PendingReward, len(genesisValidators))
	for i := range info.PendingRewards {
		info.PendingRewards[i] = &types.PendingReward{
			Address: genesisValidators[i].Address,
		}
	}
	SaveStakingInfo(ctx, info)
}

func SaveStakingInfo(ctx *mevmtypes.Context, info types.StakingInfo) {
	bz, err := info.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(StakingContractSequence, SlotStakingInfo, bz)
	readonlyStakingInfo = &info
}

func SaveEpoch(ctx *mevmtypes.Context, epoch *types.Epoch) {
	bz, err := epoch.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(StakingContractSequence, getSlotForEpoch(epoch.Number), bz)
}

func LoadEpoch(ctx *mevmtypes.Context, epochNum int64) (epoch types.Epoch, ok bool) {
	bz := ctx.GetStorageAt(StakingContractSequence, getSlotForEpoch(epochNum))
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

// get a slot number to store an epoch's validators, starting from (1<<64)
func getSlotForEpoch(epochNum int64) string {
	var buf [32]byte
	buf[23] = 1
	binary.BigEndian.PutUint64(buf[24:], uint64(epochNum))
	return string(buf[:])
}

func LoadMinGasPrice(ctx *mevmtypes.Context, isLast bool) uint64 {
	var bz []byte
	if isLast {
		bz = ctx.GetStorageAt(StakingContractSequence, SlotLastMinGasPrice)
	} else {
		bz = ctx.GetStorageAt(StakingContractSequence, SlotMinGasPriceInBlock)
	}
	if len(bz) == 0 {
		return DefaultMinGasPrice
	}
	return binary.BigEndian.Uint64(bz)
}

// In this file, only SlotMinGasPriceInBlock is written. In refresh of app.go, SlotMinGasPriceInBlock is copied to SlotLastMinGasPrice
func SaveMinGasPrice(ctx *mevmtypes.Context, minGP uint64, isLast bool) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], minGP)
	if isLast {
		ctx.SetStorageAt(StakingContractSequence, SlotLastMinGasPrice, b[:])
	} else {
		ctx.SetStorageAt(StakingContractSequence, SlotMinGasPriceInBlock, b[:])
	}
}

func LoadProposal(ctx *mevmtypes.Context) (target uint64, deadline uint64) {
	bz := ctx.GetStorageAt(StakingContractSequence, SlotMinGasPriceProposalTarget)
	if len(bz) == 0 {
		SaveProposal(ctx, 0, 0)
		return 0, 0
	}
	return binary.BigEndian.Uint64(bz[:8]), binary.BigEndian.Uint64(bz[8:])
}

func SaveProposal(ctx *mevmtypes.Context, target, deadline uint64) {
	var b [16]byte
	binary.BigEndian.PutUint64(b[:8], target)
	binary.BigEndian.PutUint64(b[8:], deadline)
	ctx.SetStorageAt(StakingContractSequence, SlotMinGasPriceProposalTarget, b[:])
}

func DeleteProposalInfos(ctx *mevmtypes.Context, voters []common.Address) {
	DeleteVoters(ctx)
	for _, v := range voters {
		DeleteVote(ctx, v)
	}
	DeleteProposal(ctx)
}

func DeleteVoters(ctx *mevmtypes.Context) {
	ctx.DeleteStorageAt(StakingContractSequence, SlotVoters)
}

func DeleteVote(ctx *mevmtypes.Context, validator common.Address) {
	key := sha256.Sum256(append(votingSlotHashPrefix[:], validator.Bytes()...))
	ctx.DeleteStorageAt(StakingContractSequence, string(key[:]))
}

func DeleteProposal(ctx *mevmtypes.Context) {
	SaveProposal(ctx, 0, 0)
}

func LoadVote(ctx *mevmtypes.Context, validator common.Address) (target, votingPower uint64) {
	//var b [16]byte
	//binary.BigEndian.PutUint64(b[:8], target)
	//binary.BigEndian.PutUint64(b[8:], votingPower)
	key := sha256.Sum256(append(votingSlotHashPrefix[:], validator.Bytes()...))
	bz := ctx.GetStorageAt(StakingContractSequence, string(key[:]))
	if len(bz) == 0 {
		return 0, 0
	}
	return binary.BigEndian.Uint64(bz[:8]), binary.BigEndian.Uint64(bz[8:])
}

func SaveVote(ctx *mevmtypes.Context, validator common.Address, target, votingPower uint64) {
	var b [16]byte
	binary.BigEndian.PutUint64(b[:8], target)
	binary.BigEndian.PutUint64(b[8:], votingPower)
	key := sha256.Sum256(append(votingSlotHashPrefix[:], validator.Bytes()...))
	ctx.SetStorageAt(StakingContractSequence, string(key[:]), b[:])
}

func AddVoters(ctx *mevmtypes.Context, validator common.Address) {
	voters := GetVoters(ctx)
	for _, v := range voters {
		if validator == v {
			return
		}
	}
	voters = append(voters, validator)
	bz, _ := json.Marshal(voters)
	ctx.SetStorageAt(StakingContractSequence, SlotVoters, bz)
}

func GetVoters(ctx *mevmtypes.Context) (voters []common.Address) {
	bz := ctx.GetStorageAt(StakingContractSequence, SlotVoters)
	if len(bz) == 0 {
		return nil
	}
	_ = json.Unmarshal(bz, &voters)
	return voters
}

func LoadOnlineInfo(ctx *mevmtypes.Context) (infos types.ValidatorOnlineInfos) {
	bz := ctx.GetStorageAt(StakingContractSequence, SlotOnlineInfo)
	if len(bz) == 0 {
		return
	}
	_, err := infos.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	return
}

func SaveOnlineInfo(ctx *mevmtypes.Context, infos types.ValidatorOnlineInfos) {
	bz, err := infos.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(StakingContractSequence, SlotOnlineInfo, bz)
}

func LoadValidatorWatchInfo(ctx *mevmtypes.Context) (infos types.ValidatorWatchInfos) {
	bz := ctx.GetStorageAt(StakingContractSequence, SlotWatchInfo)
	if len(bz) == 0 {
		return
	}
	_, err := infos.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	return
}

func SaveValidatorWatchInfo(ctx *mevmtypes.Context, infos types.ValidatorWatchInfos) {
	bz, err := infos.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(StakingContractSequence, SlotWatchInfo, bz)
}

// =========================================================================================
// Following staking functions cannot be invoked through smart contract calls

func NewOnlineInfos(activeValidators []*types.Validator, startHeight int64) *types.ValidatorOnlineInfos {
	var infos types.ValidatorOnlineInfos
	var onlineInfos []*types.OnlineInfo
	for _, val := range activeValidators {
		var onlineInfo types.OnlineInfo
		copy(onlineInfo.ValidatorConsensusAddress[:], ed25519.PubKey(val.Pubkey[:]).Address().Bytes())
		onlineInfos = append(onlineInfos, &onlineInfo)
	}
	infos.OnlineInfos = onlineInfos
	infos.StartHeight = startHeight
	return &infos
}

func NewWatchInfos(activeValidators []*types.Validator, startHeight int64) *types.ValidatorWatchInfos {
	var infos types.ValidatorWatchInfos
	var watchInfos []*types.WatchInfo
	for _, val := range activeValidators {
		var watchInfo types.WatchInfo
		copy(watchInfo.ValidatorConsensusAddress[:], ed25519.PubKey(val.Pubkey[:]).Address().Bytes())
		watchInfos = append(watchInfos, &watchInfo)
	}
	infos.WatchInfos = watchInfos
	infos.StartHeight = startHeight
	return &infos
}

func UpdateOnlineInfos(ctx *mevmtypes.Context, infos types.ValidatorOnlineInfos, voters [][]byte) (startHeight int64) {
	voterMap := make(map[[20]byte]bool, len(voters))
	for _, voter := range voters {
		var v [20]byte
		copy(v[:], voter)
		voterMap[v] = true
	}
	if ctx.Height == infos.StartHeight+param.OnlineWindowSize {
		for _, info := range infos.OnlineInfos {
			info.SignatureCount = 0
			// todo: reset info.HeightOfLastSignature ?
		}
		infos.StartHeight = ctx.Height
	}
	for _, info := range infos.OnlineInfos {
		if voterMap[info.ValidatorConsensusAddress] {
			info.SignatureCount++
			info.HeightOfLastSignature = ctx.Height
		}
	}
	// todo: make sure if has the case: validator in voters but not in info.OnlineInfos
	SaveOnlineInfo(ctx, infos)
	return infos.StartHeight
}

func HandleOnlineInfosForBugFix(ctx *mevmtypes.Context, stakingInfo *types.StakingInfo, voters [][]byte) (slashValidators [][20]byte) {
	var infos types.ValidatorOnlineInfos
	activeValidators := make([]*types.Validator, 0, len(stakingInfo.Validators))
	infos = *NewOnlineInfos(activeValidators, ctx.Height)
	UpdateOnlineInfos(ctx, infos, voters)
	return
}

func HandleWatchInfos(ctx *mevmtypes.Context, stakingInfo *types.StakingInfo, voters [][]byte) {
	infos := LoadValidatorWatchInfo(ctx)
	if infos.StartHeight == 0 {
		activeValidators := GetActiveValidators(ctx, stakingInfo.Validators)
		infos = *NewWatchInfos(activeValidators, ctx.Height)
	}
	if ctx.Height == infos.StartHeight+param.ValidatorWatchWindowSize {
		decreaseVpValidatorMap := make(map[common.Address]bool)
		recoveryVpValidatorMap := make(map[common.Address]bool)
		for _, info := range infos.WatchInfos {
			if info.SignatureCount < param.ValidatorWatchMinSignatures && !info.Handled {
				decreaseVpValidatorMap[info.ValidatorConsensusAddress] = true
				info.Handled = true
			}
			if info.SignatureCount >= param.ValidatorWatchMinSignatures && info.Handled {
				info.Handled = false
				recoveryVpValidatorMap[info.ValidatorConsensusAddress] = true
			}
			// reset
			info.SignatureCount = 0
		}
		for _, val := range stakingInfo.Validators {
			var address [20]byte
			copy(address[:], ed25519.PubKey(val.Pubkey[:]).Address().Bytes())
			if decreaseVpValidatorMap[address] {
				val.VotingPower = val.VotingPower / param.VotingPowerDivider
			}
			if recoveryVpValidatorMap[address] {
				val.VotingPower = val.VotingPower * param.VotingPowerDivider
			}
		}
		// reset
		infos.StartHeight = ctx.Height
	} else {
		voterMap := make(map[[20]byte]bool, len(voters))
		for _, voter := range voters {
			var v [20]byte
			copy(v[:], voter)
			voterMap[v] = true
		}
		for _, info := range infos.WatchInfos {
			if voterMap[info.ValidatorConsensusAddress] {
				info.SignatureCount++
				info.HeightOfLastSignature = ctx.Height
			}
		}
	}
	SaveValidatorWatchInfo(ctx, infos)
}

func HandleOnlineInfos(ctx *mevmtypes.Context, stakingInfo *types.StakingInfo, voters [][]byte) (slashValidators [][20]byte) {
	var lazyValidators = make(map[[20]byte]bool, len(voters))
	infos := LoadOnlineInfo(ctx)
	if infos.StartHeight == 0 {
		activeValidators := GetActiveValidators(ctx, stakingInfo.Validators)
		infos = *NewOnlineInfos(activeValidators, ctx.Height)
		UpdateOnlineInfos(ctx, infos, voters)
		return
	}
	if ctx.Height != infos.StartHeight+param.OnlineWindowSize {
		UpdateOnlineInfos(ctx, infos, voters)
		return
	}
	var newInfos []*types.OnlineInfo
	for _, info := range infos.OnlineInfos {
		if info.SignatureCount < param.MinOnlineSignatures {
			lazyValidators[info.ValidatorConsensusAddress] = true
		} else {
			newInfos = append(newInfos, info)
		}
	}
	if len(lazyValidators) == 0 {
		UpdateOnlineInfos(ctx, infos, voters)
		return
	}
	infos.OnlineInfos = newInfos
	for _, val := range stakingInfo.Validators {
		var address [20]byte
		copy(address[:], ed25519.PubKey(val.Pubkey[:]).Address().Bytes())
		if lazyValidators[address] && !val.IsRetiring {
			slashValidators = append(slashValidators, address)
		}
	}
	UpdateOnlineInfos(ctx, infos, voters)
	return
}

// slashValidators and lastVoters are consensus addresses generated from validator consensus pubkey
func SlashAndReward(ctx *mevmtypes.Context, duplicateSigSlashValidators [][20]byte,
	currProposer, lastProposer [20]byte, lastVoters [][]byte, /*include proposer*/
	blockReward *uint256.Int) (currValidators, newValidators []*types.Validator, currEpochNum int64) {

	stakingAcc, info := LoadStakingAccAndInfo(ctx)
	currEpochNum = info.CurrEpochNum
	currValidators = GetActiveValidators(ctx, info.Validators)

	pubkeyMapByConsAddr := make(map[[20]byte][32]byte, len(info.Validators))
	var consAddr [20]byte
	for _, v := range info.Validators {
		copy(consAddr[:], ed25519.PubKey(v.Pubkey[:]).Address().Bytes())
		pubkeyMapByConsAddr[consAddr] = v.Pubkey
	}
	//slash first
	for _, v := range duplicateSigSlashValidators {
		if pubkey, ok := pubkeyMapByConsAddr[v]; ok {
			slashAmount := uint256.NewInt(0)
			if ctx.IsStakingFork() {
				slashAmount = uint256.NewInt(0).Div(MinimumStakingAmountAfterStakingFork, uint256.NewInt(param.DuplicateSigSlashAMountDivisor))
			}
			Slash(ctx, &info, pubkey, slashAmount)
		}
	}
	if ctx.IsStakingFork() {
		HandleWatchInfos(ctx, &info, lastVoters)
		notOnlineSlashValidators := HandleOnlineInfos(ctx, &info, lastVoters)
		for _, v := range notOnlineSlashValidators {
			if pubkey, ok := pubkeyMapByConsAddr[v]; ok {
				slashAmount := uint256.NewInt(0).Div(MinimumStakingAmountAfterStakingFork, uint256.NewInt(param.NotOnlineSlashAmountDivisor))
				Slash(ctx, &info, pubkey, slashAmount)
			}
		}
	} else if ctx.Height == 8000000 {
		// hardcode for 8000000 staking fork early come bug
		HandleOnlineInfosForBugFix(ctx, &info, lastVoters)
	}
	voters := make([][32]byte, 0, len(lastVoters))
	var tmpAddr [20]byte
	for _, c := range lastVoters {
		copy(tmpAddr[:], c)
		if voter, ok := pubkeyMapByConsAddr[tmpAddr]; ok {
			voters = append(voters, voter)
		}
	}
	DistributeFee(ctx, stakingAcc, &info, blockReward, pubkeyMapByConsAddr[currProposer],
		pubkeyMapByConsAddr[lastProposer], voters)
	newValidators = GetActiveValidators(ctx, info.Validators)
	SaveStakingInfo(ctx, info)
	return
}

// Slash 'amount' of coins from the validator with 'pubkey'. These coins are burnt and booked on BlackHole acc.
func Slash(ctx *mevmtypes.Context, info *types.StakingInfo, pubkey [32]byte, amount *uint256.Int) (totalSlashed *uint256.Int) {
	val := info.GetValidatorByPubkey(pubkey)
	if val == nil {
		return // If tendermint works fine, we'll never reach here
	}
	coins := uint256.NewInt(0).SetBytes32(val.StakedCoins[:])
	if coins.Lt(amount) { // not enough coins to be slashed
		totalSlashed = coins.Clone()
		coins.SetUint64(0)
	} else {
		totalSlashed = amount.Clone()
		coins.Sub(coins, amount)
	}
	val.StakedCoins = coins.Bytes32()
	if ctx.IsStakingFork() {
		// clear the voting power and set IsRetiring flag when validator slashed after staking fork
		val.IsRetiring = true
		val.VotingPower = 0
	}

	totalCleared := info.ClearRewardsOf(val.Address)
	totalSlashed.Add(totalSlashed, totalCleared)

	if ctx.IsStakingFork() {
		slashReceiver := common.HexToAddress(param.SlashReceiver)
		err := ebp.SubSenderAccBalance(ctx, StakingContractAddress, totalSlashed)
		if err != nil {
			// should never here
			panic(err)
		}
		receiverAcc := ctx.GetAccount(slashReceiver)
		balance := receiverAcc.Balance()
		balance.Add(balance, totalSlashed)
		receiverAcc.UpdateBalance(balance)
		ctx.SetAccount(slashReceiver, receiverAcc)
	} else {
		// deduct the totalSlashed from stakingAcc and burn them, must no error, not check
		_ = ebp.TransferFromSenderAccToBlackHoleAcc(ctx, StakingContractAddress, totalSlashed)
		incrAllBurnt(ctx, totalSlashed)
	}
	return
}

// Increase the slot of 'all burnt' inside stakingAcc
func incrAllBurnt(ctx *mevmtypes.Context, amount *uint256.Int) {
	allBurnt := uint256.NewInt(0)
	bz := ctx.GetStorageAt(StakingContractSequence, SlotAllBurnt)
	if len(bz) != 0 {
		allBurnt.SetBytes32(bz)
	}
	allBurnt.Add(allBurnt, amount)
	bz32 := allBurnt.Bytes32()
	ctx.SetStorageAt(StakingContractSequence, SlotAllBurnt, bz32[:])
}

// distribute the collected gas fee to validators who voted for current block, half fee burnt to blackHole Acc.
func DistributeFee(ctx *mevmtypes.Context, stakingAcc *mevmtypes.AccountInfo, info *types.StakingInfo,
	collectedFee *uint256.Int, collector, proposer [32]byte /*operator pubKey*/, voters [][32]byte) {
	if collectedFee == nil {
		return
	}

	// the collected fee is saved as stakingAcc's balance, just the same way as the staked coins
	stakingAccBalance := stakingAcc.Balance()
	stakingAccBalance.Add(stakingAccBalance, collectedFee)
	stakingAcc.UpdateBalance(stakingAccBalance)
	ctx.SetAccount(StakingContractAddress, stakingAcc)

	//burn half of the collected fees
	halfFeeToBurn := uint256.NewInt(0).Rsh(collectedFee, 1)
	collectedFee.Sub(collectedFee, halfFeeToBurn)
	_ = ebp.TransferFromSenderAccToBlackHoleAcc(ctx, StakingContractAddress, halfFeeToBurn)

	totalVotingPower, votedPower := int64(0), int64(0)
	for _, val := range GetActiveValidators(ctx, info.Validators) {
		totalVotingPower += val.VotingPower
	}
	valMapByPubkey := info.GetValMapByPubkey()
	for _, voter := range voters {
		val := valMapByPubkey[voter]
		votedPower += val.VotingPower
	}

	proposerBaseFee := uint256.NewInt(0)
	collectorFee := uint256.NewInt(0)
	if proposer != [32]byte{} {
		// proposerBaseFee goes to the proposer, collectorFee goes to the collector
		proposerBaseFee.Mul(collectedFee,
			uint256.NewInt(param.ProposerBaseMintFeePercentage))
		proposerBaseFee.Div(proposerBaseFee, uint256.NewInt(100))
		collectedFee.Sub(collectedFee, proposerBaseFee)
		collectorFee.Mul(collectedFee,
			uint256.NewInt(param.CollectorMintFeePercentage))
		collectorFee.Mul(collectorFee, uint256.NewInt(uint64(votedPower)))
		collectorFee.Div(collectorFee, uint256.NewInt(uint64(100*totalVotingPower)))
		collectedFee.Sub(collectedFee, collectorFee)
	}
	rwdMapByAddr := info.GetCurrRewardMapByAddr()
	remainedFee := collectedFee.Clone()
	//distribute to the non-proposer voters
	for _, voter := range voters {
		if bytes.Equal(proposer[:], voter[:]) {
			continue // proposer will be handled at the next step
		}
		val := valMapByPubkey[voter]
		rwdCoins := uint256.NewInt(0).Mul(collectedFee, uint256.NewInt(uint64(val.VotingPower)))
		rwdCoins.Div(rwdCoins, uint256.NewInt(uint64(votedPower)))
		remainedFee.Sub(remainedFee, rwdCoins)
		distributeToValidator(info, rwdMapByAddr, rwdCoins, val)
	}

	if proposer != [32]byte{} {
		//distribute to the proposer
		proposerVal := valMapByPubkey[proposer]
		coins := uint256.NewInt(0).Add(proposerBaseFee, remainedFee)
		distributeToValidator(info, rwdMapByAddr, coins, proposerVal)
	} else if !remainedFee.IsZero() {
		_ = ebp.TransferFromSenderAccToBlackHoleAcc(ctx, StakingContractAddress, remainedFee)
	}

	if collector != [32]byte{} {
		distributeToValidator(info, rwdMapByAddr, collectorFee, valMapByPubkey[collector])
	} else if !collectorFee.IsZero() {
		_ = ebp.TransferFromSenderAccToBlackHoleAcc(ctx, StakingContractAddress, collectorFee)
	}
}

func distributeToValidator(info *types.StakingInfo, rwdMapByAddr map[[20]byte]*types.PendingReward,
	rwdCoins *uint256.Int, val *types.Validator) {

	rwd := rwdMapByAddr[val.Address]
	if rwd == nil {
		rwd = &types.PendingReward{
			Address:  val.Address,
			EpochNum: info.CurrEpochNum,
			Amount:   [32]byte{},
		}
		info.PendingRewards = append(info.PendingRewards, rwd)
	}
	coins := uint256.NewInt(0).SetBytes32(rwd.Amount[:])
	coins.Add(coins, rwdCoins)
	rwd.Amount = coins.Bytes32()
}

// switch to a new epoch
func SwitchEpoch(ctx *mevmtypes.Context, epoch *types.Epoch, posVotes map[[32]byte]int64, logger log.Logger) []*types.Validator {
	stakingAcc, info := LoadStakingAccAndInfo(ctx)
	//increase currEpochNum no matter if epoch is valid
	info.CurrEpochNum++
	epoch.Number = info.CurrEpochNum
	SaveEpoch(ctx, epoch)
	logger.Debug(fmt.Sprintf("Epoch info in switchEpoch [newPpochNumber:%d,startHeight:%d,EndTime:%d]", epoch.Number, epoch.StartHeight, epoch.EndTime))

	// distribute mature pending reward to rewardTo
	deliverMintRewardInEpoch(ctx, stakingAcc, &info)

	isValid, pubkey2power, oldActiveValidators := checkEpoch(ctx, info, epoch, posVotes, logger)
	if !isValid {
		updatePendingRewardsInNewEpoch(oldActiveValidators, &info, logger)
		SaveStakingInfo(ctx, info)
		return nil
	}
	// someone who call createValidator before switchEpoch can enjoy the voting power update
	// someone who call retire() before switchEpoch cannot get elected in this update
	updateVotingPower(ctx, &info, pubkey2power)
	// payback staking coins to rewardTo of useless validators and delete these validators
	clearUselessValidators(ctx, stakingAcc, &info)
	// allocate new entries in info.PendingRewards
	activeValidators := GetActiveValidators(ctx, info.Validators)
	updatePendingRewardsInNewEpoch(activeValidators, &info, logger)
	SaveStakingInfo(ctx, info)
	if ctx.IsStakingFork() {
		SaveValidatorWatchInfo(ctx, *NewWatchInfos(activeValidators, ctx.Height))
		SaveOnlineInfo(ctx, *NewOnlineInfos(activeValidators, ctx.Height))
	}
	return activeValidators
}

// deliver pending rewards which are mature now to rewardTo
func deliverMintRewardInEpoch(ctx *mevmtypes.Context, stakingAcc *mevmtypes.AccountInfo, info *types.StakingInfo) {
	stakingAccBalance := stakingAcc.Balance()
	newPRList := make([]*types.PendingReward, 0, len(info.PendingRewards))
	valMapByAddr := info.GetValMapByAddr()
	rewardMap := make(map[[20]byte]*uint256.Int, len(info.PendingRewards))
	// summarize all the mature rewards
	for _, pr := range info.PendingRewards {
		if pr.EpochNum >= info.CurrEpochNum-param.EpochCountBeforeRewardMature {
			newPRList = append(newPRList, pr) //not mature yet
			continue
		}
		val := valMapByAddr[pr.Address]
		if _, ok := rewardMap[val.RewardTo]; !ok {
			rewardMap[val.RewardTo] = uint256.NewInt(0)
		}
		rewardMap[val.RewardTo].Add(rewardMap[val.RewardTo], uint256.NewInt(0).SetBytes32(pr.Amount[:]))
	}
	info.PendingRewards = newPRList

	// increase rewardTo's balance and decrease stakingAcc's balance
	for addr, rwd := range rewardMap {
		acc := ctx.GetAccount(addr)
		if acc == nil {
			acc = mevmtypes.ZeroAccountInfo()
		}
		stakingAccBalance.Sub(stakingAccBalance, rwd)
		balance := acc.Balance()
		balance.Add(balance, rwd)
		acc.UpdateBalance(balance)
		ctx.SetAccount(addr, acc)
	}
	stakingAcc.UpdateBalance(stakingAccBalance)
}

func checkEpoch(ctx *mevmtypes.Context, info types.StakingInfo, epoch *types.Epoch, posVotes map[[32]byte]int64, logger log.Logger) (bool, map[[32]byte]int64, []*types.Validator) {
	powTotalNomination, pubkey2power := getPubkey2Power(ctx, info, epoch, posVotes, logger)
	activeValidators := GetActiveValidators(ctx, info.Validators)
	if !((param.IsAmber && ctx.IsXHedgeFork()) || (!param.IsAmber && ctx.IsStakingFork())) {
		if powTotalNomination < param.StakingNumBlocksInEpoch*int64(param.StakingMinVotingPercentPerEpoch)/100 {
			logger.Debug("PoWTotalNomination not big enough", "PoWTotalNomination", powTotalNomination)
			return false, pubkey2power, activeValidators
		}
	}
	if len(pubkey2power) < len(activeValidators)*param.StakingMinVotingPubKeysPercentPerEpoch/100 {
		logger.Debug("Voting pubKeys smaller than MinVotingPubKeysPercentPerEpoch", "validator count", len(epoch.Nominations))
		return false, pubkey2power, activeValidators
	}
	return true, pubkey2power, activeValidators
}

func getPubkey2Power(ctx *mevmtypes.Context, info types.StakingInfo, epoch *types.Epoch, posVotes map[[32]byte]int64, logger log.Logger) (powTotalNomination int64, pubkey2power map[[32]byte]int64) {
	validatorSet := make(map[[32]byte]bool, len(info.Validators))
	for _, val := range info.Validators {
		if !val.IsRetiring {
			validatorSet[val.Pubkey] = true
		}
	}
	for _, n := range epoch.Nominations {
		logger.Debug(fmt.Sprintf("Nomination: pubkey(%s), NominatedCount(%d)", ed25519.PubKey(n.Pubkey[:]).String(), n.NominatedCount))
	}
	validNominations := make([]*types.Nomination, 0, len(epoch.Nominations)+len(posVotes))
	totalVotedBlocks := int64(0) // how many blocks were used for pow voting. at most 2016
	for _, n := range epoch.Nominations {
		if validatorSet[n.Pubkey] { // votes to non-validators are ingored
			validNominations = append(validNominations, n)
			totalVotedBlocks += n.NominatedCount
		}
	}
	for _, n := range validNominations {
		logger.Debug(fmt.Sprintf("Valid Nomination: pubkey(%s), NominatedCount(%d)", ed25519.PubKey(n.Pubkey[:]).String(), n.NominatedCount))
		powTotalNomination += n.NominatedCount
	}

	validVotes := make(map[[32]byte]int64, len(posVotes))
	totalValidCoinDays := int64(0) // how many coindays were used for pos voting. at most 21_000_000*365
	for pubkey, coindays := range posVotes {
		if validatorSet[pubkey] { // votes to non-validators are ingored
			validVotes[pubkey] = coindays
			totalValidCoinDays += coindays
		}
	}
	// change block count to coindays
	if totalValidCoinDays > 0 {
		for i, n := range validNominations {
			validNominations[i].NominatedCount = totalValidCoinDays * n.NominatedCount / totalVotedBlocks
		}
	}
	// merge pos and power votes
	for _, n := range validNominations {
		coindays := validVotes[n.Pubkey]
		validVotes[n.Pubkey] = coindays + n.NominatedCount
	}
	validNominations = validNominations[:0]
	for pubkey, coindays := range validVotes {
		n := &types.Nomination{Pubkey: pubkey, NominatedCount: coindays}
		validNominations = append(validNominations, n)
	}

	// select at most MaxActiveValidatorCount validators
	nominationHeap := types.NominationHeap(validNominations)
	heap.Init(&nominationHeap)
	pubkey2power = make(map[[32]byte]int64, len(validNominations))
	for i := 0; i < param.MaxActiveValidatorCount && len(nominationHeap) > 0; i++ {
		n := heap.Pop(&nominationHeap).(*types.Nomination)
		if ctx.IsStakingFork() {
			pubkey2power[n.Pubkey] = 10000
		} else {
			pubkey2power[n.Pubkey] = 1
		}
	}
	return
}

func updatePendingRewardsInNewEpoch(activeValidators []*types.Validator, info *types.StakingInfo, logger log.Logger) {
	for _, val := range activeValidators {
		pr := &types.PendingReward{
			Address:  val.Address,
			EpochNum: info.CurrEpochNum,
		}
		info.PendingRewards = append(info.PendingRewards, pr)
		logger.Debug(fmt.Sprintf("Active validator after switch epoch, address:%s, voting power:%d", common.Address(val.Address).String(), val.VotingPower))
	}
}

// Clear the old voting powers and assign pubkey2power to validators.
func updateVotingPower(ctx *mevmtypes.Context, info *types.StakingInfo, pubkey2power map[[32]byte]int64) {
	for _, val := range info.Validators {
		val.VotingPower = 0
	}
	valMapByPubkey := info.GetValMapByPubkey()
	minimumStakingAmount := MinimumStakingAmount
	if ctx.IsStakingFork() {
		minimumStakingAmount = MinimumStakingAmountAfterStakingFork
	}
	for pubkey, power := range pubkey2power {
		val, ok := valMapByPubkey[pubkey]
		if !ok || val.IsRetiring {
			continue
		}
		if uint256.NewInt(0).SetBytes32(val.StakedCoins[:]).Cmp(minimumStakingAmount) >= 0 {
			val.VotingPower = power
		}
	}
}

// Remove the useless validators from info and return StakedCoins to them
func clearUselessValidators(ctx *mevmtypes.Context, stakingAcc *mevmtypes.AccountInfo, info *types.StakingInfo) {
	uselessValMap := info.GetUselessValidators()
	valMapByAddr := info.GetValMapByAddr()
	stakingAccBalance := stakingAcc.Balance()
	for addr := range uselessValMap {
		val := valMapByAddr[addr]
		acc := ctx.GetAccount(val.RewardTo)
		if acc == nil {
			acc = mevmtypes.ZeroAccountInfo()
		}
		coins := uint256.NewInt(0).SetBytes32(val.StakedCoins[:])
		stakingAccBalance.Sub(stakingAccBalance, coins)
		balance := acc.Balance()
		balance.Add(balance, coins)
		acc.UpdateBalance(balance)
		ctx.SetAccount(val.RewardTo, acc)
	}
	stakingAcc.UpdateBalance(stakingAccBalance)
	ctx.SetAccount(StakingContractAddress, stakingAcc)
	//delete useless validator
	if len(uselessValMap) > 0 {
		newVals := make([]*types.Validator, 0, len(info.Validators))
		for _, v := range info.Validators {
			if _, ok := uselessValMap[v.Address]; ok {
				continue
			}
			newVals = append(newVals, v)
		}
		info.Validators = newVals
	}
	stakingAcc.UpdateBalance(stakingAccBalance)
	ctx.SetAccount(StakingContractAddress, stakingAcc)
}

// Returns current validators on duty, who must have enough coins staked and be not in a retiring process
// only update validator voting power on switchEpoch
func GetActiveValidators(ctx *mevmtypes.Context, vals []*types.Validator) []*types.Validator {
	minStakedCoins := MinimumStakingAmount
	if ctx.IsStakingFork() {
		minStakedCoins = MinimumStakingAmountAfterStakingFork
	}
	res := make([]*types.Validator, 0, len(vals))
	for _, val := range vals {
		coins := uint256.NewInt(0).SetBytes32(val.StakedCoins[:])
		if coins.Cmp(minStakedCoins) >= 0 && !val.IsRetiring && val.VotingPower > 0 {
			res = append(res, val)
		}
	}
	//sort: 1.voting power; 2.create validator time (so stable sort is required)
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].VotingPower > res[j].VotingPower
	})
	if len(res) > param.MaxActiveValidatorCount {
		res = res[:param.MaxActiveValidatorCount]
	}
	return res
}
