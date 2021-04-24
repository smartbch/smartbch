package app_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/smartbch/smartbch/app"
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
	_app := testutils.CreateTestApp(key1, key2)
	defer testutils.DestroyTestApp(_app)

	//config test param
	staking.InitialStakingAmount = uint256.NewInt().SetUint64(1)
	staking.MinimumStakingAmount = uint256.NewInt().SetUint64(0)

	//test create validator through deliver tx
	ctx := _app.GetContext(app.RunTxMode)
	stakingAcc, info := staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	fmt.Printf("before test:%d\n", stakingAcc.Balance().Uint64())
	dataEncode := stakingABI.MustPack("createValidator", addr1, [32]byte{'a'}, [32]byte{'1'})
	testutils.MakeAndExecTxInBlockWithGasPrice(_app, 1, key1, 0,
		staking.StakingContractAddress, 100, dataEncode, 1)
	time.Sleep(50 * time.Millisecond)
	ctx = _app.GetContext(app.RunTxMode)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	require.Equal(t, uint64(100+staking.GasOfStakingExternalOp*1 /*gasUsedFee distribute to validators*/ +600000 /*extra gas*/), stakingAcc.Balance().Uint64())
	require.Equal(t, 2, len(info.Validators))
	require.True(t, bytes.Equal(addr1[:], info.Validators[1].Address[:]))
	require.Equal(t, [32]byte{'1'}, info.Validators[1].Pubkey)
	require.Equal(t, uint64(100), uint256.NewInt().SetBytes(info.Validators[1].StakedCoins[:]).Uint64())

	//test edit validator
	dataEncode = stakingABI.MustPack("editValidator", [32]byte{'b'}, [32]byte{'2'})
	testutils.MakeAndExecTxInBlockWithGasPrice(_app, 3, key1, 1,
		staking.StakingContractAddress, 0, dataEncode, 1)
	time.Sleep(50 * time.Millisecond)
	ctx = _app.GetContext(app.RunTxMode)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	require.Equal(t, 2, len(info.Validators))
	var intro [32]byte
	copy(intro[:], info.Validators[1].Introduction)
	require.Equal(t, [32]byte{'2'}, intro)

	//test change minGasPrice
	ctx = _app.GetContext(app.RunTxMode)
	staking.SaveMinGasPrice(ctx, 100, true)
	staking.SaveMinGasPrice(ctx, 100, false)
	acc, info := staking.LoadStakingAcc(*ctx)
	info.Validators[1].StakedCoins = staking.MinimumStakingAmount.Bytes32()
	info.Validators[1].VotingPower = 1000
	staking.SaveStakingInfo(*ctx, acc, info)
	ctx.Close(true)
	dataEncode = stakingABI.MustPack("increaseMinGasPrice")
	testutils.MakeAndExecTxInBlockWithGasPrice(_app, 5, key1, 2,
		staking.StakingContractAddress, 0, dataEncode, 1)
	time.Sleep(50 * time.Millisecond)
	ctx = _app.GetContext(app.RunTxMode)
	mp := staking.LoadMinGasPrice(ctx, false)
	ctx.Close(false)
	require.Equal(t, 105, int(mp))

	//test validator retire
	ctx = _app.GetContext(app.RunTxMode)
	staking.SaveMinGasPrice(ctx, 0, true)
	staking.SaveMinGasPrice(ctx, 0, false)
	ctx.Close(true)
	testutils.ExecTxInBlock(_app, 7, nil)
	time.Sleep(50 * time.Millisecond)

	dataEncode = stakingABI.MustPack("retire")
	testutils.MakeAndExecTxInBlockWithGasPrice(_app, 9, key1, 3,
		staking.StakingContractAddress, 0, dataEncode, 1)
	time.Sleep(50 * time.Millisecond)
	ctx = _app.GetContext(app.RunTxMode)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	require.Equal(t, 2, len(info.Validators))
	require.Equal(t, true, info.Validators[1].IsRetiring)

	// test switchEpoch
	e := &types.Epoch{
		ValMapByPubkey: make(map[[32]byte]*types.Nomination),
	}
	var pubkey [32]byte
	copy(pubkey[:], _app.TestValidatorPubkey().Bytes())
	e.ValMapByPubkey[pubkey] = &types.Nomination{
		Pubkey:         pubkey,
		NominatedCount: 2,
	}
	_app.EpochChan() <- e
	testutils.ExecTxInBlock(_app, 11, nil)
	ctx = _app.GetContext(app.RunTxMode)
	stakingAcc, info = staking.LoadStakingAcc(*ctx)
	ctx.Close(false)
	require.Equal(t, 1, len(info.Validators))
	require.Equal(t, int64(2), info.Validators[0].VotingPower)
}

func TestCallStakingMethodsFromEOA(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key1)
	defer testutils.DestroyTestApp(_app)

	intro := [32]byte{'i', 'n', 't', 'r', 'o'}
	pubKey := [32]byte{'p', 'u', 'b', 'k', 'e', 'y'}
	stakingAddr := gethcmn.HexToAddress("0x0000000000000000000000000000000000002710")

	testCases := [][]byte{
		stakingABI.MustPack("createValidator", addr1, intro, pubKey),
		stakingABI.MustPack("editValidator", addr1, intro),
		stakingABI.MustPack("retire"),
		stakingABI.MustPack("increaseMinGasPrice"),
		stakingABI.MustPack("decreaseMinGasPrice"),
	}

	for i, testCase := range testCases {
		h := int64(1 + i*2)
		tx := testutils.MakeAndExecTxInBlock(_app, h, key1, uint64(0+i), stakingAddr, 0, testCase)

		blk := testutils.GetBlock(_app, uint64(h))
		require.Equal(t, h, blk.Number)
		require.Len(t, blk.Transactions, 1)
		txInBlk := testutils.GetTx(_app, blk.Transactions[0])
		//require.Equal(t, gethtypes.ReceiptStatusSuccessful, txInBlk.Status)
		require.Equal(t, "success", txInBlk.StatusStr)
		require.Equal(t, tx.Hash(), gethcmn.Hash(txInBlk.Hash))

		var info types.StakingInfo
		ctx := _app.GetContext(RpcMode)
		bz := ctx.GetStorageAt(staking.StakingContractSequence, staking.SlotStakingInfo)
		_, err := info.UnmarshalMsg(bz)
		if err != nil {
			panic(err)
		}
		fmt.Printf("info %#v\n", info) //TODO check details
		ctx.Close(false)
	}
}

func TestCallStakingMethodsFromContract(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key1)
	defer testutils.DestroyTestApp(_app)

	// see testdata/staking/contracts/StakingTest2
	proxyCreationBytecode := testutils.HexToBytes(`
6080604052348015600f57600080fd5b50606980601d6000396000f3fe608060
405260006127109050604051366000823760008036836000865af13d80600084
3e8160008114602f578184f35b8184fdfea26469706673582212204b0d75d505
e5ecaa37fb0567c5e1d65e9b415ac736394100f34def27956650f764736f6c63
430008000033
`)

	tx1 := testutils.DeployContractInBlock(_app, 1, key1, 0, proxyCreationBytecode)
	contractAddr := gethcrypto.CreateAddress(addr1, tx1.Nonce())
	code := testutils.GetCode(_app, contractAddr)
	require.True(t, len(code) > 0)

	intro := [32]byte{'i', 'n', 't', 'r', 'o'}
	pubKey := [32]byte{'p', 'u', 'b', 'k', 'e', 'y'}
	testCases := [][]byte{
		stakingABI.MustPack("createValidator", addr1, intro, pubKey),
		stakingABI.MustPack("editValidator", addr1, intro),
		stakingABI.MustPack("retire"),
		stakingABI.MustPack("increaseMinGasPrice"),
		stakingABI.MustPack("decreaseMinGasPrice"),
	}

	for i, testCase := range testCases {
		h := int64(3 + i*2)
		tx := testutils.MakeAndExecTxInBlock(_app, h, key1, uint64(1+i), contractAddr, 0, testCase)

		blk := testutils.GetBlock(_app, uint64(h))
		require.Equal(t, h, blk.Number)
		require.Len(t, blk.Transactions, 1)
		txInBlk := testutils.GetTx(_app, blk.Transactions[0])
		//require.Equal(t, gethtypes.ReceiptStatusSuccessful, txInBlk.Status)
		require.Equal(t, "revert", txInBlk.StatusStr)
		require.Equal(t, tx.Hash(), gethcmn.Hash(txInBlk.Hash))
	}
}
