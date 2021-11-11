package filters

import (
	"fmt"
	"math/big"
	"testing"

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
	"github.com/smartbch/smartbch/internal/testutils"
)

func TestNewFilter(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createFiltersAPI(_app)
	id, err := _api.NewFilter(gethfilters.FilterCriteria{})
	require.NoError(t, err)
	require.NotEmpty(t, id)

	require.True(t, _api.UninstallFilter(id))
	require.False(t, _api.UninstallFilter(id))
}

func TestGetFilterChanges_blockFilter(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createFiltersAPI(_app)
	id := _api.NewBlockFilter()
	require.NotEmpty(t, id)

	_, err := _api.GetFilterChanges(id)
	require.NoError(t, err)

	block := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1, 0x23}).Build()
	addBlock(_app, block)

	_app.WaitMS(10)
	ret, err := _api.GetFilterChanges(id)
	require.NoError(t, err)
	hashes, ok := ret.([]gethcmn.Hash)
	require.True(t, ok)
	require.Equal(t, 1, len(hashes))
	require.Equal(t, gethcmn.Hash{0xB1, 0x23}, hashes[0])
}

func TestGetFilterChanges_addrFilter(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
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

	_app.WaitMS(10)
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
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
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

	_app.WaitMS(10)
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

	_app.WaitMS(10)
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
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
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

	_app.WaitMS(10)
	logs, err = _api.GetFilterLogs(id)
	require.NoError(t, err)
	require.Equal(t, 2, len(logs))
}

func TestGetFilterLogs_blockRangeFilter(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
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

	_app.WaitMS(10)
	fmt.Printf("====================================\n")
	logs, err = _api.GetFilterLogs(id)
	require.NoError(t, err)
	// Why test it this way? if no address nor topics specified, modb returns nothing...
	require.Equal(t, 0, len(logs))
}

func TestGetLogs_blockHashFilter(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createFiltersAPI(_app)

	b1Hash := gethcmn.Hash{0xB1}
	block1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(b1Hash).
		Tx(gethcmn.Hash{0xC1}, types.Log{Address: gethcmn.Address{0xA1}}).
		Build()

	b2Hash := gethcmn.Hash{0xB2}
	block2 := testutils.NewMdbBlockBuilder().
		Height(2).Hash(b2Hash).
		Tx(gethcmn.Hash{0xC2}, types.Log{Address: gethcmn.Address{0xA2}}).
		Build()

	_app.StoreBlocks(block1, block2)
	_app.WaitMS(10)

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
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
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

func TestGetLogs_blockRangeFilter(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createFiltersAPI(_app)

	addr := gethcmn.Address{0xA1, 0x23}
	addBlock(_app, testutils.NewMdbBlockBuilder().
		Height(0).Hash(gethcmn.Hash{0xB0}).
		Tx(gethcmn.Hash{0xC0}, types.Log{Address: addr, Topics: [][32]byte{{0xD0}}}).
		Build())
	addBlock(_app, testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, types.Log{Address: addr, Topics: [][32]byte{{0xD1}}}).
		Tx(gethcmn.Hash{0xC2}, types.Log{Address: addr, Topics: [][32]byte{{0xD2}}}).
		Build())
	addBlock(_app, testutils.NewMdbBlockBuilder().
		Height(2).Hash(gethcmn.Hash{0xB2}).
		Tx(gethcmn.Hash{0xC3}, types.Log{Address: addr, Topics: [][32]byte{{0xD3}}}).
		Tx(gethcmn.Hash{0xC4}, types.Log{Address: addr, Topics: [][32]byte{{0xD4}}}).
		Tx(gethcmn.Hash{0xC5}, types.Log{Address: addr, Topics: [][32]byte{{0xD5}}}).
		Build())

	testCases := []struct {
		hFrom int64
		hTo   int64
		nLogs int
	}{
		{0, 0, 1},
		{0, 1, 3},
		{0, 2, 6},
		{0, 3, 6},
		{0, -1, 6},
		{0, -2, 6},
		{1, 0, 0},
		{1, 1, 2},
		{1, 2, 5},
		{1, 3, 5},
		{1, -1, 5},
		{2, 1, 0},
		{2, 2, 3},
		{2, 3, 3},
		{2, -1, 3},
		{-1, -1, 3},
		{-1, 0, 0},
		{-1, 1, 0},
		{-1, 2, 3},
		{-1, 3, 3},
	}

	for _, testCase := range testCases {
		f := testutils.NewFilterBuilder().BlockRange(testCase.hFrom, testCase.hTo).Addresses(addr).Build()
		logs, err := _api.GetLogs(f)
		require.NoError(t, err)
		require.Len(t, logs, testCase.nLogs,
			"from:%d,to:%d,n:%d", testCase.hFrom, testCase.hTo, testCase.nLogs)
	}
}

func TestGetLogs_OneTx(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createFiltersAPI(_app)

	blk := testutils.NewMdbBlockBuilder().
		Height(0x222ef).
		Hash(gethcmn.HexToHash("0x7b61ffc31c9cbf2365d76d406976cd00694879bdb4ecd7aaa2bde0a11bdf1a4b")).
		Tx(gethcmn.HexToHash("0x652e16e6f6d7c473488f6b95995dfe68ebb3b413d29f6422e676576eabf261b7"),
			types.Log{Address: gethcmn.HexToAddress("0xc801a4862e5c877e46065d8547fdb3220ff441f5"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
					gethcmn.HexToHash("0x0000000000000000000000002c4487b596b6034d6a8634616a8fd9934434d20b"),
					gethcmn.HexToHash("0x000000000000000000000000a112caaefecb231b91779a9e68c12080672fcc81"),
				},
			},
			types.Log{Address: gethcmn.HexToAddress("0xc801a4862e5c877e46065d8547fdb3220ff441f5"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"),
					gethcmn.HexToHash("0x0000000000000000000000002c4487b596b6034d6a8634616a8fd9934434d20b"),
					gethcmn.HexToHash("0x000000000000000000000000a207d13e6f65799c9ab42ade81ed49f05b3b6f5d"),
				},
			},
			types.Log{Address: gethcmn.HexToAddress("0x4272d9d470e71f00adb91fbf0ea8276959e4e15d"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
					gethcmn.HexToHash("0x0000000000000000000000002c4487b596b6034d6a8634616a8fd9934434d20b"),
					gethcmn.HexToHash("0x000000000000000000000000a112caaefecb231b91779a9e68c12080672fcc81"),
				},
			},
			types.Log{Address: gethcmn.HexToAddress("0x4272d9d470e71f00adb91fbf0ea8276959e4e15d"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"),
					gethcmn.HexToHash("0x0000000000000000000000002c4487b596b6034d6a8634616a8fd9934434d20b"),
					gethcmn.HexToHash("0x000000000000000000000000a207d13e6f65799c9ab42ade81ed49f05b3b6f5d"),
				},
			},
			types.Log{Address: gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
					gethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
					gethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
				},
				Data: []byte{0xDA, 0x01},
			},
			types.Log{Address: gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
					gethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"),
					gethcmn.HexToHash("0x0000000000000000000000002c4487b596b6034d6a8634616a8fd9934434d20b"),
				},
				Data: []byte{0xDA, 0x02},
			},
			types.Log{Address: gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1"),
					gethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
				},
				Data: []byte{0xDA, 0x03},
			},
			types.Log{Address: gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0x4c209b5fc8ad50758f13e2e1088ba56a560dff690a1c6fef26394f4c03821c4f"),
					gethcmn.HexToHash("0x000000000000000000000000a207d13e6f65799c9ab42ade81ed49f05b3b6f5d"),
				},
				Data: []byte{0xDA, 0x04},
			},
		).
		Build()
	//_app.AddBlocksToHistory(blk)
	addBlock(_app, blk)

	logs, err := _api.GetLogs(gethfilters.FilterCriteria{
		FromBlock: big.NewInt(1),
		ToBlock:   big.NewInt(0x222ef + 1),
		Addresses: []gethcmn.Address{gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81")},
		// Topics: [] “anything”
	})
	require.NoError(t, err)
	require.Len(t, logs, 4)
	require.Equal(t, []byte{0xDA, 0x01}, logs[0].Data)
	require.Equal(t, []byte{0xDA, 0x02}, logs[1].Data)
	require.Equal(t, []byte{0xDA, 0x03}, logs[2].Data)
	require.Equal(t, []byte{0xDA, 0x04}, logs[3].Data)

	logs, err = _api.GetLogs(gethfilters.FilterCriteria{
		FromBlock: big.NewInt(1),
		ToBlock:   big.NewInt(0x222ef + 1),
		Addresses: []gethcmn.Address{gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81")},
		// [A] “A in first position (and anything after)”
		Topics: [][]gethcmn.Hash{{gethcmn.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")}},
	})
	require.NoError(t, err)
	require.Len(t, logs, 2)
	require.Equal(t, []byte{0xDA, 0x01}, logs[0].Data)
	require.Equal(t, []byte{0xDA, 0x02}, logs[1].Data)

	logs, err = _api.GetLogs(gethfilters.FilterCriteria{
		FromBlock: big.NewInt(1),
		ToBlock:   big.NewInt(0x222ef + 1),
		Addresses: []gethcmn.Address{gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81")},
		// [null, B] “anything in first position AND B in second position (and anything after)”
		Topics: [][]gethcmn.Hash{{}, {gethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")}},
	})
	require.NoError(t, err)
	require.Len(t, logs, 2)
	require.Equal(t, []byte{0xDA, 0x01}, logs[0].Data)
	require.Equal(t, []byte{0xDA, 0x03}, logs[1].Data)

	logs, err = _api.GetLogs(gethfilters.FilterCriteria{
		FromBlock: big.NewInt(1),
		ToBlock:   big.NewInt(0x222ef + 1),
		Addresses: []gethcmn.Address{gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81")},
		// [A, B] “A in first position AND B in second position (and anything after)”
		Topics: [][]gethcmn.Hash{
			{gethcmn.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")},
			{gethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002")},
		},
	})
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, []byte{0xDA, 0x02}, logs[0].Data)

	logs, err = _api.GetLogs(gethfilters.FilterCriteria{
		FromBlock: big.NewInt(1),
		ToBlock:   big.NewInt(0x222ef + 1),
		Addresses: []gethcmn.Address{gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81")},
		// [[A, B], [A, B]] “(A OR B) in first position AND (A OR B) in second position (and anything after)”
		Topics: [][]gethcmn.Hash{
			{
				gethcmn.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
				gethcmn.HexToHash("0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1"),
			},
			{
				gethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
				gethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"),
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, logs, 3)
	require.Equal(t, []byte{0xDA, 0x01}, logs[0].Data)
	require.Equal(t, []byte{0xDA, 0x02}, logs[1].Data)
	require.Equal(t, []byte{0xDA, 0x03}, logs[2].Data)
}

func createFiltersAPI(_app *testutils.TestApp) PublicFilterAPI {
	backend := api.NewBackend(nil, _app.App)
	return NewAPI(backend, _app.Logger())
}

func addBlock(_app *testutils.TestApp, block *modbtypes.Block) {
	_app.BeginBlock(abci.RequestBeginBlock{
		Header: tmproto.Header{Height: block.Height},
	})
	_app.StoreBlocks(block)
	_app.AddBlockFotTest(block)
}

func newTopicsFilter(t *testing.T, _api PublicFilterAPI, topics [][]gethcmn.Hash) gethrpc.ID {
	id, err := _api.NewFilter(testutils.NewTopicsFilter(topics))
	require.NoError(t, err)
	require.NotEmpty(t, id)
	return id
}
