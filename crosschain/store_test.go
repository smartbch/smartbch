package crosschain

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartbch/moeingads/store"
	"github.com/smartbch/moeingads/store/rabbit"
	mtypes "github.com/smartbch/moeingevm/types"

	"github.com/smartbch/smartbch/crosschain/types"
)

func TestCCStore(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := mtypes.NewContext(&r, nil)
	context := types.CCContext{
		LastRescannedHeight: 1,
	}
	SaveCCContext(ctx, context)
	loaded := LoadCCContext(ctx)
	require.Equal(t, context.LastRescannedHeight, loaded.LastRescannedHeight)

	record := types.UTXORecord{
		Txid:   [32]byte{0x1},
		Index:  1,
		Amount: [32]byte{0x2},
	}
	SaveUTXORecord(ctx, record)
	loadedR := LoadUTXORecord(ctx, record.Txid, record.Index)
	require.Equal(t, record.Amount, loadedR.Amount)

	voteInfo := types.MonitorVoteInfo{
		Number:      1,
		StartHeight: 100,
		EndTime:     30,
		Nominations: []*types.Nomination{
			{
				Pubkey:         [33]byte{0x01},
				NominatedCount: 1},
		},
	}
	SaveMonitorVoteInfo(ctx, voteInfo)
	loadedV := LoadMonitorVoteInfo(ctx, voteInfo.Number)
	require.Equal(t, voteInfo.StartHeight, loadedV.StartHeight)
	require.Equal(t, len(voteInfo.Nominations), len(loadedV.Nominations))
}
