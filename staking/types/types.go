package types

import (
	"bytes"
	"errors"
	"sort"

	"github.com/holiman/uint256"
)

//go:generate msgp

var MaxActiveValidatorNum = 30

// Currently the first Vout in a coinbase transaction can nominate one validator with one vote
// In the future it maybe extented to nominate multiple validators with different weights
type Nomination struct {
	Pubkey         [32]byte // The validator's ED25519 pubkey used in tendermint
	NominatedCount int64
}

// This struct contains the useful information of a BCH block
type BCHBlock struct {
	Height      int64
	Timestamp   int64
	HashId      [32]byte
	ParentBlk   [32]byte
	Nominations []Nomination
}

//not check Nominations
func (b *BCHBlock) Equal(o *BCHBlock) bool {
	return b.Height == o.Height && b.Timestamp == o.Timestamp &&
		b.HashId == o.HashId && b.ParentBlk == o.ParentBlk
}

// These functions must be provided by a client connecting to a Bitcoin Cash's fullnode
type RpcClient interface {
	GetLatestHeight() int64
	GetBlockByHeight(height int64) *BCHBlock
	GetBlockByHash(hash [32]byte) *BCHBlock
}

// An epoch elects several validators in NumBlocksInEpoch blocks
type Epoch struct {
	StartHeight    int64
	EndTime        int64
	Duration       int64
	ValMapByPubkey map[[32]byte]*Nomination
}

// This struct is stored in the world state.
// All the staking-related operations manipulate it.
type StakingInfo struct {
	CurrEpochNum   int64            `msgp:"curr_epoch_num"`
	Validators     []*Validator     `msgp:"validators"`
	PendingRewards []*PendingReward `msgp:"pending_rewards"`
}

type Validator struct {
	Address      [20]byte `msgp:"address"`   // Validator's address in moeing chain
	Pubkey       [32]byte `msgp:"pubkey"`    // Validator's pubkey for tendermint
	RewardTo     [20]byte `msgp:"reward_to"` // where validator's reward goes into
	VotingPower  int64    `msgp:"votingpower"`
	Introduction string   `msgp:"introduction"` // a short introduction
	StakedCoins  [32]byte `msgp:"staked_coins"`
	IsUnbonding  bool     `msgp:"is_unbonding"` // whether this validator is in a unbonding process
}

// Because EpochCountBeforeRewardMature >= 1, some rewards will be pending for a while before mature
type PendingReward struct {
	Address  [20]byte `msgp:"address"`   // Validator's operator address in moeing chain
	EpochNum int64    `msgp:"epoch_num"` // During which epoch were the rewards got?
	Amount   [32]byte `msgp:"coins"`     // amount of rewards
}

var (
	ValidatorAddressAlreadyExists = errors.New("Validator's address already exists")
	ValidatorPubkeyAlreadyExists  = errors.New("Validator's pubkey already exists")
)

// Change si.Validators into a map with pubkeys as keys
func (si *StakingInfo) GetValMapByPubkey() map[[32]byte]*Validator {
	res := make(map[[32]byte]*Validator)
	for _, val := range si.Validators {
		res[val.Pubkey] = val
	}
	return res
}

// Change si.Validators into a map with addresses as keys
func (si *StakingInfo) GetValMapByAddr() map[[20]byte]*Validator {
	res := make(map[[20]byte]*Validator)
	for _, val := range si.Validators {
		res[val.Address] = val
	}
	return res
}

// Get the pending rewards which are got in current epoch
func (si *StakingInfo) GetCurrRewardMapByAddr() map[[20]byte]*PendingReward {
	res := make(map[[20]byte]*PendingReward)
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
		IsUnbonding:  false,
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
// 1. unbound itself with no pending reward
// 2. inactive validator with no vote power and pending reward in prev epoch, maybe there should have more epoch not one.
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
	totalCleared = uint256.NewInt()
	rwdList := make([]*PendingReward, 0, len(si.PendingRewards))
	var bz32Zero [32]byte
	for _, rwd := range si.PendingRewards {
		if bytes.Equal(rwd.Address[:], addr[:]) {
			coins := uint256.NewInt().SetBytes32(rwd.Amount[:])
			totalCleared.Add(totalCleared, coins)
			if rwd.EpochNum == si.CurrEpochNum { // we still need this entry
				rwd.Amount = bz32Zero          // just clear the amount
				rwdList = append(rwdList, rwd) // the entry is kept
			}
		} else { // rewards of other validators
			rwdList = append(rwdList, rwd)
		}
	}
	si.PendingRewards = rwdList
	return totalCleared
}

// Returns current validators on duty, who must have enough coins staked and be not in a unbonding process
func (si *StakingInfo) GetValidatorsOnDuty(minStakedCoins *uint256.Int) []*Validator {
	res := make([]*Validator, 0, len(si.Validators))
	for _, val := range si.Validators {
		coins := uint256.NewInt().SetBytes32(val.StakedCoins[:])
		if coins.Cmp(minStakedCoins) >= 0 && !val.IsUnbonding && val.VotingPower > 0 {
			res = append(res, val)
		}
	}
	//sort: 1.voting power; 2.create validator time
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].VotingPower > res[j].VotingPower
	})
	if len(res) > MaxActiveValidatorNum {
		res = res[:MaxActiveValidatorNum]
	}
	return res
}
