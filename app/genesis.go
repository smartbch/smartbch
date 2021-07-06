package app

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"

	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

type Validator struct {
	Address      gethcmn.Address `json:"address"`
	Pubkey       gethcmn.Hash    `json:"pubkey"`
	RewardTo     gethcmn.Address `json:"reward_to"`
	VotingPower  int64           `json:"voting_power"`
	Introduction string          `json:"introduction"`
	StakedCoins  gethcmn.Hash    `json:"staked_coins"`
	IsRetiring   bool            `json:"is_retiring"`
}

type GenesisData struct {
	Validators []*Validator          `json:"validators"`
	Alloc      gethcore.GenesisAlloc `json:"alloc"`
}

func (g GenesisData) stakingValidators() []*stakingtypes.Validator {
	ret := make([]*stakingtypes.Validator, len(g.Validators))
	for i, v := range g.Validators {
		ret[i] = &stakingtypes.Validator{
			Address:      v.Address,
			Pubkey:       v.Pubkey,
			RewardTo:     v.RewardTo,
			VotingPower:  v.VotingPower,
			Introduction: v.Introduction,
			StakedCoins:  v.StakedCoins,
			IsRetiring:   v.IsRetiring,
		}
	}
	return ret
}

func FromStakingValidators(vs []*stakingtypes.Validator) []*Validator {
	ret := make([]*Validator, len(vs))
	for i, v := range vs {
		ret[i] = FromStakingValidator(v)
	}
	return ret
}

func FromStakingValidator(v *stakingtypes.Validator) *Validator {
	return &Validator{
		Address:      v.Address,
		Pubkey:       v.Pubkey,
		RewardTo:     v.RewardTo,
		VotingPower:  v.VotingPower,
		Introduction: v.Introduction,
		StakedCoins:  v.StakedCoins,
		IsRetiring:   v.IsRetiring,
	}
}
