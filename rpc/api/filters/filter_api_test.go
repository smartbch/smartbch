package filters

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethfilters "github.com/ethereum/go-ethereum/eth/filters"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/testutils"
)

func TestNewFilter(t *testing.T) {
	_app := app.CreateTestApp()
	defer app.DestroyTestApp(_app)
	_api := createFiltersAPI(_app)
	id, err := _api.NewFilter(gethfilters.FilterCriteria{})
	require.NoError(t, err)
	require.NotEmpty(t, id)

	require.True(t, _api.UninstallFilter(id))
	require.False(t, _api.UninstallFilter(id))
}

func TestGetFilterChanges_blockFilter(t *testing.T) {
	_app := app.CreateTestApp()
	defer app.DestroyTestApp(_app)
	_api := createFiltersAPI(_app)
	id := _api.NewBlockFilter()
	require.NotEmpty(t, id)

	_, err := _api.GetFilterChanges(id)
	require.NoError(t, err)

	block := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1, 0x23}).Build()
	addBlock(_app, block)

	time.Sleep(10 * time.Millisecond)
	ret, err := _api.GetFilterChanges(id)
	require.NoError(t, err)
	hashes, ok := ret.([]gethcmn.Hash)
	require.True(t, ok)
	require.Equal(t, 1, len(hashes))
	require.Equal(t, gethcmn.Hash{0xB1, 0x23}, hashes[0])
}

func TestGetFilterChanges_addrFilter(t *testing.T) {
	_app := app.CreateTestApp()
	defer app.DestroyTestApp(_app)
	_api := createFiltersAPI(_app)
	id, err := _api.NewFilter(testutils.NewAddressFilter(gethcmn.Address{0xA1}))
	require.NoError(t, err)
	require.NotEmpty(t, id)

	logs, err := _api.GetFilterChanges(id)
	require.NoError(t, err)
	require.Len(t, logs, 0)

	block := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, types.Log{
			Address: [20]byte{0xA1},
			Topics:  [][32]byte{{0xD1}, {0xD2}},
		}).Build()
	addBlock(_app, block)

	time.Sleep(10 * time.Millisecond)
	logs, err = _api.GetFilterChanges(id)
	require.NoError(t, err)
	require.Len(t, logs, 1)
}

/*
https://eth.wiki/json-rpc/API#eth_newFilter

Topics are order-dependent.
A transaction with a log with topics [A, B] will be matched by the following topic filters:

[]               “anything”
[A]              “A in first position (and anything after)”
[null, B]        “anything in first position AND B in second position (and anything after)”
[A, B]           “A in first position AND B in second position (and anything after)”
[[A, B], [A, B]] “(A OR B) in first position AND (A OR B) in second position (and anything after)”
*/
func TestGetFilterChanges_topicsFilter(t *testing.T) {
	_app := app.CreateTestApp()
	defer app.DestroyTestApp(_app)
	_api := createFiltersAPI(_app)
	ids := make([]gethrpc.ID, 5)
	ids[0] = newTopicsFilter(t, _api, [][]gethcmn.Hash{})                                           // []
	ids[1] = newTopicsFilter(t, _api, [][]gethcmn.Hash{{gethcmn.Hash{0x0A}}})                       // [A]
	ids[2] = newTopicsFilter(t, _api, [][]gethcmn.Hash{{}, {gethcmn.Hash{0x0B}}})                   // [null, B]
	ids[3] = newTopicsFilter(t, _api, [][]gethcmn.Hash{{gethcmn.Hash{0x0A}}, {gethcmn.Hash{0x0B}}}) // [A, B]
	ids[4] = newTopicsFilter(t, _api, [][]gethcmn.Hash{{gethcmn.Hash{0x0A}, gethcmn.Hash{0x0B}},    // [[A, B], [A, B]]
		{gethcmn.Hash{0x0A}, gethcmn.Hash{0x0B}}})

	block1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, types.Log{
			Address: [20]byte{0xA1, 0x23},
			Topics:  [][32]byte{{0x0A}, {0x0B}},
		}).Build()
	addBlock(_app, block1)

	time.Sleep(10 * time.Millisecond)
	for _, id := range ids {
		logs, err := _api.GetFilterChanges(id)
		require.NoError(t, err)
		require.Len(t, logs, 1)
		require.Equal(t, gethcmn.Address{0xA1, 0x23}, logs.([]*gethtypes.Log)[0].Address)
	}

	block2 := testutils.NewMdbBlockBuilder().
		Height(2).Hash(gethcmn.Hash{0xB2}).
		Tx(gethcmn.Hash{0xC2}, types.Log{
			Address: [20]byte{0xA2, 0x34},
			Topics:  [][32]byte{{0x0B}, {0x0B}},
		}).Build()
	addBlock(_app, block2)

	time.Sleep(10 * time.Millisecond)
	logs0, _ := _api.GetFilterChanges(ids[0])
	require.Len(t, logs0, 1)
	logs1, _ := _api.GetFilterChanges(ids[1])
	require.Len(t, logs1, 0)
	logs2, _ := _api.GetFilterChanges(ids[2])
	require.Len(t, logs2, 1)
	logs3, _ := _api.GetFilterChanges(ids[3])
	require.Len(t, logs3, 0)
	logs4, _ := _api.GetFilterChanges(ids[4])
	require.Len(t, logs4, 1)
}

func TestGetFilterLogs_addrFilter(t *testing.T) {
	_app := app.CreateTestApp()
	defer app.DestroyTestApp(_app)
	_api := createFiltersAPI(_app)
	fc := testutils.NewFilterBuilder().
		BlockRange(1, 3). // the ending '3' is not included
		Addresses(gethcmn.Address{0xA1, 0x23}).Build()
	id, err := _api.NewFilter(fc)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	logs, err := _api.GetFilterLogs(id)
	require.NoError(t, err)
	require.Equal(t, 0, len(logs))

	block1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, types.Log{
			Address: [20]byte{0xA1, 0x23},
			Topics:  [][32]byte{{0xD1, 0xD2}},
		}).
		Tx(gethcmn.Hash{0xC2}, types.Log{
			Address: [20]byte{0xA2, 0x34},
			Topics:  [][32]byte{{0xD3, 0xD4}},
		}).
		Build()
	addBlock(_app, block1)

	block2 := testutils.NewMdbBlockBuilder().
		Height(2).Hash(gethcmn.Hash{0xB2}).
		Tx(gethcmn.Hash{0xC3}, types.Log{
			Address: [20]byte{0xA3, 0x45}}).
		Tx(gethcmn.Hash{0xC4}, types.Log{
			Address: [20]byte{0xA1, 0x23}}).
		Build()
	addBlock(_app, block2)

	time.Sleep(10 * time.Millisecond)
	logs, err = _api.GetFilterLogs(id)
	require.NoError(t, err)
	require.Equal(t, 2, len(logs))
}

func TestGetFilterLogs_blockRangeFilter(t *testing.T) {
	_app := app.CreateTestApp()
	defer app.DestroyTestApp(_app)
	_api := createFiltersAPI(_app)
	id, err := _api.NewFilter(testutils.NewBlockRangeFilter(0, 2))
	require.NoError(t, err)
	require.NotEmpty(t, id)

	logs, err := _api.GetFilterLogs(id)
	require.NoError(t, err)
	require.Equal(t, 0, len(logs))

	block := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, types.Log{
			Address: [20]byte{0xA1, 0x23},
			Topics:  [][32]byte{{0xD1}, {0xD2}}}).
		Build()
	addBlock(_app, block)

	time.Sleep(10 * time.Millisecond)
	fmt.Printf("====================================\n")
	logs, err = _api.GetFilterLogs(id)
	require.NoError(t, err)
	// Why test it this way? if no address nor topics specified, modb returns nothing...
	require.Equal(t, 0, len(logs))
}

func TestGetLogs_blockHashFilter(t *testing.T) {
	_app := app.CreateTestApp()
	defer app.DestroyTestApp(_app)
	_api := createFiltersAPI(_app)

	b1Hash := gethcmn.Hash{0xB1}
	block1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(b1Hash).
		Tx(gethcmn.Hash{0xC1}, types.Log{Address: gethcmn.Address{0xA1}}).
		Build()
	ctx := _app.GetContext(app.RunTxMode)
	ctx.StoreBlock(block1)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)

	b2Hash := gethcmn.Hash{0xB2}
	block2 := testutils.NewMdbBlockBuilder().
		Height(2).Hash(b2Hash).
		Tx(gethcmn.Hash{0xC2}, types.Log{Address: gethcmn.Address{0xA2}}).
		Build()
	ctx = _app.GetContext(app.RunTxMode)
	ctx.StoreBlock(block2)
	ctx.Close(true)

	logs, err := _api.GetLogs(testutils.NewBlockHashFilter(&b1Hash))
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, gethcmn.Address{0xA1}, logs[0].Address)

	logs, err = _api.GetLogs(testutils.NewBlockHashFilter(&b2Hash))
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, gethcmn.Address{0xA2}, logs[0].Address)
}

func TestGetLogs_addrFilter(t *testing.T) {
	_app := app.CreateTestApp()
	defer app.DestroyTestApp(_app)
	_api := createFiltersAPI(_app)

	addr1 := gethcmn.Address{0xA1, 0x23}
	block1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, types.Log{
			Address: addr1,
			Topics:  [][32]byte{{0xD1, 0xD2}},
		}).
		Tx(gethcmn.Hash{0xC2}, types.Log{
			Address: [20]byte{0xA2, 0x34},
			Topics:  [][32]byte{{0xD3, 0xD4}},
		}).
		Build()
	addBlock(_app, block1)

	addr2 := gethcmn.Address{0xA3, 0x45}
	block2 := testutils.NewMdbBlockBuilder().
		Height(2).Hash(gethcmn.Hash{0xB2}).
		Tx(gethcmn.Hash{0xC3}, types.Log{
			Address: addr2,
			Topics:  [][32]byte{{0xD5, 0xD6}}}).
		Tx(gethcmn.Hash{0xC4}, types.Log{
			Address: addr1,
			Topics:  [][32]byte{{0xD7, 0xD8}}}).
		Build()
	addBlock(_app, block2)

	f1 := testutils.NewFilterBuilder().BlockRange(1, 2).Addresses(addr1).Build()
	logs, err := _api.GetLogs(f1)
	require.NoError(t, err)
	require.Len(t, logs, 2)

	f2 := testutils.NewFilterBuilder().BlockRange(1, 2).Addresses(addr2).Build()
	logs, err = _api.GetLogs(f2)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	f3 := testutils.NewFilterBuilder().BlockRange(1, 3).Addresses(addr1).Build()
	logs, err = _api.GetLogs(f3)
	require.NoError(t, err)
	require.Len(t, logs, 2)
}

func createFiltersAPI(_app *app.App) PublicFilterAPI {
	backend := api.NewBackend(nil, _app)
	return NewAPI(backend)
}

func addBlock(_app *app.App, block *modbtypes.Block) {
	_app.BeginBlock(abci.RequestBeginBlock{
		Header: tmproto.Header{Height: block.Height},
	})
	ctx := _app.GetContext(app.RunTxMode)
	ctx.StoreBlock(block)
	ctx.Close(true)
	app.AddBlockFotTest(_app, block)
}

func newTopicsFilter(t *testing.T, _api PublicFilterAPI, topics [][]gethcmn.Hash) gethrpc.ID {
	id, err := _api.NewFilter(testutils.NewTopicsFilter(topics))
	require.NoError(t, err)
	require.NotEmpty(t, id)
	return id
}
