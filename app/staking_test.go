package app_test

import (
	"bytes"
	"fmt"
	"testing"

	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/staking"
	"github.com/smartbch/smartbch/staking/types"
)

func TestStaking(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	defer _app.Destroy()

	//config test param
	_init, _min := staking.InitialStakingAmount, staking.MinimumStakingAmount
	defer func() {
		staking.InitialStakingAmount = _init
		staking.MinimumStakingAmount = _min
	}()
	staking.InitialStakingAmount = uint256.NewInt().SetUint64(1)
	staking.MinimumStakingAmount = uint256.NewInt().SetUint64(0)

	//test create validator through deliver tx
	ctx := _app.GetRunTxContext()
	stakingAcc, info := staking.LoadStakingAccAndInfo(ctx)
	ctx.Close(false)
	fmt.Printf("before test:%d, %d\n", stakingAcc.Balance().Uint64(), info.CurrEpochNum)
	dataEncode := staking.PackCreateValidator(addr1, [32]byte{'a'}, [32]byte{'1'})
	_app.MakeAndExecTxInBlockWithGasPrice(key1,
		staking.StakingContractAddress, 100, dataEncode, 1)
	_app.WaitMS(50)
	ctx = _app.GetRunTxContext()
	stakingAcc, info = staking.LoadStakingAccAndInfo(ctx)
	ctx.Close(false)
	require.Equal(t, 100+staking.GasOfStakingExternalOp*1 /*gasUsedFee distribute to validators*/ +600000 /*extra gas*/, stakingAcc.Balance().Uint64())
	require.Equal(t, 2, len(info.Validators))
	require.True(t, bytes.Equal(addr1[:], info.Validators[1].Address[:]))
	require.Equal(t, [32]byte{'1'}, info.Validators[1].Pubkey)
	require.Equal(t, uint64(100), uint256.NewInt().SetBytes(info.Validators[1].StakedCoins[:]).Uint64())

	//test edit validator
	dataEncode = staking.PackEditValidator([20]byte{'b'}, [32]byte{'2'})
	_app.MakeAndExecTxInBlockWithGasPrice(key1,
		staking.StakingContractAddress, 0, dataEncode, 1)
	_app.WaitMS(50)
	ctx = _app.GetRunTxContext()
	_, info = staking.LoadStakingAccAndInfo(ctx)
	ctx.Close(false)
	require.Equal(t, 2, len(info.Validators))
	var intro [32]byte
	copy(intro[:], info.Validators[1].Introduction)
	require.Equal(t, [32]byte{'2'}, intro)
	var SInfo types.StakingInfo
	rCtx := _app.GetRpcContext()
	bz := rCtx.GetStorageAt(staking.StakingContractSequence, staking.SlotStakingInfo)
	_, err := SInfo.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	require.Equal(t, "2", info.Validators[1].Introduction)
	rCtx.Close(false)

	//test change minGasPrice
	ctx = _app.GetRunTxContext()
	staking.SaveMinGasPrice(ctx, 100, true)
	staking.SaveMinGasPrice(ctx, 100, false)
	info = staking.LoadStakingInfo(ctx)
	info.Validators[1].StakedCoins = staking.MinimumStakingAmount.Bytes32()
	info.Validators[1].VotingPower = 1000
	staking.SaveStakingInfo(ctx, info)
	ctx.Close(true)
	dataEncode = staking.PackIncreaseMinGasPrice()
	_app.MakeAndExecTxInBlockWithGasPrice(key1,
		staking.StakingContractAddress, 0, dataEncode, 1)
	_app.WaitMS(50)
	ctx = _app.GetRunTxContext()
	mp := staking.LoadMinGasPrice(ctx, false)
	ctx.Close(false)
	require.Equal(t, 105, int(mp))

	//test validator retire
	ctx = _app.GetRunTxContext()
	staking.SaveMinGasPrice(ctx, 0, true)
	staking.SaveMinGasPrice(ctx, 0, false)
	ctx.Close(true)
	_app.ExecTxInBlock(nil)
	_app.WaitMS(50)

	dataEncode = staking.PackRetire()
	_app.MakeAndExecTxInBlockWithGasPrice(key1,
		staking.StakingContractAddress, 0, dataEncode, 1)
	_app.WaitMS(50)
	ctx = _app.GetRunTxContext()
	_, info = staking.LoadStakingAccAndInfo(ctx)
	ctx.Close(false)
	require.Equal(t, 2, len(info.Validators))
	require.Equal(t, true, info.Validators[1].IsRetiring)
	rCtx = _app.GetRpcContext()
	bz = rCtx.GetStorageAt(staking.StakingContractSequence, staking.SlotStakingInfo)
	_, err = SInfo.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	require.Equal(t, true, info.Validators[1].IsRetiring)
	rCtx.Close(false)

	// test switchEpoch
	e := &types.Epoch{
		Nominations: make([]*types.Nomination, 0, 10),
	}
	var pubkey [32]byte
	copy(pubkey[:], _app.GetTestPubkey().Bytes())
	e.Nominations = append(e.Nominations, &types.Nomination{
		Pubkey:         pubkey,
		NominatedCount: 2,
	})
	_app.EpochChan() <- e
	_app.ExecTxInBlock(nil)
	ctx = _app.GetRunTxContext()
	_, info = staking.LoadStakingAccAndInfo(ctx)
	ctx.Close(false)
	require.Equal(t, 2, len(info.Validators))
	require.Equal(t, int64(1), info.Validators[0].VotingPower)
}

func TestStaking_InvalidSelector(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()

	data := []byte{1, 2, 3}
	tx, _ := _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 0, data)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", staking.InvalidCallData.Error())

	data = []byte{1, 2, 3, 4}
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 0, data)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", staking.InvalidSelector.Error())
}

func TestCreateValidator(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()

	_init, _min := staking.InitialStakingAmount, staking.MinimumStakingAmount
	defer func() {
		staking.InitialStakingAmount = _init
		staking.MinimumStakingAmount = _min
	}()
	staking.InitialStakingAmount = uint256.NewInt().SetUint64(2000)
	staking.MinimumStakingAmount = uint256.NewInt().SetUint64(1000)

	data := make([]byte, 50)
	copy(data, staking.SelectorCreateValidator[:])
	tx, _ := _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 0, data)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", staking.InvalidCallData.Error())

	data = staking.PackCreateValidator(addr1, [32]byte{}, [32]byte{})
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 123, data)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", staking.CreateValidatorCoinLtInitAmount.Error())

	vals := _app.GetValidatorsInfo()
	require.Len(t, vals.Validators, 1)
	data = staking.PackCreateValidator(addr1, [32]byte{}, vals.Validators[0].Pubkey)
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 2001, data)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", types.ValidatorPubkeyAlreadyExists.Error())

	data = staking.PackCreateValidator(addr1, [32]byte{}, [32]byte{'p', 'b', 'k', '1'})
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 2001, data)
	_app.EnsureTxSuccess(tx.Hash())

	data = staking.PackCreateValidator(addr1, [32]byte{}, [32]byte{'p', 'b', 'k', '1'})
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 2001, data)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", types.ValidatorAddressAlreadyExists.Error())

	vals = _app.GetValidatorsInfo()
	require.Len(t, vals.CurrValidators, 1)
	require.Len(t, vals.Validators, 2)
	require.Equal(t, addr1, vals.Validators[1].Address)
}

//func TestStakingUpdate(t *testing.T) {
//	key1, addr1 := testutils.GenKeyAndAddr()
//	key2, _ := testutils.GenKeyAndAddr()
//	_app := testutils.CreateTestApp(key1, key2)
//	defer _app.Destroy()
//
//	//config test param
//	staking.InitialStakingAmount = uint256.NewInt().SetUint64(1)
//	staking.MinimumStakingAmount = uint256.NewInt().SetUint64(0)
//
//	dataEncode := staking.PackCreateValidator(addr1, [32]byte{'a'}, [32]byte{'1'})
//	_app.MakeAndExecTxInBlockWithGasPrice(key1,
//		staking.StakingContractAddress, 100, dataEncode, 1)
//	_app.WaitMS(50)
//
//	ctx := _app.GetContext(app.RunTxMode)
//	acc, info := staking.LoadStakingAccAndInfo(ctx)
//	info.Validators[1].VotingPower = 2
//	staking.SaveStakingInfo(ctx, info)
//	ctx.Close(true)
//
//	_app.AddTxsInBlock(_app.BlockNum() + 1)
//
//	require.Equal(t, 1, len(_app.App.ValidatorUpdate()))
//	require.Equal(t, addr1, common.Address(_app.App.ValidatorUpdate()[0].Address))
//
//	res := _app.EndBlock(abcitypes.RequestEndBlock{})
//	require.Equal(t, 1, len(res.ValidatorUpdates))
//	require.Equal(t, 2, int(res.ValidatorUpdates[0].Power))
//
//	dataEncode = staking.PackRetire()()
//	_app.MakeAndExecTxInBlockWithGasPrice(key1,
//		staking.StakingContractAddress, 100, dataEncode, 1)
//	_app.WaitMS(50)
//
//	require.Equal(t, 1, len(_app.App.ValidatorUpdate()))
//	require.Equal(t, addr1, common.Address(_app.App.ValidatorUpdate()[0].Address))
//	require.Equal(t, int64(0), _app.App.ValidatorUpdate()[0].VotingPower)
//
//	ctx = _app.GetRunTxContext()
//	_, info = staking.LoadStakingAccAndInfo(ctx)
//	require.Equal(t, 2, len(info.Validators))
//	require.Equal(t, int64(1), info.Validators[0].VotingPower)
//	require.Equal(t, true, info.Validators[1].IsRetiring)
//	require.Equal(t, addr1, common.Address(info.ValidatorsUpdate[0].Address))
//	ctx.Close(false)
//}

func TestCallStakingMethodsFromContract(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key1)
	defer _app.Destroy()

	// see testdata/staking/contracts/StakingTest2
	proxyCreationBytecode := testutils.HexToBytes(`
6080604052348015600f57600080fd5b50606980601d6000396000f3fe608060
405260006127109050604051366000823760008036836000865af13d80600084
3e8160008114602f578184f35b8184fdfea26469706673582212204b0d75d505
e5ecaa37fb0567c5e1d65e9b415ac736394100f34def27956650f764736f6c63
430008000033
`)

	_, _, contractAddr := _app.DeployContractInBlock(key1, proxyCreationBytecode)
	require.NotEmpty(t, _app.GetCode(contractAddr))

	intro := [32]byte{'i', 'n', 't', 'r', 'o'}
	pubKey := [32]byte{'p', 'u', 'b', 'k', 'e', 'y'}
	testCases := [][]byte{
		staking.PackCreateValidator(addr1, intro, pubKey),
		staking.PackEditValidator(addr1, intro),
		staking.PackRetire(),
		staking.PackIncreaseMinGasPrice(),
		staking.ABI.MustPack("decreaseMinGasPrice"),
	}

	for _, testCase := range testCases {
		tx, _ := _app.MakeAndExecTxInBlock(key1, contractAddr, 0, testCase)

		_app.WaitMS(200)
		txQuery := _app.GetTx(tx.Hash())
		require.Equal(t, gethtypes.ReceiptStatusFailed, txQuery.Status)
		require.Equal(t, "revert", txQuery.StatusStr)
	}
}
