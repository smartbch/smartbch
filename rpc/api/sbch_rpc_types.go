package api

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

// StakingEpoch

type StakingEpoch struct {
	Number      hexutil.Uint64 `json:"number"`
	StartHeight hexutil.Uint64 `json:"startHeight"`
	EndTime     int64          `json:"endTime"`
	Nominations []*Nomination  `json:"nominations"`
}
type Nomination struct {
	Pubkey         gethcmn.Hash `json:"pubkey"`
	NominatedCount int64        `json:"nominatedCount"`
}

func castStakingEpochs(epochs []*stakingtypes.Epoch) []*StakingEpoch {
	rpcEpochs := make([]*StakingEpoch, len(epochs))
	for i, epoch := range epochs {
		rpcEpochs[i] = castStakingEpoch(epoch)
	}
	return rpcEpochs
}
func castStakingEpoch(epoch *stakingtypes.Epoch) *StakingEpoch {
	return &StakingEpoch{
		Number:      hexutil.Uint64(epoch.Number),
		StartHeight: hexutil.Uint64(epoch.StartHeight),
		EndTime:     epoch.EndTime,
		Nominations: castNominations(epoch.Nominations),
	}
}
func castNominations(nominations []*stakingtypes.Nomination) []*Nomination {
	rpcNominations := make([]*Nomination, len(nominations))
	for i, nomination := range nominations {
		rpcNominations[i] = &Nomination{
			Pubkey:         nomination.Pubkey,
			NominatedCount: nomination.NominatedCount,
		}
	}
	return rpcNominations
}
