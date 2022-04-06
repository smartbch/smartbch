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
