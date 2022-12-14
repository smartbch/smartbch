package crosschain

import (
	"math/rand"
	"testing"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestSortOperators(t *testing.T) {
	ops := []*OperatorInfo{
		{Addr: gethcmn.Address{0xA8}, SelfStakedAmt: uint256.NewInt(444), TotalStakedAmt: uint256.NewInt(456)},
		{Addr: gethcmn.Address{0xA7}, SelfStakedAmt: uint256.NewInt(333), TotalStakedAmt: uint256.NewInt(567)},
		{Addr: gethcmn.Address{0xA6}, SelfStakedAmt: uint256.NewInt(777), TotalStakedAmt: uint256.NewInt(567)},
		{Addr: gethcmn.Address{0xA5}, SelfStakedAmt: uint256.NewInt(777), TotalStakedAmt: uint256.NewInt(567)},
		{Addr: gethcmn.Address{0xA4}, SelfStakedAmt: uint256.NewInt(777), TotalStakedAmt: uint256.NewInt(567)},
		{Addr: gethcmn.Address{0xA3}, SelfStakedAmt: uint256.NewInt(888), TotalStakedAmt: uint256.NewInt(567)},
		{Addr: gethcmn.Address{0xA2}, SelfStakedAmt: uint256.NewInt(999), TotalStakedAmt: uint256.NewInt(567)},
		{Addr: gethcmn.Address{0xA1}, SelfStakedAmt: uint256.NewInt(222), TotalStakedAmt: uint256.NewInt(678)},
		{Addr: gethcmn.Address{0xA0}, SelfStakedAmt: uint256.NewInt(111), TotalStakedAmt: uint256.NewInt(789)},
	}
	shuffle(ops)
	sortOperatorInfosDesc(ops)
	require.Equal(t, gethcmn.Address{0xA0}, ops[0].Addr)
	require.Equal(t, gethcmn.Address{0xA1}, ops[1].Addr)
	require.Equal(t, gethcmn.Address{0xA2}, ops[2].Addr)
	require.Equal(t, gethcmn.Address{0xA3}, ops[3].Addr)
	require.Equal(t, gethcmn.Address{0xA6}, ops[4].Addr)
	require.Equal(t, gethcmn.Address{0xA5}, ops[5].Addr)
	require.Equal(t, gethcmn.Address{0xA4}, ops[6].Addr)
	require.Equal(t, gethcmn.Address{0xA7}, ops[7].Addr)
	require.Equal(t, gethcmn.Address{0xA8}, ops[8].Addr)
}

func TestSortMonitors(t *testing.T) {
	mis := []*MonitorInfo{
		{Addr: gethcmn.Address{0xA8}, NominatedByOps: uint256.NewInt(500), StakedAmt: uint256.NewInt(666), powNominatedCount: 456},
		{Addr: gethcmn.Address{0xA7}, NominatedByOps: uint256.NewInt(600), StakedAmt: uint256.NewInt(777), powNominatedCount: 567},
		{Addr: gethcmn.Address{0xA6}, NominatedByOps: uint256.NewInt(700), StakedAmt: uint256.NewInt(888), powNominatedCount: 567},
		{Addr: gethcmn.Address{0xA5}, NominatedByOps: uint256.NewInt(800), StakedAmt: uint256.NewInt(888), powNominatedCount: 567},
		{Addr: gethcmn.Address{0xA4}, NominatedByOps: uint256.NewInt(800), StakedAmt: uint256.NewInt(888), powNominatedCount: 567},
		{Addr: gethcmn.Address{0xA3}, NominatedByOps: uint256.NewInt(900), StakedAmt: uint256.NewInt(888), powNominatedCount: 567},
		{Addr: gethcmn.Address{0xA2}, NominatedByOps: uint256.NewInt(300), StakedAmt: uint256.NewInt(999), powNominatedCount: 567},
		{Addr: gethcmn.Address{0xA1}, NominatedByOps: uint256.NewInt(200), StakedAmt: uint256.NewInt(222), powNominatedCount: 678},
		{Addr: gethcmn.Address{0xA0}, NominatedByOps: uint256.NewInt(100), StakedAmt: uint256.NewInt(111), powNominatedCount: 789},
	}
	shuffle(mis)
	sortMonitorInfosDesc(mis)
	require.Equal(t, gethcmn.Address{0xA0}, mis[0].Addr)
	require.Equal(t, gethcmn.Address{0xA1}, mis[1].Addr)
	require.Equal(t, gethcmn.Address{0xA2}, mis[2].Addr)
	require.Equal(t, gethcmn.Address{0xA3}, mis[3].Addr)
	require.Equal(t, gethcmn.Address{0xA5}, mis[4].Addr)
	require.Equal(t, gethcmn.Address{0xA4}, mis[5].Addr)
	require.Equal(t, gethcmn.Address{0xA6}, mis[6].Addr)
	require.Equal(t, gethcmn.Address{0xA7}, mis[7].Addr)
	require.Equal(t, gethcmn.Address{0xA8}, mis[8].Addr)
}

func TestFilterOperators(t *testing.T) {
	ops := []*OperatorInfo{
		// current ops
		{Addr: gethcmn.Address{0xA0}, ElectedTime: uint256.NewInt(123)},
		{Addr: gethcmn.Address{0xA1}, ElectedTime: uint256.NewInt(123)},
		{Addr: gethcmn.Address{0xA2}, ElectedTime: uint256.NewInt(123)},
		// selfStakedAmt > min
		{Addr: gethcmn.Address{0xA3}, ElectedTime: uint256.NewInt(0), SelfStakedAmt: uint256.NewInt(0).SubUint64(operatorMinStakedAmt, 1)},
		// new candidates
		{Addr: gethcmn.Address{0xA4}, ElectedTime: uint256.NewInt(0), SelfStakedAmt: uint256.NewInt(0).AddUint64(operatorMinStakedAmt, 1)},
		{Addr: gethcmn.Address{0xA5}, ElectedTime: uint256.NewInt(0), SelfStakedAmt: uint256.NewInt(0).AddUint64(operatorMinStakedAmt, 2)},
	}
	//shuffle(ops)

	currOps, newCandidates := getCurrOperatorsAndCandidates(ops)
	//sortOperatorInfosDesc(currOps)
	require.Len(t, currOps, 3)
	require.Equal(t, gethcmn.Address{0xA0}, currOps[0].Addr)
	require.Equal(t, gethcmn.Address{0xA1}, currOps[1].Addr)
	require.Equal(t, gethcmn.Address{0xA2}, currOps[2].Addr)
	//sortOperatorInfosDesc(newCandidates)
	require.Len(t, newCandidates, 2)
	require.Equal(t, gethcmn.Address{0xA4}, newCandidates[0].Addr)
	require.Equal(t, gethcmn.Address{0xA5}, newCandidates[1].Addr)
}

func TestFilterMonitors(t *testing.T) {
	mis := []*MonitorInfo{
		// current monitors
		{Addr: gethcmn.Address{0xA0}, ElectedTime: uint256.NewInt(123), Pubkey: []byte{0xB0}},
		{Addr: gethcmn.Address{0xA1}, ElectedTime: uint256.NewInt(123), Pubkey: []byte{0xB1}},
		{Addr: gethcmn.Address{0xA2}, ElectedTime: uint256.NewInt(123), Pubkey: []byte{0xB2}},
		// no pow nomination
		{Addr: gethcmn.Address{0xA3}, ElectedTime: uint256.NewInt(0), Pubkey: []byte{0xB3}},
		// stakedAmt < min
		{
			Addr:        gethcmn.Address{0xA4},
			ElectedTime: uint256.NewInt(0),
			Pubkey:      []byte{0xB4},
			StakedAmt:   uint256.NewInt(0).SubUint64(monitorMinStakedAmt, 1),
		},
		// nominatedByOps < min
		{
			Addr:           gethcmn.Address{0xA5},
			ElectedTime:    uint256.NewInt(0),
			Pubkey:         []byte{0xB5},
			StakedAmt:      uint256.NewInt(0).AddUint64(monitorMinStakedAmt, 1),
			NominatedByOps: uint256.NewInt(5),
		},
		// new candidates
		{
			Addr:           gethcmn.Address{0xA6},
			ElectedTime:    uint256.NewInt(0),
			Pubkey:         []byte{0xB6},
			StakedAmt:      uint256.NewInt(0).AddUint64(monitorMinStakedAmt, 1),
			NominatedByOps: uint256.NewInt(6),
		},
		{
			Addr:           gethcmn.Address{0xA7},
			ElectedTime:    uint256.NewInt(0),
			Pubkey:         []byte{0xB7},
			StakedAmt:      uint256.NewInt(0).AddUint64(monitorMinStakedAmt, 1),
			NominatedByOps: uint256.NewInt(7),
		},
	}
	//shuffle(mis)
	powNominations := map[[33]byte]int64{
		to33byte([]byte{0xB0}): 2,
		to33byte([]byte{0xB1}): 1,
		to33byte([]byte{0xB2}): 0,
		to33byte([]byte{0xB4}): 4,
		to33byte([]byte{0xB5}): 5,
		to33byte([]byte{0xB6}): 7,
		to33byte([]byte{0xB7}): 6,
		to33byte([]byte{0xB8}): 8,
	}

	currMonitors, _ := getCurrMonitorsAndCandidates(mis, powNominations)
	//sortOperatorInfosDesc(currOps)
	require.Len(t, currMonitors, 3)
	require.Equal(t, gethcmn.Address{0xA0}, currMonitors[0].Addr)
	require.Equal(t, int64(2), currMonitors[0].powNominatedCount)
	require.Equal(t, gethcmn.Address{0xA1}, currMonitors[1].Addr)
	require.Equal(t, int64(1), currMonitors[1].powNominatedCount)
	require.Equal(t, gethcmn.Address{0xA2}, currMonitors[2].Addr)
	require.Equal(t, int64(0), currMonitors[2].powNominatedCount)

	_, newCandidates := getCurrMonitorsAndCandidates(mis, powNominations)
	//sortOperatorInfosDesc(newCandidates)
	require.Len(t, newCandidates, 2)
	require.Equal(t, gethcmn.Address{0xA6}, newCandidates[0].Addr)
	require.Equal(t, int64(7), newCandidates[0].powNominatedCount)
	require.Equal(t, gethcmn.Address{0xA7}, newCandidates[1].Addr)
	require.Equal(t, int64(6), newCandidates[1].powNominatedCount)
}

func shuffle[T any](ops []T) {
	rand.Shuffle(len(ops), func(i, j int) {
		ops[i], ops[j] = ops[j], ops[i]
	})
}

func to33byte(bz []byte) (pk [33]byte) {
	copy(pk[:], bz)
	return
}
