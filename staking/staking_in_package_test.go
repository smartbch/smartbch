package staking

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartbch/moeingevm/types"
	"testing"

	"github.com/stretchr/testify/require"
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
