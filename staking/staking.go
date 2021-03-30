package staking

import (
	"bytes"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	mevmtypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/staking/types"
)

var (
	StakingContractAddress [20]byte = [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		byte('s'), byte('t'), byte('a'), byte('k'), byte('i'), byte('n'), byte('g')}
	InitialStakingAmount *uint256.Int = uint256.NewInt().Mul(
		uint256.NewInt().SetUint64(1000),
		uint256.NewInt().SetUint64(1000_000_000_000_000_000))
	MinimumStakingAmount *uint256.Int = uint256.NewInt().Mul(
		uint256.NewInt().SetUint64(800),
		uint256.NewInt().SetUint64(1000_000_000_000_000_000))

	/*interface Staking {
	    //0x24d1ed5d
	    function createValidator(address rewardTo, bytes32 introduction, bytes32 pubkey) external;
	    //0x9dc159b6
	    function editValidator(address rewardTo, bytes32 introduction) external;
	    //0xa4874d77
	    function retire() external;
	}*/
	SelectorCreateValidator [4]byte = [4]byte{0x24, 0xd1, 0xed, 0x5d}
	SelectorEditValidator   [4]byte = [4]byte{0x9d, 0xc1, 0x59, 0xb6}
	SelectorRetire          [4]byte = [4]byte{0xa4, 0x87, 0x4d, 0x77}

	SlotStakingInfo string = strings.Repeat(string([]byte{0}), 32)
	SlotAllBurnt    string = strings.Repeat(string([]byte{0}), 31) + string([]byte{1})

	EpochCountBeforeRewardMature int64 = 1

	GasOfStakingExternalOp uint64 = 400_000

	BaseProposerPercentage  *uint256.Int = uint256.NewInt().SetUint64(15)
	ExtraProposerPercentage *uint256.Int = uint256.NewInt().SetUint64(15)

	InvalidCallData  = errors.New("Invalid call data")
	BalanceNotEnough = errors.New("Balance is not enough")
	NoSuchValidator  = errors.New("No such validator")
)

type StakingContractExecutor struct{}

var _ mevmtypes.SystemContractExecutor = &StakingContractExecutor{}

func (_ *StakingContractExecutor) Init(ctx *mevmtypes.Context) {
	stakingAcc := ctx.GetAccount(StakingContractAddress)
	if stakingAcc == nil {
		stakingAcc = mevmtypes.ZeroAccountInfo()
		ctx.SetAccount(StakingContractAddress, stakingAcc)
	}
}

func (_ *StakingContractExecutor) IsSystemContract(addr common.Address) bool {
	return bytes.Equal(addr[:], StakingContractAddress[:])
}

// Staking functions which can be invoked through smart contract calls
func (_ *StakingContractExecutor) Execute(ctx mevmtypes.Context, currBlock *mevmtypes.BlockInfo, tx *mevmtypes.TxToRun) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	if len(tx.Data) < 4 {
		status = int(mevmtypes.ReceiptStatusFailed)
		return
	}
	selector := tx.Data[:4]
	if bytes.Equal(selector, SelectorCreateValidator[:]) {
		//createValidator(address rewardTo, bytes32 introduction, bytes32 pubkey)
		return externalOp(ctx, tx, true, false)
	} else if bytes.Equal(selector, SelectorEditValidator[:]) {
		//editValidator(address rewardTo, bytes32 introduction)
		return externalOp(ctx, tx, false, false)
	} else if bytes.Equal(selector, SelectorRetire[:]) {
		//retire()
		return externalOp(ctx, tx, false, true)
	} else {
		status = int(mevmtypes.ReceiptStatusFailed)
	}
	return
}

// This function implements the underlying logic for three external functions: createValidator, editValidator and retire
func externalOp(ctx mevmtypes.Context, tx *mevmtypes.TxToRun, create bool, retire bool) (status int, logs []mevmtypes.EvmLog, gasUsed uint64, outData []byte) {
	status = int(mevmtypes.ReceiptStatusFailed)
	gasUsed = GasOfStakingExternalOp
	var pubkey [32]byte
	var intro string
	var rewardTo [20]byte
	if !retire { // retire has no arguments
		callData := tx.Data[4:]
		if !((create && len(callData) >= 96) || (!create && len(callData) >= 64)) {
			outData = []byte(InvalidCallData.Error())
			return
		}
		// First argument: rewardTo
		copy(rewardTo[:], callData[12:])
		callData = callData[32:]
		// Second argument: introduction, byte32, limited to 32 byte
		intro = string(callData[:32])
		if create {
			// Third argument: pubkey (only createValidator has it)
			callData = callData[32:]
			copy(pubkey[:], callData)
		}
	}

	sender := ctx.GetAccount(tx.From)
	balance := sender.Balance()
	coins4staking := uint256.NewInt().SetBytes32(tx.Value[:])
	if balance.Lt(coins4staking) {
		outData = []byte(BalanceNotEnough.Error())
		return
	}

	stakingAcc, info := LoadStakingAcc(ctx)

	if create { //createValidator
		if uint256.NewInt().SetBytes(tx.Value[:]).Cmp(InitialStakingAmount) <= 0 {
			outData = []byte(types.CreateValidatorCoinLtInitAmount.Error())
			return
		}
		err := info.AddValidator(tx.From, pubkey, intro, tx.Value, rewardTo)
		if err != nil {
			outData = []byte(err.Error())
			return
		}
	} else { // retire or editValidator
		val := info.GetValidatorByAddr(tx.From)
		if val == nil {
			outData = []byte(NoSuchValidator.Error())
			return
		}
		var bz20Zero [20]byte
		if !bytes.Equal(rewardTo[:], bz20Zero[:]) {
			val.RewardTo = rewardTo
		}
		if len(intro) != 0 {
			val.Introduction = intro
		}
		if !coins4staking.IsZero() {
			stakedCoins := uint256.NewInt().SetBytes32(val.StakedCoins[:])
			stakedCoins.Add(stakedCoins, coins4staking)
			val.StakedCoins = stakedCoins.Bytes32()
		}
		if retire {
			val.IsRetiring = true
		}
	}

	// Now let's update the states
	SaveStakingInfo(ctx, stakingAcc, info)

	if !coins4staking.IsZero() {
		balance.Sub(balance, coins4staking)
		sender.UpdateBalance(balance)
		stakingAccBalance := stakingAcc.Balance()
		stakingAccBalance.Add(stakingAccBalance, coins4staking)
		stakingAcc.UpdateBalance(stakingAccBalance)
		ctx.SetAccount(tx.From, sender)
		ctx.SetAccount(StakingContractAddress, stakingAcc)
	}

	status = int(mevmtypes.ReceiptStatusSuccessful)
	return
}

func LoadStakingAcc(ctx mevmtypes.Context) (stakingAcc *mevmtypes.AccountInfo, info types.StakingInfo) {
	stakingAcc = ctx.GetAccount(StakingContractAddress)
	if stakingAcc == nil {
		panic("Cannot find staking contract")
	}
	bz := ctx.GetStorageAt(stakingAcc.Sequence(), SlotStakingInfo)
	if bz == nil {
		return stakingAcc, types.StakingInfo{}
	}
	_, err := info.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	return
}

func SaveStakingInfo(ctx mevmtypes.Context, stakingAcc *mevmtypes.AccountInfo, info types.StakingInfo) {
	bz, err := info.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	ctx.SetStorageAt(stakingAcc.Sequence(), SlotStakingInfo, bz)
}

// =========================================================================================
// Staking functions which cannot be invoked through smart contract calls

// Slash 'amount' of coins from the validator with 'pubkey'. These coins are burnt.
func Slash(ctx mevmtypes.Context, pubkey [32]byte, amount *uint256.Int) (totalSlashed *uint256.Int) {
	stakingAcc, info := LoadStakingAcc(ctx)
	val := info.GetValidatorByPubkey(pubkey)
	if val == nil {
		return // If tendermint works fine, we'll never reach here
	}
	coins := uint256.NewInt().SetBytes32(val.StakedCoins[:])
	if coins.Lt(amount) { // not enough coins to be slashed
		totalSlashed = coins.Clone()
		coins.SetUint64(0)
	} else {
		totalSlashed = amount.Clone()
		coins.Sub(coins, amount)
	}
	val.StakedCoins = coins.Bytes32()

	totalCleared := info.ClearRewardsOf(val.Address)
	totalSlashed.Add(totalSlashed, totalCleared)

	// deduct the totalSlashed from stakingAcc and burn them
	balance := stakingAcc.Balance()
	balance.Sub(balance, totalSlashed)
	stakingAcc.UpdateBalance(balance)
	ctx.SetAccount(StakingContractAddress, stakingAcc)

	incrAllBurnt(ctx, stakingAcc, totalSlashed)
	return
}

// Increase the slot of 'all burnt' inside stakingAcc
func incrAllBurnt(ctx mevmtypes.Context, stakingAcc *mevmtypes.AccountInfo, amount *uint256.Int) {
	allBurnt := uint256.NewInt()
	bz := ctx.GetStorageAt(stakingAcc.Sequence(), SlotAllBurnt)
	if len(bz) != 0 {
		allBurnt.SetBytes32(bz)
	}
	allBurnt.Add(allBurnt, amount)
	bz32 := allBurnt.Bytes32()
	ctx.SetStorageAt(stakingAcc.Sequence(), SlotAllBurnt, bz32[:])
}

// distribute the collected gas fee to validators who voted for current block
func DistributeFee(ctx mevmtypes.Context, collectedFee *uint256.Int, proposer [32]byte /*pubKey*/, voters [][32]byte) {
	if collectedFee == nil {
		return
	}

	// the collected fee is saved as stakingAcc's balance, just as the staked coins
	stakingAcc, info := LoadStakingAcc(ctx)
	stakingAccBalance := stakingAcc.Balance()
	stakingAccBalance.Add(stakingAccBalance, collectedFee)
	stakingAcc.UpdateBalance(stakingAccBalance)
	ctx.SetAccount(StakingContractAddress, stakingAcc)

	totalVotingPower, votedPower := int64(0), int64(0)
	for _, val := range info.GetActiveValidators(MinimumStakingAmount) {
		totalVotingPower += val.VotingPower
	}
	valMapByPubkey := info.GetValMapByPubkey()
	for _, voter := range voters {
		val := valMapByPubkey[voter]
		votedPower += val.VotingPower
	}

	// proposerBaseFee and proposerExtraFee both go to the proposer
	proposerBaseFee := uint256.NewInt().Mul(collectedFee, BaseProposerPercentage)
	proposerBaseFee.Div(proposerBaseFee, uint256.NewInt().SetUint64(100))
	collectedFee.Sub(collectedFee, proposerBaseFee)
	proposerExtraFee := uint256.NewInt().Mul(collectedFee, ExtraProposerPercentage)
	proposerExtraFee.Mul(proposerExtraFee, uint256.NewInt().SetUint64(uint64(votedPower)))
	proposerExtraFee.Div(proposerExtraFee, uint256.NewInt().SetUint64(uint64(100*totalVotingPower)))
	collectedFee.Sub(collectedFee, proposerExtraFee)

	rwdMapByAddr := info.GetCurrRewardMapByAddr()
	remainedFee := collectedFee.Clone()
	//distribute to the non-proposer voters
	for _, voter := range voters {
		if bytes.Equal(proposer[:], voter[:]) {
			continue
		}
		val := valMapByPubkey[voter]
		rwdCoins := uint256.NewInt().Mul(collectedFee, uint256.NewInt().SetUint64(uint64(val.VotingPower)))
		rwdCoins.Div(rwdCoins, uint256.NewInt().SetUint64(uint64(votedPower)))
		remainedFee.Sub(remainedFee, rwdCoins)
		rwd := rwdMapByAddr[val.Address]
		if rwd == nil {
			rwd = &types.PendingReward{
				Address:  val.Address,
				EpochNum: info.CurrEpochNum,
				Amount:   [32]byte{},
			}
			info.PendingRewards = append(info.PendingRewards, rwd)
		}
		coins := uint256.NewInt().SetBytes32(rwd.Amount[:])
		coins.Add(coins, rwdCoins)
		rwd.Amount = coins.Bytes32()
	}

	//distribute to the proposer
	proposerVal := valMapByPubkey[proposer]
	rwd := rwdMapByAddr[proposerVal.Address]
	if rwd == nil {
		rwd = &types.PendingReward{
			Address:  proposerVal.Address,
			EpochNum: info.CurrEpochNum,
			Amount:   [32]byte{},
		}
		info.PendingRewards = append(info.PendingRewards, rwd)
	}
	coins := uint256.NewInt().SetBytes32(rwd.Amount[:])
	coins.Add(coins, proposerBaseFee)
	coins.Add(coins, proposerExtraFee)
	coins.Add(coins, remainedFee)
	rwd.Amount = coins.Bytes32()

	SaveStakingInfo(ctx, stakingAcc, info)
}

// switch to a new epoch
func SwitchEpoch(ctx *mevmtypes.Context, epoch *types.Epoch) []*types.Validator {
	pubkey2power := make(map[[32]byte]int64)
	for _, v := range epoch.ValMapByPubkey {
		pubkey2power[v.Pubkey] = v.NominatedCount
	}
	// distribute mature pending reward to rewardTo
	stakingAcc, info := endEpoch(ctx)
	// someone who call createValidator before switchEpoch can enjoy the voting power update
	// someone who call retire() before switchEpoch missed this update
	updateVotingPower(&info, pubkey2power)
	// payback staking coins to rewardTo of useless validators and delete these validators
	clearUp(ctx, stakingAcc, &info)
	// allocate new entries in info.PendingRewards
	activeValidators := info.GetActiveValidators(MinimumStakingAmount)
	for _, val := range activeValidators {
		pr := &types.PendingReward{
			Address:  val.Address,
			EpochNum: info.CurrEpochNum,
		}
		info.PendingRewards = append(info.PendingRewards, pr)
	}
	SaveStakingInfo(*ctx, stakingAcc, info)
	return activeValidators
}

// deliver pending rewards which are mature now to rewardTo
func endEpoch(ctx *mevmtypes.Context) (stakingAcc *mevmtypes.AccountInfo, info types.StakingInfo) {
	stakingAcc, info = LoadStakingAcc(*ctx)
	info.CurrEpochNum++
	stakingAccBalance := stakingAcc.Balance()

	newPRList := make([]*types.PendingReward, 0, len(info.PendingRewards))
	valMapByAddr := info.GetValMapByAddr()
	rewardMap := make(map[[20]byte]*uint256.Int)
	// summarize all the mature rewards
	for _, pr := range info.PendingRewards {
		if pr.EpochNum >= info.CurrEpochNum-EpochCountBeforeRewardMature {
			newPRList = append(newPRList, pr) //not mature yet
			continue
		}
		val := valMapByAddr[pr.Address]
		if _, ok := rewardMap[val.RewardTo]; !ok {
			rewardMap[val.RewardTo] = uint256.NewInt()
		}
		rewardMap[val.RewardTo].Add(rewardMap[val.RewardTo], uint256.NewInt().SetBytes32(pr.Amount[:]))
	}

	// increase rewardTo's balance and decrease stakingAcc's balance
	for addr, rwd := range rewardMap {
		acc := ctx.GetAccount(addr)
		if acc == nil {
			acc = mevmtypes.ZeroAccountInfo()
		}
		stakingAccBalance.Sub(stakingAccBalance, rwd)
		balance := acc.Balance()
		balance.Add(balance, rwd)
		acc.UpdateBalance(balance)
		ctx.SetAccount(addr, acc)
	}
	stakingAcc.UpdateBalance(stakingAccBalance)
	info.PendingRewards = newPRList
	return
}

// Clear the old voting powers and assign pubkey2power to validators.
func updateVotingPower(info *types.StakingInfo, pubkey2power map[[32]byte]int64) {
	for _, val := range info.Validators {
		val.VotingPower = 0
	}
	valMapByPubkey := info.GetValMapByPubkey()
	for pubkey, power := range pubkey2power {
		val, ok := valMapByPubkey[pubkey]
		if !ok || val.IsRetiring {
			continue
		}
		if uint256.NewInt().SetBytes32(val.StakedCoins[:]).Cmp(MinimumStakingAmount) >= 0 {
			val.VotingPower = power
		}
	}
}

// Remove the useless validators from info and return StakedCoins to them
func clearUp(ctx *mevmtypes.Context, stakingAcc *mevmtypes.AccountInfo, info *types.StakingInfo) {
	uselessValMap := info.GetUselessValidators()
	valMapByAddr := info.GetValMapByAddr()
	stakingAccBalance := stakingAcc.Balance()
	for addr := range uselessValMap {
		val := valMapByAddr[addr]
		acc := ctx.GetAccount(val.RewardTo)
		if acc == nil {
			acc = mevmtypes.ZeroAccountInfo()
		}
		coins := uint256.NewInt().SetBytes32(val.StakedCoins[:])
		stakingAccBalance.Sub(stakingAccBalance, coins)
		balance := acc.Balance()
		balance.Add(balance, coins)
		acc.UpdateBalance(balance)
		ctx.SetAccount(val.RewardTo, acc)
	}
	//delete useless validator
	for i, v := range info.Validators {
		if _, ok := uselessValMap[v.Address]; ok {
			info.Validators = append(info.Validators[:i], info.Validators[i+1:]...)
		}
	}
	stakingAcc.UpdateBalance(stakingAccBalance)
	ctx.SetAccount(StakingContractAddress, stakingAcc)
	//save info out here
	//SaveStakingInfo(*ctx, stakingAcc, *info)
}
