package staking

import (
	"encoding/binary"
	"encoding/hex"
	"math"
	"strings"

	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/libs/log"

	mevmtypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/staking/types"
)

const (
	StakingContractSequence uint64 = math.MaxUint64 - 2 /*uint64(-3)*/
	Uint64_1e18             uint64 = 1000_000_000_000_000_000
)

var (
	//contract address, 10000
	StakingContractAddress = [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x27, 0x10}

	//slot
	SlotStakingInfo     = strings.Repeat(string([]byte{0}), 32)
	SlotMinGasPrice     = strings.Repeat(string([]byte{0}), 31) + string([]byte{2})
	SlotLastMinGasPrice = strings.Repeat(string([]byte{0}), 31) + string([]byte{3})

	// slot in hex
	SlotMinGasPriceHex = hex.EncodeToString([]byte(SlotMinGasPrice))
)

var (
	/*------param------*/
	MinimumStakingAmount        = uint256.NewInt(0)
	DefaultMinGasPrice   uint64 = param.DefaultMinGasPrice //10gwei
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

func SaveStakingInfo(ctx *mevmtypes.Context, info types.StakingInfo) {
	bz, err := info.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(StakingContractSequence, SlotStakingInfo, bz)
	readonlyStakingInfo = &info
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
		bz = ctx.GetStorageAt(StakingContractSequence, SlotMinGasPrice)
	}
	if len(bz) == 0 {
		return DefaultMinGasPrice
	}
	return binary.BigEndian.Uint64(bz)
}

func SaveMinGasPrice(ctx *mevmtypes.Context, minGP uint64, isLast bool) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], minGP)
	if isLast {
		ctx.SetStorageAt(StakingContractSequence, SlotLastMinGasPrice, b[:])
	} else {
		ctx.SetStorageAt(StakingContractSequence, SlotMinGasPrice, b[:])
	}
}
