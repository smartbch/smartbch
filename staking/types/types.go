package types

import (
	"bytes"
	"errors"

	"github.com/holiman/uint256"
)

//go:generate msgp

type Nomination struct {
	Pubkey         [32]byte
	NominatedCount int
}

type BCHBlock struct {
	Height      int64
	Timestamp   int64
	HashId      [32]byte
	ParentBlk   [32]byte
	Nominations []Nomination
}

type RpcClient interface {
	Dial()
	Close()
	GetLatestHeight() int64
	GetBlockByHeight(height int64) *BCHBlock
	GetBlockByHash(hash [32]byte) *BCHBlock
}

type Epoch struct {
	StartHeight    int64
	EndTime        int64
	Duration       int64
	ValMapByPubkey map[[32]byte]*Nomination
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

type PendingReward struct {
	Address  [20]byte `msgp:"address"`   // Validator's address in moeing chain
	EpochNum int64    `msgp:"epoch_num"` // During which epoch were the rewards got?
	Amount   [32]byte `msgp:"coins"`     // amount of rewards
}

type StakingInfo struct {
	CurrEpochNum   int64            `msgp:"curr_epoch_num"`
	Validators     []*Validator     `msgp:"validators"`
	PendingRewards []*PendingReward `msgp:"pending_rewards"`
}

var (
	ValidatorAddressAlreadyExists = errors.New("Validator's address already exists")
	ValidatorPubkeyAlreadyExists  = errors.New("Validator's pubkey already exists")
)

func (si *StakingInfo) GetValMapByPubkey() map[[32]byte]*Validator {
	res := make(map[[32]byte]*Validator)
	for _, val := range si.Validators {
		res[val.Pubkey] = val
	}
	return res
}

func (si *StakingInfo) GetValMapByAddr() map[[20]byte]*Validator {
	res := make(map[[20]byte]*Validator)
	for _, val := range si.Validators {
		res[val.Address] = val
	}
	return res
}

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
		if bytes.Equal(addr[:], val.Pubkey[:]) {
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

func (si *StakingInfo) GetValidatorByAddr(addr [20]byte) *Validator {
	for _, val := range si.Validators {
		if bytes.Equal(addr[:], val.Address[:]) {
			return val
		}
	}
	return nil
}

func (si *StakingInfo) GetValidatorByPubkey(pubkey [32]byte) *Validator {
	for _, val := range si.Validators {
		if bytes.Equal(pubkey[:], val.Pubkey[:]) {
			return val
		}
	}
	return nil
}

// Get useless validators who have no voting power and no pending rewards
func (si *StakingInfo) GetUselessValidators() map[[20]byte]struct{} {
	res := make(map[[20]byte]struct{})
	for _, val := range si.Validators {
		if val.VotingPower == 0 {
			res[val.Address] = struct{}{}
		}
	}
	for _, pr := range si.PendingRewards {
		delete(res, pr.Address)
	}
	return res
}

// Clear all the pending rewards belonging to an validator. Return the accumulated cleared amount.
func (si *StakingInfo) ClearRewardsOf(addr [20]byte) (totalCleared *uint256.Int) {
	totalCleared = uint256.NewInt()
	rwdList := make([]*PendingReward, len(si.PendingRewards), 0)
	var bz32Zero [32]byte
	for _, rwd := range si.PendingRewards {
		if !bytes.Equal(rwd.Address[:], addr[:]) {
			rwdList = append(rwdList, rwd)
		} else {
			coins := uint256.NewInt().SetBytes32(rwd.Amount[:])
			totalCleared.Add(totalCleared, coins)
			if rwd.EpochNum == si.CurrEpochNum { // we still need this entry
				rwd.Amount = bz32Zero
				rwdList = append(rwdList, rwd)
			}
		}
	}
	si.PendingRewards = rwdList
	return totalCleared
}

// Returns current validators on duty, who must have enough coins staked and be not in a unbonding process
func (si *StakingInfo) GetValidatorsOnDuty(minStakedCoins *uint256.Int) []*Validator {
	res := make([]*Validator, len(si.Validators), 0)
	for _, val := range si.Validators {
		coins := uint256.NewInt().SetBytes32(val.StakedCoins[:])
		if coins.Cmp(minStakedCoins) >= 0 && !val.IsUnbonding {
			res = append(res, val)
		}
	}
	return res
}
