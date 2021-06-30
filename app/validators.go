package app

import (
	"github.com/holiman/uint256"

	gethcmn "github.com/ethereum/go-ethereum/common"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

type ValidatorsInfo struct {
	// StakingInfo
	GenesisMainnetBlockHeight int64            `json:"genesisMainnetBlockHeight"`
	CurrEpochNum              int64            `json:"currEpochNum"`
	Validators                []*Validator     `json:"validators"`
	ValidatorsUpdate          []*Validator     `json:"validatorsUpdate"`
	PendingRewards            []*PendingReward `json:"pendingRewards"`

	// App
	CurrValidators []*Validator `json:"currValidators"`
}

type PendingReward struct {
	Address  gethcmn.Address `json:"address"`
	EpochNum int64           `json:"epochNum"`
	Amount   string          `json:"amount"`
}

func newValidatorsInfo(currValidators []*stakingtypes.Validator,
	stakingInfo stakingtypes.StakingInfo) ValidatorsInfo {

	info := ValidatorsInfo{
		GenesisMainnetBlockHeight: stakingInfo.GenesisMainnetBlockHeight,
		CurrEpochNum:              stakingInfo.CurrEpochNum,
		Validators:                FromStakingValidators(stakingInfo.Validators),
		ValidatorsUpdate:          FromStakingValidators(stakingInfo.ValidatorsUpdate),
		CurrValidators:            FromStakingValidators(currValidators),
	}

	info.PendingRewards = make([]*PendingReward, len(stakingInfo.PendingRewards))
	for i, pr := range stakingInfo.PendingRewards {
		info.PendingRewards[i] = &PendingReward{
			Address:  pr.Address,
			EpochNum: pr.EpochNum,
			Amount:   uint256.NewInt().SetBytes(pr.Amount[:]).String(),
		}
	}

	return info
}
