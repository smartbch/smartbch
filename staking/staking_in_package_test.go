package staking

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/smartbch/moeingevm/types"
	"testing"

	"github.com/stretchr/testify/require"

	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

func TestCheckTarget(t *testing.T) {
	lastMinGasPrice := MinGasPriceLowerBound
	target := MinGasPriceDeltaRate * lastMinGasPrice
	err := checkTarget(lastMinGasPrice, target)
	require.NoError(t, err)

	target += 1
	err = checkTarget(lastMinGasPrice, target)
	require.Error(t, err, TargetExceedChangeDelta)

	target = lastMinGasPrice/MinGasPriceDeltaRate - 1
	err = checkTarget(lastMinGasPrice, target)
	require.Error(t, err, TargetExceedChangeDelta)

	lastMinGasPrice = 0
	target = MinGasPriceUpperBound + 1
	err = checkTarget(lastMinGasPrice, target)
	require.Error(t, err, MinGasPriceTooBig)

	target = MinGasPriceLowerBound - 1
	err = checkTarget(lastMinGasPrice, target)
	require.Error(t, err, MinGasPriceTooSmall)
}

func TestCalcMedian(t *testing.T) {
	nums := [8]uint64{5, 3, 1, 2, 4, 6, 7, 3}
	m := CalcMedian(nums[:])
	require.Equal(t, uint64(3), m)

	nums2 := [7]uint64{5, 3, 1, 2, 4, 6, 7}
	m = CalcMedian(nums2[:])
	require.Equal(t, uint64(4), m)

	nums3 := [7]uint64{1, 2, 2, 3, 3, 4, 1}
	m = CalcMedian(nums3[:])
	require.Equal(t, uint64(2), m)
}

//add gas consume
func TestHandleMinGasPrice(t *testing.T) {
	ctx := types.NewContext(nil, nil)
	ctx.SetCurrentHeight(1)
	ctx.SetXHedgeForkBlock(0)
	status, _, gasUsed, _ := handleMinGasPrice(ctx, common.Address{}, false, nil)
	require.Equal(t, StatusSuccess, status)
	require.Equal(t, GasOfMinGasPriceOp, gasUsed)
}

func TestGetActiveValidators(t *testing.T) {
	si := &stakingtypes.StakingInfo{
		Validators: []*stakingtypes.Validator{
			{Address: [20]byte{0xad, 0x01}, StakedCoins: [32]byte{0x10}, IsRetiring: false, VotingPower: 1},
			{Address: [20]byte{0xad, 0x02}, StakedCoins: [32]byte{0x11}, IsRetiring: true, VotingPower: 10},
			{Address: [20]byte{0xad, 0x03}, StakedCoins: [32]byte{0x12}, IsRetiring: false, VotingPower: 0},
			{Address: [20]byte{0xad, 0x04}, StakedCoins: [32]byte{0x13}, IsRetiring: false, VotingPower: 3},
			{Address: [20]byte{0xad, 0x05}, StakedCoins: [32]byte{0x14}, IsRetiring: false, VotingPower: 1},
			{Address: [20]byte{0xad, 0x06}, StakedCoins: [32]byte{0x15}, IsRetiring: false, VotingPower: 7},
			{Address: [20]byte{0xad, 0x07}, StakedCoins: [32]byte{0x16}, IsRetiring: false, VotingPower: 4},
		},
	}

	min := [32]byte{0x11}
	ctx := types.NewContext(nil, nil)
	ctx.SetCurrentHeight(1)
	ctx.SetStakingForkBlock(100)
	MinimumStakingAmount = uint256.NewInt(0).SetBytes32(min[:])
	vals := GetActiveValidators(ctx, si.Validators)
	fmt.Println(len(vals))
	require.Len(t, vals, 4)
	require.Equal(t, [20]byte{0xad, 0x06}, vals[0].Address)
	require.Equal(t, [20]byte{0xad, 0x07}, vals[1].Address)
	require.Equal(t, [20]byte{0xad, 0x04}, vals[2].Address)
	require.Equal(t, [20]byte{0xad, 0x05}, vals[3].Address)
}
