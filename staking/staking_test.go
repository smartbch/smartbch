package staking_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/staking"
	types2 "github.com/smartbch/smartbch/staking/types"
)

type callEntry struct {
	Address [20]byte
	Tx      *types.TxToRun
}

func buildCreateValCallEntry(sender common.Address, rewardTo byte, introduction byte, pubkey byte) *callEntry {
	c := &callEntry{
		Address: staking.StakingContractAddress,
		Tx:      nil,
	}
	c.Tx = &types.TxToRun{
		BasicTx: types.BasicTx{
			From: sender,
			To:   c.Address,
		},
	}
	c.Tx.Value[31] = 100
	// createValidator(address rewardTo, bytes32 introduction, bytes32 pubkey)
	// data: (4B selector | 32B rewardTo | 32B intro | 32B pubkey)
	c.Tx.Data = make([]byte, 0, 100)
	c.Tx.Data = append(c.Tx.Data, staking.SelectorCreateValidator[:]...)
	r := [32]byte{rewardTo}
	c.Tx.Data = append(c.Tx.Data, r[:]...)
	i := [32]byte{introduction}
	c.Tx.Data = append(c.Tx.Data, i[:]...)
	p := [32]byte{pubkey}
	c.Tx.Data = append(c.Tx.Data, p[:]...)
	return c
}

func buildEditValCallEntry(sender common.Address, rewardTo byte, introduction byte) *callEntry {
	c := &callEntry{
		Address: staking.StakingContractAddress,
		Tx:      nil,
	}
	c.Tx = &types.TxToRun{
		BasicTx: types.BasicTx{
			From: sender,
			To:   c.Address,
		},
	}
	c.Tx.Value[31] = 100
	// editValidator(address rewardTo, bytes32 introduction)
	// data: (4B selector | 32B rewardTo | 32B intro)
	c.Tx.Data = make([]byte, 0, 100)
	c.Tx.Data = append(c.Tx.Data, staking.SelectorEditValidator[:]...)
	r := [32]byte{rewardTo}
	c.Tx.Data = append(c.Tx.Data, r[:]...)
	i := [32]byte{introduction}
	c.Tx.Data = append(c.Tx.Data, i[:]...)
	return c
}

func buildRetireValCallEntry(sender common.Address) *callEntry {
	c := &callEntry{
		Address: staking.StakingContractAddress,
		Tx:      nil,
	}
	c.Tx = &types.TxToRun{
		BasicTx: types.BasicTx{
			From: sender,
			To:   c.Address,
		},
	}
	// retire()
	// data: (4B selector)
	c.Tx.Data = make([]byte, 0, 100)
	c.Tx.Data = append(c.Tx.Data, staking.SelectorRetire[:]...)
	return c
}

func buildChangeMinGasPriceCallEntry(sender common.Address, isIncrease bool) *callEntry {
	c := &callEntry{
		Address: staking.StakingContractAddress,
		Tx:      nil,
	}
	c.Tx = &types.TxToRun{
		BasicTx: types.BasicTx{
			From: sender,
			To:   c.Address,
		},
	}
	c.Tx.Data = make([]byte, 0, 100)
	if isIncrease {
		c.Tx.Data = append(c.Tx.Data, staking.SelectorIncreaseMinGasPrice[:]...)
	} else {
		c.Tx.Data = append(c.Tx.Data, staking.SelectorDecreaseMinGasPrice[:]...)
	}
	return c
}

func TestStaking(t *testing.T) {
	key, sender := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	ctx := _app.GetRunTxContext()
	e := &staking.StakingContractExecutor{}
	e.Init(ctx)

	staking.InitialStakingAmount = uint256.NewInt(0)

	// test create validator
	c := buildCreateValCallEntry(sender, 101, 11, 1)
	require.True(t, e.IsSystemContract(c.Address))
	e.Execute(ctx, nil, c.Tx)
	stakingAcc, info := staking.LoadStakingAccAndInfo(ctx)
	require.Equal(t, 1+1 /*include app.testValidatorPubKey*/, len(info.Validators))
	require.True(t, bytes.Equal(sender.Bytes(), info.Validators[1].Address[:]))
	require.Equal(t, 11, int(info.Validators[1].Introduction[0]))
	require.Equal(t, uint64(100), stakingAcc.Balance().Uint64())

	// invalid create call
	c.Tx.Data = c.Tx.Data[:95]
	status, _, _, outData := e.Execute(ctx, nil, c.Tx)
	require.Equal(t, staking.StatusFailed, status)
	require.Equal(t, staking.InvalidCallData.Error(), string(outData))

	//invalid selector
	c.Tx.Data = c.Tx.Data[:3]
	e.Execute(ctx, nil, c.Tx)
	require.Equal(t, staking.StatusFailed, status)

	// test edit validator
	c = buildEditValCallEntry(sender, 102, 12)
	e.Execute(ctx, nil, c.Tx)
	_, info = staking.LoadStakingAccAndInfo(ctx)
	require.Equal(t, 12, int(info.Validators[1].Introduction[0]))
	require.Equal(t, 200, int(info.Validators[1].StakedCoins[31]))

	// test retire validator
	c = buildRetireValCallEntry(sender)
	e.Execute(ctx, nil, c.Tx)
	_, info = staking.LoadStakingAccAndInfo(ctx)
	require.True(t, info.Validators[1].IsRetiring)
}

func TestSwitchEpoch(t *testing.T) {
	key, sender := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	staking.InitialStakingAmount = uint256.NewInt(0)
	ctx := _app.GetRunTxContext()
	//build new epoch
	e := &types2.Epoch{
		StartHeight: 100,
		EndTime:     2000,
		Nominations: make([]*types2.Nomination, 0, 10),
	}
	var pubkey [32]byte
	copy(pubkey[:], _app.GetTestPubkey().Bytes())
	e.Nominations = append(e.Nominations, &types2.Nomination{
		Pubkey:         pubkey,
		NominatedCount: 1,
	})
	staking.MinimumStakingAmount = uint256.NewInt(0)
	//add another validator
	exe := &staking.StakingContractExecutor{}
	exe.Init(ctx)
	c := buildCreateValCallEntry(sender, 101, 11, 1)
	exe.Execute(ctx, nil, c.Tx)
	_, info := staking.LoadStakingAccAndInfo(ctx)
	require.Equal(t, 1+1 /*include app.testValidatorPubKey*/, len(info.Validators))
	require.True(t, bytes.Equal(sender.Bytes(), info.Validators[1].Address[:]))
	acc := ctx.GetAccount(sender)
	require.Equal(t, uint64(9999900), acc.Balance().Uint64())
	//retire it for clear
	c = buildRetireValCallEntry(sender)
	exe.Execute(ctx, nil, c.Tx)
	//test distribute
	info.Validators[0].VotingPower = 1
	info.Validators[1].VotingPower = 1
	staking.SaveStakingInfo(ctx, info)
	collectedFee := uint256.NewInt(10000)
	voters := make([][32]byte, 2)
	voters[0] = pubkey
	voters[1] = info.Validators[1].Pubkey
	stakingAcc, info := staking.LoadStakingAccAndInfo(ctx)
	staking.DistributeFee(ctx, stakingAcc, &info, collectedFee, pubkey, pubkey, voters)

	var voterReward *types2.PendingReward
	var proposerReward *types2.PendingReward
	if info.PendingRewards[1].Address == info.Validators[1].Address {
		voterReward = info.PendingRewards[1]
		proposerReward = info.PendingRewards[0]
	} else {
		voterReward = info.PendingRewards[0]
		proposerReward = info.PendingRewards[1]
	}
	require.Equal(t, uint64((10000-1500-8500*15/100)/2/2), uint256.NewInt(0).SetBytes32(voterReward.Amount[:]).Uint64())
	require.Equal(t, uint64(10000/2-(10000-1500-8500*15/100)/2/2), uint256.NewInt(0).SetBytes32(proposerReward.Amount[:]).Uint64())
	require.Equal(t, uint64(10100), stakingAcc.Balance().Uint64())
	//clear validator pendingReward for testing clearUp
	info.PendingRewards = []*types2.PendingReward{proposerReward}
	staking.SaveStakingInfo(ctx, info)
	rewardTo := info.Validators[0].RewardTo
	staking.SwitchEpoch(ctx, e, log.NewNopLogger(), 0, param.StakingMinVotingPubKeysPercentPerEpoch)
	stakingAcc, info = staking.LoadStakingAccAndInfo(ctx)
	require.Equal(t, uint64(10000/2 /*pending reward not transfer to validator as of EpochCountBeforeRewardMature*/), stakingAcc.Balance().Uint64())
	acc = ctx.GetAccount(sender)
	//if validator retire in current epoch,
	//he can only exit on next epoch when there has no pending reward on his address,
	//otherwise exit on next next epoch, and staking coins return back to rewardTo acc
	require.Equal(t, uint64(9999900), acc.Balance().Uint64())
	rewardAcc := ctx.GetAccount(rewardTo)
	require.Equal(t, uint64(100), rewardAcc.Balance().Uint64())

	staking.SwitchEpoch(ctx, e, log.NewNopLogger(), 0, param.StakingMinVotingPubKeysPercentPerEpoch)
	stakingAcc, info = staking.LoadStakingAccAndInfo(ctx)
	require.Equal(t, uint64((10000-1500-8500*15/100)/2/2), stakingAcc.Balance().Uint64())
}

func TestSlash(t *testing.T) {
	key, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	ctx := _app.GetRunTxContext()
	var slashedPubkey [32]byte
	copy(slashedPubkey[:], _app.GetTestPubkey().Bytes())
	stakingAddr := common.Address{}
	copy(stakingAddr[:], _app.GetTestPubkey().Address())
	ctx.SetAccount(stakingAddr, types.ZeroAccountInfo())
	info := staking.LoadStakingInfo(ctx)
	info.Validators[0].StakedCoins[31] = 100
	staking.SaveStakingInfo(ctx, info)
	totalSlashed := staking.Slash(ctx, &info, slashedPubkey, uint256.NewInt(1))
	require.Equal(t, uint64(1), totalSlashed.Uint64())
	allBurnt := uint256.NewInt(0)
	bz := ctx.GetStorageAt(staking.StakingContractSequence, staking.SlotAllBurnt)
	if len(bz) != 0 {
		allBurnt.SetBytes32(bz)
	}
	require.Equal(t, uint64(1), allBurnt.Uint64())
}

func TestGasPriceAdjustment(t *testing.T) {
	key, sender := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	staking.DefaultMinGasPrice = 100
	staking.MinGasPriceUpperBound = 500
	ctx := _app.GetRunTxContext()
	e := staking.NewStakingContractExecutor(log.NewNopLogger())
	e.Init(ctx)
	staking.SaveMinGasPrice(ctx, staking.DefaultMinGasPrice, true)
	staking.InitialStakingAmount = uint256.NewInt(0)

	//create validator
	c := buildCreateValCallEntry(sender, 101, 11, 1)
	require.True(t, e.IsSystemContract(c.Address))
	e.Execute(ctx, nil, c.Tx)

	//increase gasPrice failed as of validator not active
	c = buildChangeMinGasPriceCallEntry(sender, true)
	_, _, _, out := e.Execute(ctx, nil, c.Tx)
	p := staking.LoadMinGasPrice(ctx, false)
	require.Equal(t, 100, int(p))
	require.True(t, bytes.Equal([]byte(staking.OperatorNotValidator.Error()), out))

	//make validator active
	info := staking.LoadStakingInfo(ctx)
	info.Validators[1].StakedCoins = staking.MinimumStakingAmount.Bytes32()
	info.Validators[1].VotingPower = 1000
	staking.SaveStakingInfo(ctx, info)

	//increase gasPrice
	c = buildChangeMinGasPriceCallEntry(sender, true)
	e.Execute(ctx, nil, c.Tx)
	p = staking.LoadMinGasPrice(ctx, false)
	require.Equal(t, 105, int(p))

	//increase gasPrice
	e.Execute(ctx, nil, c.Tx)
	p = staking.LoadMinGasPrice(ctx, false)
	require.Equal(t, 110, int(p))

	//increase gasPrice
	e.Execute(ctx, nil, c.Tx)
	p = staking.LoadMinGasPrice(ctx, false)
	require.Equal(t, 115, int(p))

	//increase gasPrice failed because out of range
	_, _, _, out = e.Execute(ctx, nil, c.Tx)
	p = staking.LoadMinGasPrice(ctx, false)
	pLast := staking.LoadMinGasPrice(ctx, true)
	require.Equal(t, 100, int(pLast))
	require.Equal(t, 115, int(p))
	require.True(t, bytes.Equal([]byte(staking.MinGasPriceExceedBlockChangeDelta.Error()), out))

	//decrease gasPrice
	c = buildChangeMinGasPriceCallEntry(sender, false)
	e.Execute(ctx, nil, c.Tx)
	p = staking.LoadMinGasPrice(ctx, false)
	require.Equal(t, 110, int(p))
}
