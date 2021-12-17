package app_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/stretchr/testify/require"

	"github.com/smartbch/smartbch/internal/bigutils"
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
	staking.InitialStakingAmount = uint256.NewInt(1)
	staking.MinimumStakingAmount = uint256.NewInt(0)

	//test create validator through deliver tx
	ctx := _app.GetRunTxContext()
	stakingAcc, info := staking.LoadStakingAccAndInfo(ctx)
	ctx.Close(false)
	fmt.Printf("before test:%d, %d\n", stakingAcc.Balance().Uint64(), info.CurrEpochNum)
	dataEncode := staking.PackCreateValidator(addr1, [32]byte{'a'}, [32]byte{'1'})
	_app.MakeAndExecTxInBlockWithGas(key1,
		staking.StakingContractAddress, 100, dataEncode, testutils.DefaultGasLimit, 1)
	_app.WaitMS(50)
	ctx = _app.GetRunTxContext()
	stakingAcc, info = staking.LoadStakingAccAndInfo(ctx)
	ctx.Close(false)
	require.Equal(t, 100+staking.GasOfValidatorOp*1/2 /*gasUsedFee distribute to validators*/ +600000/2 /*extra gas*/, stakingAcc.Balance().Uint64())
	require.Equal(t, 2, len(info.Validators))
	require.True(t, bytes.Equal(addr1[:], info.Validators[1].Address[:]))
	require.Equal(t, [32]byte{'1'}, info.Validators[1].Pubkey)
	require.Equal(t, uint64(100), uint256.NewInt(0).SetBytes(info.Validators[1].StakedCoins[:]).Uint64())

	//test edit validator
	dataEncode = staking.PackEditValidator([20]byte{'b'}, [32]byte{'2'})
	_app.MakeAndExecTxInBlockWithGas(key1,
		staking.StakingContractAddress, 0, dataEncode, testutils.DefaultGasLimit, 1)
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
	_app.MakeAndExecTxInBlockWithGas(key1,
		staking.StakingContractAddress, 0, dataEncode, testutils.DefaultGasLimit, 1)
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
	_app.MakeAndExecTxInBlockWithGas(key1,
		staking.StakingContractAddress, 0, dataEncode, testutils.DefaultGasLimit, 1)
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
	_app.AddEpochForTest(e)
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

	_initAmt := staking.InitialStakingAmount
	defer func() { staking.InitialStakingAmount = _initAmt }()
	staking.InitialStakingAmount = uint256.NewInt(2000)

	data := make([]byte, 90)
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

	data = staking.PackCreateValidator(addr1, [32]byte{'h', 'e', 'l', 'l', 'o'}, [32]byte{'p', 'b', 'k', '1'})
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 2001, data)
	_app.EnsureTxSuccess(tx.Hash())

	data = staking.PackCreateValidator(addr1, [32]byte{}, [32]byte{'p', 'b', 'k', '1'})
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 2001, data)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", types.ValidatorAddressAlreadyExists.Error())

	vals = _app.GetValidatorsInfo()
	require.Len(t, vals.CurrValidators, 1)
	require.Len(t, vals.Validators, 2)
	require.Equal(t, addr1, vals.Validators[1].Address)
	require.Equal(t, addr1, vals.Validators[1].RewardTo)
	require.Equal(t, big.NewInt(2001), big.NewInt(0).SetBytes(vals.Validators[1].StakedCoins[:]))
	require.Equal(t, "hello", vals.Validators[1].Introduction)
	require.Equal(t, false, vals.Validators[1].IsRetiring)
}

func TestEditValidator(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()

	_initAmt := staking.InitialStakingAmount
	defer func() { staking.InitialStakingAmount = _initAmt }()
	staking.InitialStakingAmount = uint256.NewInt(2000)

	data := make([]byte, 60)
	copy(data, staking.SelectorEditValidator[:])
	tx, _ := _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 0, data)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", staking.InvalidCallData.Error())

	data = staking.PackEditValidator(addr1, [32]byte{})
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 123, data)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", staking.NoSuchValidator.Error())

	data = staking.PackCreateValidator(addr1, [32]byte{}, [32]byte{'p', 'b', 'k', '1'})
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 2001, data)
	_app.EnsureTxSuccess(tx.Hash())

	data = staking.PackEditValidator(addr2, [32]byte{'i', 'n', 't', 'r', 'o'})
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 456, data)
	_app.EnsureTxSuccess(tx.Hash())

	vals := _app.GetValidatorsInfo()
	require.Len(t, vals.CurrValidators, 1)
	require.Len(t, vals.Validators, 2)
	require.Equal(t, addr1, vals.Validators[1].Address)
	require.Equal(t, addr2, vals.Validators[1].RewardTo)
	require.Equal(t, big.NewInt(2457), big.NewInt(0).SetBytes(vals.Validators[1].StakedCoins[:]))
	require.Equal(t, "intro", vals.Validators[1].Introduction)
	require.Equal(t, false, vals.Validators[1].IsRetiring)
}

func TestRetireValidator(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()

	_initAmt := staking.InitialStakingAmount
	defer func() { staking.InitialStakingAmount = _initAmt }()
	staking.InitialStakingAmount = uint256.NewInt(2000)

	data := staking.PackRetire()
	tx, _ := _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 123, data)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", staking.NoSuchValidator.Error())

	data = staking.PackCreateValidator(addr1, [32]byte{}, [32]byte{'p', 'b', 'k', '1'})
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 2001, data)
	_app.EnsureTxSuccess(tx.Hash())

	data = staking.PackRetire()
	tx, _ = _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 0, data)
	_app.EnsureTxSuccess(tx.Hash())

	vals := _app.GetValidatorsInfo()
	require.Len(t, vals.CurrValidators, 1)
	require.Len(t, vals.Validators, 2)
	require.Equal(t, true, vals.Validators[1].IsRetiring)
}

func TestUpdateMinGasPrice(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()

	require.Equal(t, uint64(0), _app.GetMinGasPrice(true))
	require.Equal(t, uint64(0), _app.GetMinGasPrice(false))

	data := staking.PackIncreaseMinGasPrice()
	tx, _ := _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 123, data)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", staking.OperatorNotValidator.Error())
}

//func TestStakingUpdate(t *testing.T) {
//	key1, addr1 := testutils.GenKeyAndAddr()
//	key2, _ := testutils.GenKeyAndAddr()
//	_app := testutils.CreateTestApp(key1, key2)
//	defer _app.Destroy()
//
//	//config test param
//	staking.InitialStakingAmount = uint256.NewInt(1)
//	staking.MinimumStakingAmount = uint256.NewInt(0)
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
		staking.PackDecreaseMinGasPrice(),
		//staking.PackSumVotingPower([]common.Address{addr1}),
	}

	for _, testCase := range testCases {
		tx, _ := _app.MakeAndExecTxInBlock(key1, contractAddr, 0, testCase)

		_app.WaitMS(200)
		txQuery := _app.GetTx(tx.Hash())
		require.Equal(t, gethtypes.ReceiptStatusFailed, txQuery.Status)
		require.Equal(t, "revert", txQuery.StatusStr)
	}
}

func TestSumVotingPower(t *testing.T) {
	_initAmt := staking.InitialStakingAmount
	_minAmt := staking.MinimumStakingAmount
	defer func() {
		staking.InitialStakingAmount = _initAmt
		staking.MinimumStakingAmount = _minAmt
	}()
	staking.InitialStakingAmount = uint256.NewInt(2000)
	staking.MinimumStakingAmount = uint256.NewInt(2000)

	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	key3, addr3 := testutils.GenKeyAndAddr()
	key4, addr4 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2, key3, key4)
	defer _app.Destroy()

	data := staking.PackCreateValidator(addr1, [32]byte{'v', '1'}, [32]byte{'p', '1'})
	tx, _ := _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 2001, data)
	_app.EnsureTxSuccess(tx.Hash())

	data = staking.PackCreateValidator(addr2, [32]byte{'v', '2'}, [32]byte{'p', '2'})
	tx, _ = _app.MakeAndExecTxInBlock(key2, staking.StakingContractAddress, 2001, data)
	_app.EnsureTxSuccess(tx.Hash())

	data = staking.PackCreateValidator(addr3, [32]byte{'v', '3'}, [32]byte{'p', '3'})
	tx, _ = _app.MakeAndExecTxInBlock(key3, staking.StakingContractAddress, 2001, data)
	_app.EnsureTxSuccess(tx.Hash())

	_app.AddEpochForTest(&types.Epoch{
		Nominations: []*types.Nomination{
			{Pubkey: [32]byte{'p', '1'}, NominatedCount: 300},
			{Pubkey: [32]byte{'p', '2'}, NominatedCount: 400},
			{Pubkey: [32]byte{'p', '3'}, NominatedCount: 500},
		},
	})
	_app.ExecTxsInBlock()

	vals := _app.GetValidatorsInfo()
	require.Len(t, vals.Validators, 4)
	require.Len(t, vals.CurrValidators, 3)
	require.Equal(t, int64(0), vals.Validators[0].VotingPower)
	require.Equal(t, int64(1), vals.Validators[1].VotingPower)
	require.Equal(t, int64(1), vals.Validators[2].VotingPower)
	require.Equal(t, int64(1), vals.Validators[3].VotingPower)

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

	// call sumVotingPower
	data = staking.PackSumVotingPower([]gethcmn.Address{addr1})
	tx, _ = _app.MakeAndExecTxInBlock(key1, contractAddr, 0, data)
	_app.EnsureTxSuccess(tx.Hash())

	data = staking.PackSumVotingPower([]gethcmn.Address{addr1})
	statusCode, statusStr, outData := _app.Call(addr1, contractAddr, data)
	require.Equal(t, "success", statusStr)
	require.Equal(t, 0, statusCode)
	summedPower, totalPower := staking.UnpackSumVotingPowerReturnData(outData)
	require.Equal(t, "1", summedPower.String())
	require.Equal(t, "3", totalPower.String())

	data = staking.PackSumVotingPower([]gethcmn.Address{addr1, addr2})
	statusCode, statusStr, outData = _app.Call(addr1, contractAddr, data)
	require.Equal(t, "success", statusStr)
	require.Equal(t, 0, statusCode)
	summedPower, totalPower = staking.UnpackSumVotingPowerReturnData(outData)
	require.Equal(t, "2", summedPower.String())
	require.Equal(t, "3", totalPower.String())

	data = staking.PackSumVotingPower([]gethcmn.Address{addr1, addr2, addr3})
	statusCode, statusStr, outData = _app.Call(addr1, contractAddr, data)
	require.Equal(t, "success", statusStr)
	require.Equal(t, 0, statusCode)
	summedPower, totalPower = staking.UnpackSumVotingPowerReturnData(outData)
	require.Equal(t, "3", summedPower.String())
	require.Equal(t, "3", totalPower.String())

	data = staking.PackSumVotingPower([]gethcmn.Address{addr1, addr2, addr3, addr4})
	statusCode, statusStr, outData = _app.Call(addr1, contractAddr, data)
	require.Equal(t, "success", statusStr)
	require.Equal(t, 0, statusCode)
	summedPower, totalPower = staking.UnpackSumVotingPowerReturnData(outData)
	require.Equal(t, "3", summedPower.String())
	require.Equal(t, "3", totalPower.String())
}

func TestSumVotingPowerFromEOA(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()

	data := staking.PackSumVotingPower([]gethcmn.Address{addr1})
	tx, _ := _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 0, data)
	_app.EnsureTxFailed(tx.Hash(), "failure")

	statusCode, statusStr, _ := _app.Call(addr1, staking.StakingContractAddress, data)
	require.Equal(t, "failure", statusStr)
	require.Equal(t, 1, statusCode)
}

func TestStakingDetermination(t *testing.T) {
	_initAmt := staking.InitialStakingAmount
	_minAmt := staking.MinimumStakingAmount
	defer func() {
		staking.InitialStakingAmount = _initAmt
		staking.MinimumStakingAmount = _minAmt
	}()
	staking.InitialStakingAmount = uint256.NewInt(2000)
	staking.MinimumStakingAmount = uint256.NewInt(2000)

	valPubKey := ed25519.GenPrivKey().PubKey()
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	key3, addr3 := testutils.GenKeyAndAddr()

	startTime := time.Now()
	var stateRoot []byte
	for i := 0; i < 5; i++ {
		//println("----------")
		_app := testutils.CreateTestApp0(startTime, valPubKey,
			bigutils.NewU256(testutils.DefaultInitBalance), key1, key2, key3)

		var pubKey0 [32]byte
		copy(pubKey0[:], _app.TestPubkey.Bytes())
		pubKey1 := [32]byte{'p', 'b', 'k', '1'}
		pubKey2 := [32]byte{'p', 'b', 'k', '2'}
		pubKey3 := [32]byte{'p', 'b', 'k', '3'}

		data := staking.PackCreateValidator(addr1, [32]byte{'v', 'a', 'l', '1'}, pubKey1)
		tx, _ := _app.MakeAndExecTxInBlock(key1, staking.StakingContractAddress, 2001, data)
		_app.EnsureTxSuccess(tx.Hash())

		data = staking.PackCreateValidator(addr2, [32]byte{'v', 'a', 'l', '2'}, pubKey2)
		tx, _ = _app.MakeAndExecTxInBlock(key2, staking.StakingContractAddress, 2001, data)
		_app.EnsureTxSuccess(tx.Hash())

		data = staking.PackCreateValidator(addr3, [32]byte{'v', 'a', 'l', '3'}, pubKey3)
		tx, _ = _app.MakeAndExecTxInBlock(key3, staking.StakingContractAddress, 2001, data)
		_app.EnsureTxSuccess(tx.Hash())

		vals := _app.GetValidatorsInfo()
		require.Len(t, vals.Validators, 4)
		require.Len(t, vals.CurrValidators, 1)
		require.Equal(t, int64(1), vals.Validators[0].VotingPower)
		require.Equal(t, int64(0), vals.Validators[1].VotingPower)
		require.Equal(t, int64(0), vals.Validators[2].VotingPower)
		require.Equal(t, int64(0), vals.Validators[3].VotingPower)

		_app.AddEpochForTest(&types.Epoch{
			Nominations: []*types.Nomination{
				{Pubkey: pubKey0, NominatedCount: 300},
				{Pubkey: pubKey1, NominatedCount: 400},
				{Pubkey: pubKey2, NominatedCount: 500},
				{Pubkey: pubKey3, NominatedCount: 200},
			},
		})
		_app.ExecTxsInBlock()

		vals = _app.GetValidatorsInfo()
		require.Len(t, vals.Validators, 4)
		require.Len(t, vals.CurrValidators, 4)
		require.Equal(t, int64(1), vals.Validators[0].VotingPower)
		require.Equal(t, int64(1), vals.Validators[1].VotingPower)
		require.Equal(t, int64(1), vals.Validators[2].VotingPower)
		require.Equal(t, int64(1), vals.Validators[3].VotingPower)

		data = staking.PackEditValidator(addr3, [32]byte{'V', 'A', 'L', '3'})
		tx, _ = _app.MakeAndExecTxInBlock(key3, staking.StakingContractAddress, 3000, data)
		_app.EnsureTxSuccess(tx.Hash())

		data = staking.PackRetire()
		tx, _ = _app.MakeAndExecTxInBlock(key2, staking.StakingContractAddress, 0, data)
		_app.EnsureTxSuccess(tx.Hash())

		vals = _app.GetValidatorsInfo()
		require.Len(t, vals.Validators, 4)
		require.Len(t, vals.CurrValidators, 3)
		require.Equal(t, int64(1), vals.Validators[0].VotingPower)
		require.Equal(t, int64(1), vals.Validators[1].VotingPower)
		require.Equal(t, int64(1), vals.Validators[3].VotingPower)

		_app.AddEpochForTest(&types.Epoch{
			Nominations: []*types.Nomination{
				{Pubkey: pubKey0, NominatedCount: 100},
				{Pubkey: pubKey1, NominatedCount: 200},
				{Pubkey: pubKey2, NominatedCount: 300},
				{Pubkey: pubKey3, NominatedCount: 400},
			},
		})
		_app.ExecTxsInBlock()

		vals = _app.GetValidatorsInfo()
		require.Len(t, vals.Validators, 4)
		require.Len(t, vals.CurrValidators, 3)
		require.Equal(t, int64(1), vals.Validators[0].VotingPower)
		require.Equal(t, int64(1), vals.Validators[1].VotingPower)
		require.Equal(t, int64(1), vals.Validators[3].VotingPower)

		_app.Destroy()
		if i == 0 {
			stateRoot = _app.StateRoot
		} else {
			//println(hex.EncodeToString(stateRoot))
			require.Equal(t, hex.EncodeToString(stateRoot),
				hex.EncodeToString(_app.StateRoot))
		}
	}
}
