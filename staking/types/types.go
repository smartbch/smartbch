package types

import (
	"bytes"
	"sort"

	"github.com/holiman/uint256"
	"github.com/smartbch/smartbch/param"
)

//go:generate msgp

// Currently the first Vout in a coinbase transaction can nominate one validator with one vote
// In the future it maybe extend to nominate multiple validators with different weights
type Nomination struct {
	Pubkey         [32]byte // The validator's ED25519 pubkey used in tendermint
	NominatedCount int64
}

// An NominationHeap is a max-heap of *Nomination
type NominationHeap []*Nomination

func (h NominationHeap) Len() int {
	return len(h)
}

func (h NominationHeap) Less(i, j int) bool {
	if h[i].NominatedCount > h[j].NominatedCount { // larger first
		return true
	} else if h[i].NominatedCount == h[j].NominatedCount {
		return bytes.Compare(h[i].Pubkey[:], h[j].Pubkey[:]) < 0
	} else {
		return false
	}
}

func (h NominationHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *NominationHeap) Push(x any) {
	*h = append(*h, x.(*Nomination))
}

func (h *NominationHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil //avoid memory leak
	*h = old[0 : n-1]
	return x
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

// Returns current validators on duty, who must have enough coins staked and be not in a retiring process
// only update validator voting power on switchEpoch
func GetActiveValidators(vals []*Validator, minStakedCoins *uint256.Int) []*Validator {
	res := make([]*Validator, 0, len(vals))
	for _, val := range vals {
		coins := uint256.NewInt(0).SetBytes32(val.StakedCoins[:])
		if coins.Cmp(minStakedCoins) >= 0 && !val.IsRetiring && val.VotingPower > 0 {
			res = append(res, val)
		}
	}
	//sort: 1.voting power; 2.create validator time (so stable sort is required)
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].VotingPower > res[j].VotingPower
	})
	if len(res) > param.MaxActiveValidatorCount {
		res = res[:param.MaxActiveValidatorCount]
	}
	return res
}
