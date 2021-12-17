package types

import (
	"testing"

	"github.com/holiman/uint256"

	"github.com/stretchr/testify/require"
)

func TestAddValidator(t *testing.T) {
	si := &StakingInfo{}

	addr1 := [20]byte{0xad, 0x01}
	addr2 := [20]byte{0xad, 0x02}
	addr3 := [20]byte{0xad, 0x03}
	addr4 := [20]byte{0xad, 0x04}
	pubkey1 := [32]byte{0xbe, 0x01}
	pubkey2 := [32]byte{0xbe, 0x02}
	pubkey3 := [32]byte{0xbe, 0x03}
	pubkey4 := [32]byte{0xbe, 0x04}

	require.NoError(t, si.AddValidator(addr1, pubkey1, "val1", [32]byte{0xcc, 0x01}, addr1))
	require.NoError(t, si.AddValidator(addr2, pubkey2, "val2", [32]byte{0xcc, 0x02}, addr2))
	require.NoError(t, si.AddValidator(addr3, pubkey3, "val3", [32]byte{0xcc, 0x03}, addr3))

	require.Equal(t, ValidatorAddressAlreadyExists,
		si.AddValidator(addr1, pubkey4, "val4", [32]byte{0xcc, 0x04}, addr4))
	require.Equal(t, ValidatorPubkeyAlreadyExists,
		si.AddValidator(addr4, pubkey2, "val4", [32]byte{0xcc, 0x04}, addr4))

	require.Equal(t, si.Validators[1], si.GetValidatorByAddr(addr2))
	require.Equal(t, si.Validators[2], si.GetValidatorByPubkey(pubkey3))

	require.Nil(t, si.GetValidatorByAddr([20]byte{0xad, 0x99}))
	require.Nil(t, si.GetValidatorByPubkey([32]byte{0xbe, 0x99}))
}

func TestGetUselessValidators(t *testing.T) {
	si := &StakingInfo{
		Validators: []*Validator{
			{Address: [20]byte{0xad, 0x01}, VotingPower: 1},
			{Address: [20]byte{0xad, 0x02}, VotingPower: 0},
			{Address: [20]byte{0xad, 0x03}, VotingPower: 2},
			{Address: [20]byte{0xad, 0x04}, VotingPower: 0},
			{Address: [20]byte{0xad, 0x05}, VotingPower: 3},
			{Address: [20]byte{0xad, 0x06}, VotingPower: 0},
			{Address: [20]byte{0xad, 0x07}, VotingPower: 4},
			{Address: [20]byte{0xad, 0x08}, VotingPower: 0},
			{Address: [20]byte{0xad, 0x09}, VotingPower: 5},
		},
		PendingRewards: []*PendingReward{
			{Address: [20]byte{0xad, 0x04}},
			{Address: [20]byte{0xad, 0x05}},
			{Address: [20]byte{0xad, 0x06}},
		},
	}
	vs := si.GetUselessValidators()
	require.Len(t, vs, 2)
	require.Contains(t, vs, [20]byte{0xad, 0x02})
	require.Contains(t, vs, [20]byte{0xad, 0x08})
}

func TestGetActiveValidators(t *testing.T) {
	si := &StakingInfo{
		Validators: []*Validator{
			{Address: [20]byte{0xad, 0x01}, StakedCoins: [32]byte{0x10}, IsRetiring: false, VotingPower: 1},
			{Address: [20]byte{0xad, 0x02}, StakedCoins: [32]byte{0x11}, IsRetiring: true, VotingPower: 10},
			{Address: [20]byte{0xad, 0x03}, StakedCoins: [32]byte{0x12}, IsRetiring: false, VotingPower: 0},
			{Address: [20]byte{0xad, 0x04}, StakedCoins: [32]byte{0x13}, IsRetiring: false, VotingPower: 3},
			{Address: [20]byte{0xad, 0x05}, StakedCoins: [32]byte{0x14}, IsRetiring: false, VotingPower: 1},
			{Address: [20]byte{0xad, 0x06}, StakedCoins: [32]byte{0x15}, IsRetiring: false, VotingPower: 7},
			{Address: [20]byte{0xad, 0x07}, StakedCoins: [32]byte{0x16}, IsRetiring: false, VotingPower: 4},
		},
	}

	min := [32]byte{0x11}
	vals := si.GetActiveValidators(uint256.NewInt(0).SetBytes32(min[:]))
	require.Len(t, vals, 4)
	require.Equal(t, [20]byte{0xad, 0x06}, vals[0].Address)
	require.Equal(t, [20]byte{0xad, 0x07}, vals[1].Address)
	require.Equal(t, [20]byte{0xad, 0x04}, vals[2].Address)
	require.Equal(t, [20]byte{0xad, 0x05}, vals[3].Address)
}

func TestClearRewardsOf(t *testing.T) {
	si := &StakingInfo{
		CurrEpochNum: 99,
		PendingRewards: []*PendingReward{
			{Address: [20]byte{0xad, 0x01}, Amount: [32]byte{0x00, 0x03}},
			{Address: [20]byte{0xad, 0x02}, Amount: [32]byte{0x00, 0x02}},
			{Address: [20]byte{0xad, 0x01}, Amount: [32]byte{0x00, 0x09}, EpochNum: 99},
			{Address: [20]byte{0xad, 0x03}, Amount: [32]byte{0x00, 0x01}},
			{Address: [20]byte{0xad, 0x01}, Amount: [32]byte{0x00, 0x05}},
			{Address: [20]byte{0xad, 0x04}, Amount: [32]byte{0x00, 0x04}},
			{Address: [20]byte{0xad, 0x01}, Amount: [32]byte{0x00, 0x06}},
		},
	}

	r := si.ClearRewardsOf([20]byte{0xad, 0x01})
	require.Equal(t, [32]byte{0x00, 0x17}, r.Bytes32())
	require.Len(t, si.PendingRewards, 4)
	require.Equal(t, [20]byte{0xad, 0x02}, si.PendingRewards[0].Address)
	require.Equal(t, [20]byte{0xad, 0x01}, si.PendingRewards[1].Address)
	require.Equal(t, [20]byte{0xad, 0x03}, si.PendingRewards[2].Address)
	require.Equal(t, [20]byte{0xad, 0x04}, si.PendingRewards[3].Address)
}
