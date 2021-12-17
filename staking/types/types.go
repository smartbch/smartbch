package types

import (
	"bytes"
	"errors"
	"sort"

	"github.com/holiman/uint256"
)

//go:generate msgp

const MaxActiveValidatorNum = 30

var (
	ValidatorAddressAlreadyExists = errors.New("Validator's address already exists")
	ValidatorPubkeyAlreadyExists  = errors.New("Validator's pubkey already exists")
)

// Currently the first Vout in a coinbase transaction can nominate one validator with one vote
// In the future it maybe extend to nominate multiple validators with different weights
type Nomination struct {
	Pubkey         [32]byte // The validator's ED25519 pubkey used in tendermint
	NominatedCount int64
}

// An epoch elects several validators in NumBlocksInEpoch blocks
type Epoch struct {
	Number      int64
	StartHeight int64
	EndTime     int64
	Nominations []*Nomination
}

type Validator struct {
	Address      [20]byte `msgp:"address"`   // Validator's address in smartbch chain
	Pubkey       [32]byte `msgp:"pubkey"`    // Validator's pubkey for tendermint
	RewardTo     [20]byte `msgp:"reward_to"` // where validator's reward goes into
	VotingPower  int64    `msgp:"voting_power"`
	Introduction string   `msgp:"introduction"` // a short introduction
	StakedCoins  [32]byte `msgp:"staked_coins"`
	IsRetiring   bool     `msgp:"is_retiring"` // whether this validator is in a retiring process
}

// Because EpochCountBeforeRewardMature >= 1, some rewards will be pending for a while before mature
type PendingReward struct {
	Address  [20]byte `msgp:"address"`   // Validator's operator address in smartbch chain
	EpochNum int64    `msgp:"epoch_num"` // During which epoch were the rewards got?
	Amount   [32]byte `msgp:"amount"`    // amount of rewards
}

// This struct is stored in the world state.
// All the staking-related operations manipulate it.
type StakingInfo struct {
	GenesisMainnetBlockHeight int64            `msgp:"genesis_mainnet_block_height"`
	CurrEpochNum              int64            `msgp:"curr_epoch_num"`
	Validators                []*Validator     `msgp:"validators"`
	ValidatorsUpdate          []*Validator     `msgp:"validators_update"`
	PendingRewards            []*PendingReward `msgp:"pending_rewards"`
}

// Change si.Validators into a map with pubkeys as keys
func (si *StakingInfo) GetValMapByPubkey() map[[32]byte]*Validator {
	res := make(map[[32]byte]*Validator, len(si.Validators))
	for _, val := range si.Validators {
		res[val.Pubkey] = val
	}
	return res
}

// Change si.Validators into a map with addresses as keys
func (si *StakingInfo) GetValMapByAddr() map[[20]byte]*Validator {
	res := make(map[[20]byte]*Validator, len(si.Validators))
	for _, val := range si.Validators {
		res[val.Address] = val
	}
	return res
}

// Get the pending rewards which are got in current epoch
func (si *StakingInfo) GetCurrRewardMapByAddr() map[[20]byte]*PendingReward {
	res := make(map[[20]byte]*PendingReward, len(si.PendingRewards))
	for _, pr := range si.PendingRewards {
		if pr.EpochNum == si.CurrEpochNum {
			res[pr.Address] = pr
		}
	}
	return res
}

// Append new entry to si.Validators. Pubkey and Address must be unique.
func (si *StakingInfo) AddValidator(addr [20]byte, pubkey [32]byte, intro string, stakedCoins [32]byte, rewardTo [20]byte) error {
	for _, val := range si.Validators {
		if bytes.Equal(addr[:], val.Address[:]) {
			return ValidatorAddressAlreadyExists
		}
		if bytes.Equal(pubkey[:], val.Pubkey[:]) {
			return ValidatorPubkeyAlreadyExists
		}
	}
	val := &Validator{
		Address:      addr,
		Pubkey:       pubkey,
		RewardTo:     rewardTo,
		VotingPower:  0,
		Introduction: intro,
		StakedCoins:  stakedCoins,
		IsRetiring:   false,
	}
	si.Validators = append(si.Validators, val)
	return nil
}

// Find a validator with matching address
func (si *StakingInfo) GetValidatorByAddr(addr [20]byte) *Validator {
	for _, val := range si.Validators {
		if bytes.Equal(addr[:], val.Address[:]) {
			return val
		}
	}
	return nil
}

// Find a validator with matching pubkey
func (si *StakingInfo) GetValidatorByPubkey(pubkey [32]byte) *Validator {
	for _, val := range si.Validators {
		if bytes.Equal(pubkey[:], val.Pubkey[:]) {
			return val
		}
	}
	return nil
}

// Get useless validators who have zero voting power and no pending reward entries
// there has two scenario one validator may be useless:
// 1. retire itself with no pending reward
// 2. inactive validator with no vote power and pending reward in prev epoch,
//    which may escape slash if it votes nothing after double sign !!!
//    maybe there should have more epoch not one.
func (si *StakingInfo) GetUselessValidators() map[[20]byte]struct{} {
	res := make(map[[20]byte]struct{})
	for _, val := range si.Validators {
		if val.VotingPower == 0 {
			res[val.Address] = struct{}{}
		}
	}
	for _, pr := range si.PendingRewards {
		delete(res, pr.Address) // remove the ones with pending reward entries
	}
	return res
}

// Clear all the pending rewards belonging to an validator. Return the accumulated cleared amount.
func (si *StakingInfo) ClearRewardsOf(addr [20]byte) (totalCleared *uint256.Int) {
	totalCleared = uint256.NewInt(0)
	rwdList := make([]*PendingReward, 0, len(si.PendingRewards))
	for _, rwd := range si.PendingRewards {
		if bytes.Equal(rwd.Address[:], addr[:]) {
			coins := uint256.NewInt(0).SetBytes32(rwd.Amount[:])
			totalCleared.Add(totalCleared, coins)
			if rwd.EpochNum == si.CurrEpochNum { // we still need this entry
				rwd.Amount = [32]byte{}        // just clear the amount
				rwdList = append(rwdList, rwd) // the entry is kept
			}
		} else { // rewards of other validators
			rwdList = append(rwdList, rwd)
		}
	}
	si.PendingRewards = rwdList
	return totalCleared
}

// Returns current validators on duty, who must have enough coins staked and be not in a retiring process
// only update validator voting power on switchEpoch
func (si *StakingInfo) GetActiveValidators(minStakedCoins *uint256.Int) []*Validator {
	res := GetActiveValidators(si.Validators, minStakedCoins)

	//sort: 1.voting power; 2.create validator time (so stable sort is required)
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].VotingPower > res[j].VotingPower
	})
	if len(res) > MaxActiveValidatorNum {
		res = res[:MaxActiveValidatorNum]
	}
	return res
}

func GetActiveValidators(vals []*Validator, minStakedCoins *uint256.Int) []*Validator {
	res := make([]*Validator, 0, len(vals))
	for _, val := range vals {
		coins := uint256.NewInt(0).SetBytes32(val.StakedCoins[:])
		if coins.Cmp(minStakedCoins) >= 0 && !val.IsRetiring && val.VotingPower > 0 {
			res = append(res, val)
		}
	}
	return res
}
