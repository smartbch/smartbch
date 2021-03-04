package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/moeing-chain/MoeingEVM/types"
	"github.com/moeing-chain/moeing-chain/internal/testutils"
)

func TestGetBlock(t *testing.T) {
	_app := CreateTestApp()
	defer DestroyTestApp(_app)

	blk := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, types.Log{
			Address: gethcmn.Address{0xAD, 0x01},
		}).
		Tx(gethcmn.Hash{0xC2}, types.Log{
			Address: gethcmn.Address{0xAD, 0x02},
		}).
		Build()

	_app.historyStore.AddBlock(blk, -1)
	_app.historyStore.AddBlock(nil, -1)
	time.Sleep(10 * time.Millisecond)

	ctx := _app.GetContext(RpcMode)
	defer ctx.Close(false)
	blk1, err := ctx.GetBlockByHeight(1)
	require.NoError(t, err)
	require.Len(t, blk1.Transactions, 2)
}

func TestQueryLogs(t *testing.T) {
	_app := CreateTestApp()
	defer DestroyTestApp(_app)

	addr1 := gethcmn.Address{0xA1, 0x23}
	addr2 := gethcmn.Address{0xA2, 0x34}
	topic1 := gethcmn.Hash{0xD1, 0x23}
	topic2 := gethcmn.Hash{0xD2, 0x34}
	topic3 := gethcmn.Hash{0xD3, 0x45}
	topic4 := gethcmn.Hash{0xD4, 0x56}

	blk := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, types.Log{
			Address: addr1,
			Topics:  [][32]byte{topic1, topic2},
		}).
		Tx(gethcmn.Hash{0xC2}, types.Log{
			Address: addr2,
			Topics:  [][32]byte{topic3, topic4},
		}).
		Build()
	_app.historyStore.AddBlock(blk, -1)
	_app.historyStore.AddBlock(nil, -1)
	time.Sleep(10 * time.Millisecond)

	ctx := _app.GetContext(RpcMode)
	defer ctx.Close(false)

	logs, err := ctx.QueryLogs([]gethcmn.Address{addr1}, [][]gethcmn.Hash{}, 1, 1)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1}, [][]gethcmn.Hash{{topic1}}, 1, 1)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1}, [][]gethcmn.Hash{{}, {topic2}}, 1, 1)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1}, [][]gethcmn.Hash{{topic1}, {topic2}}, 1, 1)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1, addr2}, [][]gethcmn.Hash{{topic1, topic3}}, 1, 1)
	require.NoError(t, err)
	require.Len(t, logs, 2)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1, addr2}, [][]gethcmn.Hash{{}, {topic2, topic4}}, 1, 1)
	require.NoError(t, err)
	require.Len(t, logs, 2)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1, addr2}, [][]gethcmn.Hash{{topic1, topic3}, {topic2, topic4}}, 1, 1)
	require.NoError(t, err)
	require.Len(t, logs, 2)
}

func TestGetLogsMaxResults(t *testing.T) {
	_app := CreateTestApp()
	defer DestroyTestApp(_app)

	addr := gethcmn.Address{0xA1}
	blk := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, types.Log{Address: addr}).
		Tx(gethcmn.Hash{0xC2}, types.Log{Address: addr}).
		Tx(gethcmn.Hash{0xC3}, types.Log{Address: addr}).
		Tx(gethcmn.Hash{0xC4}, types.Log{Address: addr}).
		Tx(gethcmn.Hash{0xC5}, types.Log{Address: addr}).
		Tx(gethcmn.Hash{0xC6}, types.Log{Address: addr}).
		Tx(gethcmn.Hash{0xC7}, types.Log{Address: addr}).
		Tx(gethcmn.Hash{0xC8}, types.Log{Address: addr}).
		Tx(gethcmn.Hash{0xC9}, types.Log{Address: addr}).
		Tx(gethcmn.Hash{0xC0}, types.Log{Address: addr}).
		Build()

	_app.historyStore.AddBlock(blk, -1)
	_app.historyStore.AddBlock(nil, -1)
	time.Sleep(10 * time.Millisecond)

	ctx := _app.GetContext(RpcMode)
	defer ctx.Close(false)

	logs, err := ctx.QueryLogs([]gethcmn.Address{addr}, nil, 1, 1)
	require.NoError(t, err)
	require.Len(t, logs, 10)

	_app.historyStore.SetMaxEntryCount(5)
	logs, err = ctx.QueryLogs([]gethcmn.Address{addr}, nil, 1, 1)
	require.NoError(t, err)
	require.Len(t, logs, 5)
}
