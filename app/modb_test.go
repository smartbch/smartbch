package app_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/internal/testutils"
)

func filterFunc(addr gethcmn.Address, topics []gethcmn.Hash, addrList []gethcmn.Address, topicsList [][]gethcmn.Hash) (ok bool) {
	return true
}

func TestGetBlock(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()

	blk := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, types.Log{
			Address: gethcmn.Address{0xAD, 0x01},
		}).
		Tx(gethcmn.Hash{0xC2}, types.Log{
			Address: gethcmn.Address{0xAD, 0x02},
		}).
		Build()
	_app.AddBlocksToHistory(blk)

	ctx := _app.GetRpcContext()
	defer ctx.Close(false)
	blk1, err := ctx.GetBlockByHeight(1)
	require.NoError(t, err)
	require.Len(t, blk1.Transactions, 2)
}

func TestQueryLogs(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()

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
	_app.AddBlocksToHistory(blk)

	ctx := _app.GetRpcContext()
	defer ctx.Close(false)

	logs, err := ctx.QueryLogs([]gethcmn.Address{addr1}, [][]gethcmn.Hash{}, 1, 2, filterFunc)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1}, [][]gethcmn.Hash{{topic1}}, 1, 2, filterFunc)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1}, [][]gethcmn.Hash{{}, {topic2}}, 1, 2, filterFunc)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1}, [][]gethcmn.Hash{{topic1}, {topic2}}, 1, 2, filterFunc)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1, addr2}, [][]gethcmn.Hash{{topic1, topic3}}, 1, 2, filterFunc)
	require.NoError(t, err)
	require.Len(t, logs, 2)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1, addr2}, [][]gethcmn.Hash{{}, {topic2, topic4}}, 1, 2, filterFunc)
	require.NoError(t, err)
	require.Len(t, logs, 2)

	logs, err = ctx.QueryLogs([]gethcmn.Address{addr1, addr2}, [][]gethcmn.Hash{{topic1, topic3}, {topic2, topic4}}, 1, 2, filterFunc)
	require.NoError(t, err)
	require.Len(t, logs, 2)
}

func TestGetLogsMaxResults(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()

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
	_app.AddBlocksToHistory(blk)

	ctx := _app.GetRpcContext()
	defer ctx.Close(false)

	logs, err := ctx.QueryLogs([]gethcmn.Address{addr}, nil, 1, 2, filterFunc)
	require.NoError(t, err)
	require.Len(t, logs, 10)

	_app.HistoryStore().SetMaxEntryCount(5)
	logs, err = ctx.QueryLogs([]gethcmn.Address{addr}, nil, 1, 2, filterFunc)
	require.Equal(t, "too many candidicate entries to be returned, please limit the difference between startHeight and endHeight", err.Error())
	require.Len(t, logs, 0)
}

func TestGetBlockHashByHeight(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)

	tx, _ := _app.MakeAndExecTxInBlock(key1, addr2, 100, nil)
	_app.EnsureTxSuccess(tx.Hash())
	_app.ExecTxsInBlock()
	_app.WaitMS(500)

	ctx := _app.GetRpcContext()
	hash := ctx.GetBlockHashByHeight(1)
	require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000001",
		hex.EncodeToString(hash[:]))
	hash = ctx.GetBlockHashByHeight(2)
	require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000002",
		hex.EncodeToString(hash[:]))
	hash = ctx.GetBlockHashByHeight(3)
	require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000003",
		hex.EncodeToString(hash[:]))
	//hash = ctx.GetBlockHashByHeight(4)
	//require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000000",
	//	hex.EncodeToString(hash[:]))
	ctx.Close(false)
	_app.Stop()

	_app = _app.ReloadApp()
	ctx = _app.GetRpcContext()
	hash = ctx.GetBlockHashByHeight(1)
	require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000001",
		hex.EncodeToString(hash[:]))
	hash = ctx.GetBlockHashByHeight(2)
	require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000002",
		hex.EncodeToString(hash[:]))
	hash = ctx.GetBlockHashByHeight(3)
	require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000003",
		hex.EncodeToString(hash[:]))
	//hash = ctx.GetBlockHashByHeight(4)
	//require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000000",
	//	hex.EncodeToString(hash[:]))
	ctx.Close(false)
	_app.DestroyWithoutCheck()
}
