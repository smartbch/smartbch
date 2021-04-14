package staking_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/testutils"
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

func TestStaking(t *testing.T) {
	key, sender := testutils.GenKeyAndAddr()
	_app := app.CreateTestApp(key)
	defer app.DestroyTestApp(_app)
	ctx := _app.GetContext(app.RunTxMode)
	e := &staking.StakingContractExecutor{}
	e.Init(ctx)

	staking.InitialStakingAmount = uint256.NewInt().SetUint64(0)

	// test create validator
	c := buildCreateValCallEntry(sender, 101, 11, 1)
	require.True(t, e.IsSystemContract(c.Address))
	e.Execute(*ctx, nil, c.Tx)
	stakingAcc, info := staking.LoadStakingAcc(*ctx)
	require.Equal(t, 1+1 /*include app.testValidatorPubKey*/, len(info.Validators))
	require.True(t, bytes.Equal(sender.Bytes(), info.Validators[1].Address[:]))
	require.Equal(t, 11, int(info.Validators[1].Introduction[0]))
	require.Equal(t, uint64(100), stakingAcc.Balance().Uint64())

	// invalid create call
	c.Tx.Data = c.Tx.Data[:95]
	status, _, _, outData := e.Execute(*ctx, nil, c.Tx)
	require.Equal(t, types.ReceiptStatusFailed, uint64(status))
	require.Equal(t, staking.InvalidCallData.Error(), string(outData[:]))

	//invalid selector
	c.Tx.Data = c.Tx.Data[:3]
	e.Execute(*ctx, nil, c.Tx)
	require.Equal(t, types.ReceiptStatusFailed, uint64(status))

	// test edit validator
	c = buildEditValCallEntry(sender, 102, 12)
	e.Execute(*ctx, nil, c.Tx)
	_, info = staking.LoadStakingAcc(*ctx)
	require.Equal(t, 12, int(info.Validators[1].Introduction[0]))
	require.Equal(t, 200, int(info.Validators[1].StakedCoins[31]))

	// test retire validator
	c = buildRetireValCallEntry(sender)
	e.Execute(*ctx, nil, c.Tx)
	_, info = staking.LoadStakingAcc(*ctx)
	require.True(t, info.Validators[1].IsRetiring)
}

func TestSwitchEpoch(t *testing.T) {
	key, sender := testutils.GenKeyAndAddr()
	//key, addr1 := testutils.GenKeyAndAddr()
	_app := app.CreateTestApp(key)
	defer app.DestroyTestApp(_app)
	staking.InitialStakingAmount = uint256.NewInt().SetUint64(0)
	ctx := _app.GetContext(app.RunTxMode)
	//build new epoch
	e := &types2.Epoch{
		StartHeight:    100,
		EndTime:        2000,
		Duration:       1000,
		ValMapByPubkey: make(map[[32]byte]*types2.Nomination),
	}
	var pubkey [32]byte
	copy(pubkey[:], _app.TestValidatorPubkey().Bytes())
	e.ValMapByPubkey[pubkey] = &types2.Nomination{
		Pubkey:         pubkey,
		NominatedCount: 1,
	}
	staking.MinimumStakingAmount = uint256.NewInt().SetUint64(0)
	//add another validator
	exe := &staking.StakingContractExecutor{}
	exe.Init(ctx)
	c := buildCreateValCallEntry(sender, 101, 11, 1)
	exe.Execute(*ctx, nil, c.Tx)
	stakingAcc, info := staking.LoadStakingAcc(*ctx)
	require.Equal(t, 1+1 /*include app.testValidatorPubKey*/, len(info.Validators))
	require.True(t, bytes.Equal(sender.Bytes(), info.Validators[1].Address[:]))
	acc := ctx.GetAccount(sender)
	require.Equal(t, uint64(9999900), acc.Balance().Uint64())
	//retire it for clear
	c = buildRetireValCallEntry(sender)
	exe.Execute(*ctx, nil, c.Tx)
	//test distribute
	info.Validators[0].VotingPower = 1
	info.Validators[1].VotingPower = 1
	staking.SaveStakingInfo(*ctx, stakingAcc, info)
	collectedFee := uint256.NewInt().SetUint64(10000)
	voters := make([][32]byte, 2)
	voters[0] = pubkey
	voters[1] = info.Validators[1].Pubkey
	staking.DistributeFee(*ctx, collectedFee, pubkey, voters)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	require.Equal(t, uint64((10000-1500-8500*15/100)/2), uint256.NewInt().SetBytes32(info.PendingRewards[1].Amount[:]).Uint64())
	require.Equal(t, uint64(10000-(10000-1500-8500*15/100)/2), uint256.NewInt().SetBytes32(info.PendingRewards[0].Amount[:]).Uint64())
	require.Equal(t, uint64(10100), stakingAcc.Balance().Uint64())
	//clear validator pendingReward for testing clearUp
	info.PendingRewards = info.PendingRewards[:1]
	staking.SaveStakingInfo(*ctx, stakingAcc, info)
	rewardTo := info.Validators[0].RewardTo
	staking.SwitchEpoch(ctx, e)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	require.Equal(t, uint64(10000 /*pending reward not transfer to validator as of EpochCountBeforeRewardMature*/), stakingAcc.Balance().Uint64())
	acc = ctx.GetAccount(sender)
	//if validator retire in current epoch,
	//he can only exit on next epoch when there has no pending reward on his address,
	//otherwise exit on next next epoch, and staking coins return back to rewardTo acc
	require.Equal(t, uint64(9999900), acc.Balance().Uint64())
	rewardAcc := ctx.GetAccount(rewardTo)
	require.Equal(t, uint64(100), rewardAcc.Balance().Uint64())

	staking.SwitchEpoch(ctx, e)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	require.Equal(t, uint64((10000-1500-8500*15/100)/2), stakingAcc.Balance().Uint64())
}

func TestSlash(t *testing.T) {
	key, _ := testutils.GenKeyAndAddr()
	_app := app.CreateTestApp(key)
	defer app.DestroyTestApp(_app)
	ctx := _app.GetContext(app.RunTxMode)
	var slashedPubkey [32]byte
	copy(slashedPubkey[:], _app.TestValidatorPubkey().Bytes())
	stakingAddr := common.Address{}
	copy(stakingAddr[:], _app.TestValidatorPubkey().Address())
	ctx.SetAccount(stakingAddr, types.ZeroAccountInfo())
	stakingAcc, info := staking.LoadStakingAcc(*ctx)
	info.Validators[0].StakedCoins[31] = 100
	staking.SaveStakingInfo(*ctx, stakingAcc, info)
	totalSlashed := staking.Slash(ctx, slashedPubkey, uint256.NewInt().SetUint64(1))
	require.Equal(t, uint64(1), totalSlashed.Uint64())
	allBurnt := uint256.NewInt()
	acc := ctx.GetAccount(stakingAddr)
	bz := ctx.GetStorageAt(acc.Sequence(), staking.SlotAllBurnt)
	if len(bz) != 0 {
		allBurnt.SetBytes32(bz)
	}
	require.Equal(t, uint64(1), allBurnt.Uint64())
}
