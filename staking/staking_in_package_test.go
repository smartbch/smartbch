package staking

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartbch/moeingevm/types"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/smartbch/moeingads/store"
	"github.com/smartbch/moeingads/store/rabbit"
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
	tx := types.TxToRun{
		BasicTx: types.BasicTx{
			From: common.Address{},
			To:   common.Address{},
			Gas:  GasOfMinGasPriceOp,
		},
	}
	status, _, gasUsed, _ := handleMinGasPrice(ctx, &tx, false, nil)
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

func TestUpdateOnlineInfos(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := types.NewContext(&r, nil)
	ctx.SetCurrentHeight(100)
	ctx.SetStakingForkBlock(90)
	validator1 := [32]byte{0x01}
	validator2 := [32]byte{0x02}
	validators := [][]byte{ed25519.PubKey(validator1[:]).Address(), ed25519.PubKey(validator2[:]).Address()}
	infos := *NewOnlineInfos([]*stakingtypes.Validator{
		{
			Pubkey:      validator1,
			VotingPower: 1,
			StakedCoins: [32]byte{0x1},
		},
		{
			Pubkey:      validator2,
			VotingPower: 1,
			StakedCoins: [32]byte{0x1},
		},
	}, 100)
	startHeight := UpdateOnlineInfos(ctx, infos, validators)
	require.Equal(t, int64(100), startHeight)
	infosLoaded := LoadOnlineInfo(ctx)
	require.Equal(t, int64(100), infosLoaded.StartHeight)
	require.Equal(t, 2, len(infosLoaded.OnlineInfos))
	for i, info := range infosLoaded.OnlineInfos {
		require.Equal(t, crypto.Address(validators[i]).String(), crypto.Address(info.ValidatorConsensusAddress[:]).String())
		require.Equal(t, int32(1), info.SignatureCount)
		require.Equal(t, int64(100), info.HeightOfLastSignature)
	}

	// validator0 mint 1 block but validator1 not sign
	ctx.SetCurrentHeight(101)
	startHeight = UpdateOnlineInfos(ctx, infosLoaded, [][]byte{validators[0]})
	require.Equal(t, int64(100), startHeight)
	infosLoaded = LoadOnlineInfo(ctx)
	require.Equal(t, int64(100), infosLoaded.StartHeight)
	require.Equal(t, 2, len(infosLoaded.OnlineInfos))
	for i, info := range infosLoaded.OnlineInfos {
		require.Equal(t, crypto.Address(validators[i]).String(), crypto.Address(info.ValidatorConsensusAddress[:]).String())
	}
	require.Equal(t, int32(2), infosLoaded.OnlineInfos[0].SignatureCount)
	require.Equal(t, int64(101), infosLoaded.OnlineInfos[0].HeightOfLastSignature)
	require.Equal(t, int32(1), infosLoaded.OnlineInfos[1].SignatureCount)
	require.Equal(t, int64(100), infosLoaded.OnlineInfos[1].HeightOfLastSignature)

	// meet the 500 block
	ctx.SetCurrentHeight(600)
	infosLoaded.OnlineInfos[0].SignatureCount = 450
	SaveOnlineInfo(ctx, infosLoaded)
	startHeight = UpdateOnlineInfos(ctx, infosLoaded, [][]byte{validators[0]})
	require.Equal(t, int64(600), startHeight)
	infosLoaded = LoadOnlineInfo(ctx)
	require.Equal(t, int64(600), infosLoaded.StartHeight)
	require.Equal(t, 2, len(infosLoaded.OnlineInfos))
	for i, info := range infosLoaded.OnlineInfos {
		require.Equal(t, crypto.Address(validators[i]).String(), crypto.Address(info.ValidatorConsensusAddress[:]).String())
	}
	require.Equal(t, int32(1), infosLoaded.OnlineInfos[0].SignatureCount)
	require.Equal(t, int32(0), infosLoaded.OnlineInfos[1].SignatureCount)
	require.Equal(t, int64(600), infosLoaded.OnlineInfos[0].HeightOfLastSignature)
	require.Equal(t, int64(100), infosLoaded.OnlineInfos[1].HeightOfLastSignature)
}

func BuildAndSaveStakingInfo(ctx *types.Context, validatorPubkeys [][32]byte) [][]byte {
	info := stakingtypes.StakingInfo{
		GenesisMainnetBlockHeight: 1,
		CurrEpochNum:              1,
	}
	var validatorAddresses [][]byte
	for _, v := range validatorPubkeys {
		info.Validators = append(info.Validators, &stakingtypes.Validator{
			Pubkey:      v,
			VotingPower: 1,
			StakedCoins: [32]byte{0x1},
		})
		validatorAddresses = append(validatorAddresses, ed25519.PubKey(v[:]).Address())
	}
	SaveStakingInfo(ctx, info)
	return validatorAddresses
}

func TestHandleOnlineInfos(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := types.NewContext(&r, nil)
	ctx.SetCurrentHeight(100)
	ctx.SetStakingForkBlock(90)
	validator1 := [32]byte{0x01}
	validator2 := [32]byte{0x02}
	validators := BuildAndSaveStakingInfo(ctx, [][32]byte{validator1, validator2})
	stakingInfo := LoadStakingInfo(ctx)
	slashValidators := HandleOnlineInfos(ctx, &stakingInfo, validators)
	require.Equal(t, 0, len(slashValidators))
	infosLoaded := LoadOnlineInfo(ctx)
	SaveStakingInfo(ctx, stakingInfo)
	require.Equal(t, int64(100), infosLoaded.StartHeight)
	require.Equal(t, 2, len(infosLoaded.OnlineInfos))
	for i, info := range infosLoaded.OnlineInfos {
		require.Equal(t, crypto.Address(validators[i]).String(), crypto.Address(info.ValidatorConsensusAddress[:]).String())
		require.Equal(t, int32(1), info.SignatureCount)
		require.Equal(t, int64(100), info.HeightOfLastSignature)
	}

	ctx.SetCurrentHeight(101)
	stakingInfo = LoadStakingInfo(ctx)
	slashValidators = HandleOnlineInfos(ctx, &stakingInfo, validators)
	SaveStakingInfo(ctx, stakingInfo)
	require.Equal(t, 0, len(slashValidators))
	infosLoaded = LoadOnlineInfo(ctx)
	require.Equal(t, int64(100), infosLoaded.StartHeight)
	require.Equal(t, 2, len(infosLoaded.OnlineInfos))
	for i, info := range infosLoaded.OnlineInfos {
		require.Equal(t, crypto.Address(validators[i]).String(), crypto.Address(info.ValidatorConsensusAddress[:]).String())
		require.Equal(t, int32(2), info.SignatureCount)
		require.Equal(t, int64(101), info.HeightOfLastSignature)
	}

	infosLoaded = LoadOnlineInfo(ctx)
	infosLoaded.OnlineInfos[0].SignatureCount = 450
	SaveOnlineInfo(ctx, infosLoaded)

	ctx.SetCurrentHeight(600)
	stakingInfo = LoadStakingInfo(ctx)
	slashValidators = HandleOnlineInfos(ctx, &stakingInfo, validators)
	SaveStakingInfo(ctx, stakingInfo)
	require.Equal(t, 1, len(slashValidators))
	infosLoaded = LoadOnlineInfo(ctx)
	require.Equal(t, int64(600), infosLoaded.StartHeight)
	require.Equal(t, 1, len(infosLoaded.OnlineInfos))

	require.Equal(t, crypto.Address(validators[0]).String(), crypto.Address(infosLoaded.OnlineInfos[0].ValidatorConsensusAddress[:]).String())
	require.Equal(t, int32(1), infosLoaded.OnlineInfos[0].SignatureCount)
	require.Equal(t, int64(600), infosLoaded.OnlineInfos[0].HeightOfLastSignature)
	require.Equal(t, crypto.Address(validators[1]).String(), crypto.Address(slashValidators[0][:]).String())

	stakingInfo = LoadStakingInfo(ctx)
	require.Equal(t, int64(0), stakingInfo.Validators[1].VotingPower)
	require.Equal(t, true, stakingInfo.Validators[1].IsRetiring)
}
