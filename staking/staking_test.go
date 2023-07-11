package staking_test

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingads/store"
	"github.com/smartbch/moeingads/store/rabbit"
	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/moeingevm/types"

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
			Gas:  staking.GasOfValidatorOp,
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
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	ctx := _app.GetRunTxContext()
	e := &staking.StakingContractExecutor{}
	e.Init(ctx)

	staking.InitialStakingAmount = uint256.NewInt(0)
	staking.GasOfValidatorOp = 0

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

func TestMinGasPriceAdjust(t *testing.T) {
	key, sender := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	staking.InitialStakingAmount = uint256.NewInt(0)
	staking.GasOfMinGasPriceOp = 0
	staking.GasOfValidatorOp = 0
	ctx := _app.GetRunTxContext()
	ctx.SetXHedgeForkBlock(0)

	e := &staking.StakingContractExecutor{}
	e.Init(ctx)

	target := big.NewInt(1000_000_000)
	tx := types.TxToRun{
		BasicTx: types.BasicTx{
			Data: staking.PackProposal(target),
			Gas:  staking.GasOfValidatorOp,
		},
	}
	now := time.Now().Unix()
	blk := types.BlockInfo{
		Timestamp: now,
	}
	status, _, _, outData := e.Execute(ctx, &blk, &tx)
	require.Equal(t, status, 1)
	require.True(t, bytes.Equal(outData, []byte(staking.NoSuchValidator.Error())))

	tx.BasicTx.Data = staking.PackCreateValidator(sender, [32]byte{1}, [32]byte{2})
	tx.BasicTx.Value[31] = 100
	tx.BasicTx.From = sender
	status, _, _, _ = e.Execute(ctx, &blk, &tx)
	require.Equal(t, status, 0)

	tx.BasicTx.Data = staking.PackProposal(target)
	status, _, _, outData = e.Execute(ctx, &blk, &tx)
	require.Equal(t, status, 1)
	require.True(t, bytes.Equal(outData, []byte(staking.ValidatorNotActive.Error())))

	info := staking.LoadStakingInfo(ctx)

	info.Validators[1].VotingPower = 1
	staking.SaveStakingInfo(ctx, info)

	tx.BasicTx.Data = staking.PackProposal(target)
	status, _, _, _ = e.Execute(ctx, &blk, &tx)
	require.Equal(t, status, 0)
	target1, deadline := staking.LoadProposal(ctx)
	require.Equal(t, target.Uint64(), target1)
	require.Equal(t, now+int64(staking.DefaultProposalDuration), int64(deadline))

	tx.BasicTx.Data = staking.PackProposal(target)
	status, _, _, outData = e.Execute(ctx, &blk, &tx)
	require.Equal(t, staking.StatusFailed, status)
	require.True(t, bytes.Equal(outData, []byte(staking.StillInProposal.Error())))

	voteTarget := target.Add(target, big.NewInt(100))
	tx.BasicTx.Data = staking.PackVote(voteTarget)
	status, _, _, _ = e.Execute(ctx, &blk, &tx)
	require.Equal(t, status, 0)
	tar, votingPower := staking.LoadVote(ctx, sender)
	require.Equal(t, voteTarget.Uint64(), tar)
	require.Equal(t, uint64(1), votingPower)
	voters := staking.GetVoters(ctx)
	require.Equal(t, 1, len(voters))
	require.Equal(t, sender, voters[0])

	tx.BasicTx.Data = staking.PackGetVote(sender)
	status, _, _, outData = e.Execute(ctx, &blk, &tx)
	require.Equal(t, status, 0)
	require.Equal(t, 32, len(outData))
	out := big.Int{}
	out.SetBytes(outData)
	require.Equal(t, target.Uint64(), out.Uint64())

	blk.Timestamp = now + int64(staking.DefaultProposalDuration) + 1
	tx.BasicTx.Data = staking.PackExecuteProposal()
	status, _, _, _ = e.Execute(ctx, &blk, &tx)
	require.Equal(t, status, 0)
	minGasPrice := staking.LoadMinGasPrice(ctx, false)
	require.Equal(t, voteTarget.Uint64(), minGasPrice)
	target1, deadline = staking.LoadProposal(ctx)
	require.Equal(t, uint64(0), target1)
	require.Equal(t, uint64(0), deadline)
	voters = staking.GetVoters(ctx)
	require.Equal(t, 0, len(voters))
	target1, votingPower = staking.LoadVote(ctx, sender)
	require.Equal(t, uint64(0), target1)
	require.Equal(t, uint64(0), votingPower)
}

func TestSwitchEpoch(t *testing.T) {
	key, sender := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	staking.InitialStakingAmount = uint256.NewInt(0)
	staking.GasOfMinGasPriceOp = 0
	staking.GasOfValidatorOp = 0
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
		NominatedCount: 1000,
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
	//clear validator pendingReward for testing clearUselessValidators
	info.PendingRewards = []*types2.PendingReward{proposerReward}
	staking.SaveStakingInfo(ctx, info)
	rewardTo := info.Validators[0].RewardTo
	staking.SwitchEpoch(ctx, e, nil, log.NewNopLogger())
	stakingAcc, info = staking.LoadStakingAccAndInfo(ctx)
	require.Equal(t, uint64(10000/2 /*pending reward not transfer to validator as of EpochCountBeforeRewardMature*/), stakingAcc.Balance().Uint64())
	acc = ctx.GetAccount(sender)
	//if validator retire in current epoch,
	//he can only exit on next epoch when there has no pending reward on his address,
	//otherwise exit on next next epoch, and staking coins return back to rewardTo acc
	require.Equal(t, uint64(9999900), acc.Balance().Uint64())
	rewardAcc := ctx.GetAccount(rewardTo)
	require.Equal(t, uint64(100), rewardAcc.Balance().Uint64())

	staking.SwitchEpoch(ctx, e, nil, log.NewNopLogger())
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

func TestSlashAndReward(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := types.NewContext(&r, nil)
	stakingAcc := types.ZeroAccountInfo()
	balance := uint256.NewInt(0).Mul(uint256.NewInt(1000), uint256.NewInt(staking.Uint64_1e18))
	stakingAcc.UpdateBalance(balance)
	ctx.SetAccount(staking.StakingContractAddress, stakingAcc)
	ctx.SetCurrentHeight(100)
	ctx.SetStakingForkBlock(90)

	validator1 := [32]byte{0x01}
	validator2 := [32]byte{0x02}
	var valAddress1 [20]byte
	var valAddress2 [20]byte
	copy(valAddress1[:], ed25519.PubKey(validator1[:]).Address().Bytes())
	copy(valAddress2[:], ed25519.PubKey(validator2[:]).Address().Bytes())
	staking.BuildAndSaveStakingInfo(ctx, [][32]byte{validator1, validator2})
	currValidators, newValidators, _ := staking.SlashAndReward(ctx, nil, valAddress1, valAddress2, [][]byte{valAddress1[:], valAddress2[:]}, nil)
	require.Equal(t, 2, len(currValidators))
	require.Equal(t, 2, len(newValidators))
	onlineInfos := staking.LoadOnlineInfo(ctx)
	require.Equal(t, int64(100), onlineInfos.StartHeight)
	require.Equal(t, 2, len(onlineInfos.OnlineInfos))
	require.Equal(t, valAddress1, onlineInfos.OnlineInfos[0].ValidatorConsensusAddress)

	ctx.SetCurrentHeight(600)
	currValidators, newValidators, _ = staking.SlashAndReward(ctx, nil, valAddress1, valAddress2, [][]byte{valAddress1[:], valAddress2[:]}, nil)
	require.Equal(t, 2, len(currValidators))
	require.Equal(t, 0, len(newValidators))
	onlineInfos = staking.LoadOnlineInfo(ctx)
	require.Equal(t, int64(600), onlineInfos.StartHeight)
	require.Equal(t, 0, len(onlineInfos.OnlineInfos))

	stakingAccLoaded := ctx.GetAccount(staking.StakingContractAddress)
	require.Equal(t, stakingAcc.Balance().String(), uint256.NewInt(0).Add(stakingAccLoaded.Balance(), uint256.NewInt(0).Mul(uint256.NewInt(20), uint256.NewInt(staking.Uint64_1e18))).String())
	blackHoleBalance := ebp.GetBlackHoleBalance(ctx)
	require.Equal(t, blackHoleBalance, uint256.NewInt(0).Mul(uint256.NewInt(20), uint256.NewInt(staking.Uint64_1e18)))
}

func TestLoadEpoch(t *testing.T) {
	key, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	ctx := _app.GetRunTxContext()

	epoch, ok := staking.LoadEpoch(ctx, 0)
	require.Equal(t, int64(0), epoch.StartHeight)
	require.Equal(t, false, ok)

	epoch = types2.Epoch{Number: 1, StartHeight: 10}
	staking.SaveEpoch(ctx, &epoch)
	epoch, ok = staking.LoadEpoch(ctx, 1)
	require.Equal(t, int64(10), epoch.StartHeight)
	require.Equal(t, true, ok)
}

func TestInvalidExecuteProposal(t *testing.T) {
	key, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	staking.InitialStakingAmount = uint256.NewInt(0)
	staking.GasOfMinGasPriceOp = 0
	staking.GasOfValidatorOp = 0
	ctx := _app.GetRunTxContext()
	ctx.SetXHedgeForkBlock(0)

	e := &staking.StakingContractExecutor{}
	e.Init(ctx)

	tx := types.TxToRun{
		BasicTx: types.BasicTx{
			Data: staking.PackExecuteProposal(),
			Gas:  staking.GasOfValidatorOp,
		},
	}
	now := time.Now().Unix()
	blk := types.BlockInfo{
		Timestamp: now,
	}
	status, _, _, outData := e.Execute(ctx, &blk, &tx)
	require.Equal(t, staking.StatusFailed, status)
	require.True(t, bytes.Equal(outData, []byte(staking.NotInProposal.Error())))

	staking.SaveProposal(ctx, 100, uint64(now+10000))
	status, _, _, outData = e.Execute(ctx, &blk, &tx)
	require.Equal(t, staking.StatusFailed, status)
	require.True(t, bytes.Equal(outData, []byte(staking.ProposalNotFinished.Error())))
}

func TestInvalidVote(t *testing.T) {
	key, sender := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	staking.InitialStakingAmount = uint256.NewInt(0)
	staking.GasOfMinGasPriceOp = 0
	staking.GasOfValidatorOp = 0
	ctx := _app.GetRunTxContext()
	ctx.SetXHedgeForkBlock(0)

	e := &staking.StakingContractExecutor{}
	e.Init(ctx)

	target := big.NewInt(100)
	tx := types.TxToRun{
		BasicTx: types.BasicTx{
			Data: staking.PackVote(target),
			Gas:  staking.GasOfValidatorOp,
		},
	}
	now := time.Now().Unix()

	blk := types.BlockInfo{
		Timestamp: now,
	}
	status, _, _, outData := e.Execute(ctx, &blk, &tx)
	require.Equal(t, staking.StatusFailed, status)
	require.True(t, bytes.Equal(outData, []byte(staking.NoSuchValidator.Error())))

	tx.BasicTx.Data = staking.PackCreateValidator(sender, [32]byte{1}, [32]byte{2})
	tx.BasicTx.Value[31] = 100
	tx.BasicTx.From = sender
	status, _, _, _ = e.Execute(ctx, &blk, &tx)
	require.Equal(t, status, 0)

	tx.BasicTx.Data = staking.PackVote(target)
	status, _, _, outData = e.Execute(ctx, &blk, &tx)
	require.Equal(t, staking.StatusFailed, status)
	require.True(t, bytes.Equal(outData, []byte(staking.ValidatorNotActive.Error())))

	info := staking.LoadStakingInfo(ctx)

	info.Validators[1].VotingPower = 1
	staking.SaveStakingInfo(ctx, info)

	status, _, _, outData = e.Execute(ctx, &blk, &tx)
	require.Equal(t, staking.StatusFailed, status)
	require.True(t, bytes.Equal(outData, []byte(staking.NotInProposal.Error())))

	staking.SaveProposal(ctx, 100, uint64(now-1000))
	status, _, _, outData = e.Execute(ctx, &blk, &tx)
	require.Equal(t, staking.StatusFailed, status)
	require.True(t, bytes.Equal(outData, []byte(staking.ProposalHasFinished.Error())))

	staking.SaveProposal(ctx, 100, uint64(now+2000))
	tx.BasicTx.Data = staking.PackVote(big.NewInt(0))
	staking.SaveMinGasPrice(ctx, staking.MinGasPriceLowerBound-1, true)
	status, _, _, outData = e.Execute(ctx, &blk, &tx)
	require.Equal(t, staking.StatusFailed, status)
	require.True(t, bytes.Equal(outData, []byte(staking.TargetExceedChangeDelta.Error())))
}
