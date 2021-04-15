package app

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"
	"time"

	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/staking"
	"github.com/smartbch/smartbch/staking/types"
)

var stakingABI = testutils.MustParseABI(`
[
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "rewardTo",
				"type": "address"
			},
			{
				"internalType": "bytes32",
				"name": "introduction",
				"type": "bytes32"
			},
			{
				"internalType": "bytes32",
				"name": "pubkey",
				"type": "bytes32"
			}
		],
		"name": "createValidator",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "decreaseMinGasPrice",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "rewardTo",
				"type": "address"
			},
			{
				"internalType": "bytes32",
				"name": "introduction",
				"type": "bytes32"
			}
		],
		"name": "editValidator",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "increaseMinGasPrice",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "retire",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]
`)

func TestStaking(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, _ := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key1, key2)
	defer DestroyTestApp(_app)

	//config test param
	staking.InitialStakingAmount = uint256.NewInt().SetUint64(1)
	staking.MinimumStakingAmount = uint256.NewInt().SetUint64(0)

	//test create validator through deliver tx
	ctx := _app.GetContext(RunTxMode)
	stakingAcc, info := staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	fmt.Printf("before test:%d\n", stakingAcc.Balance().Uint64())
	dataEncode := stakingABI.MustPack("createValidator", addr1, [32]byte{'a'}, [32]byte{'1'})
	tx := gethtypes.NewTransaction(0, staking.StakingContractAddress, big.NewInt(100), 1000000, big.NewInt(1), dataEncode)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	testutils.ExecTxInBlock(_app, 1, tx)
	time.Sleep(50 * time.Millisecond)
	ctx = _app.GetContext(RunTxMode)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	require.Equal(t, uint64(100+staking.GasOfStakingExternalOp*1 /*gasUsedFee distribute to validators*/ +600000 /*extra gas*/), stakingAcc.Balance().Uint64())
	require.Equal(t, 2, len(info.Validators))
	require.True(t, bytes.Equal(addr1[:], info.Validators[1].Address[:]))
	require.Equal(t, [32]byte{'1'}, info.Validators[1].Pubkey)
	require.Equal(t, uint64(100), uint256.NewInt().SetBytes(info.Validators[1].StakedCoins[:]).Uint64())

	//test edit validator
	dataEncode = stakingABI.MustPack("editValidator", [32]byte{'b'}, [32]byte{'2'})
	tx = gethtypes.NewTransaction(1, staking.StakingContractAddress, big.NewInt(0), 400000, big.NewInt(1), dataEncode)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	testutils.ExecTxInBlock(_app, 3, tx)
	time.Sleep(50 * time.Millisecond)
	ctx = _app.GetContext(RunTxMode)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	require.Equal(t, 2, len(info.Validators))
	var intro [32]byte
	copy(intro[:], info.Validators[1].Introduction)
	require.Equal(t, [32]byte{'2'}, intro)

	//test change minGasPrice
	ctx = _app.GetContext(RunTxMode)
	staking.SaveMinGasPrice(ctx, 100, true)
	staking.SaveMinGasPrice(ctx, 100, false)
	acc, info := staking.LoadStakingAcc(*ctx)
	info.Validators[1].StakedCoins = staking.MinimumStakingAmount.Bytes32()
	info.Validators[1].VotingPower = 1000
	staking.SaveStakingInfo(*ctx, acc, info)
	ctx.Close(true)
	dataEncode = stakingABI.MustPack("increaseMinGasPrice")
	tx = gethtypes.NewTransaction(2, staking.StakingContractAddress, big.NewInt(0), 400000, big.NewInt(1), dataEncode)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	testutils.ExecTxInBlock(_app, 5, tx)
	time.Sleep(50 * time.Millisecond)
	ctx = _app.GetContext(RunTxMode)
	mp := staking.LoadMinGasPrice(ctx, false)
	ctx.Close(false)
	require.Equal(t, 105, int(mp))

	//test validator retire
	ctx = _app.GetContext(RunTxMode)
	staking.SaveMinGasPrice(ctx, 0, true)
	staking.SaveMinGasPrice(ctx, 0, false)
	ctx.Close(true)
	testutils.ExecTxInBlock(_app, 7, nil)
	time.Sleep(50 * time.Millisecond)

	dataEncode = stakingABI.MustPack("retire")
	tx = gethtypes.NewTransaction(3, staking.StakingContractAddress, big.NewInt(0), 400000, big.NewInt(0), dataEncode)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key1))
	testutils.ExecTxInBlock(_app, 9, tx)
	time.Sleep(50 * time.Millisecond)
	ctx = _app.GetContext(RunTxMode)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	require.Equal(t, 2, len(info.Validators))
	require.Equal(t, true, info.Validators[1].IsRetiring)

	// test switchEpoch
	e := &types.Epoch{
		ValMapByPubkey: make(map[[32]byte]*types.Nomination),
	}
	var pubkey [32]byte
	copy(pubkey[:], _app.testValidatorPubKey.Bytes())
	e.ValMapByPubkey[pubkey] = &types.Nomination{
		Pubkey:         pubkey,
		NominatedCount: 2,
	}
	_app.watcher.EpochChan <- e
	testutils.ExecTxInBlock(_app, 11, nil)
	ctx = _app.GetContext(RunTxMode)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	require.Equal(t, 1, len(info.Validators))
	require.Equal(t, int64(2), info.Validators[0].VotingPower)
}
