package types

import (
	"testing"

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
