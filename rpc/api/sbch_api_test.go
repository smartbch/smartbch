package api

import (
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	motypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/internal/testutils"
	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
)

func TestQueryTxBySrcDstAddr(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createSbchAPI(_app)

	addr1 := gethcmn.Address{0xA1}
	addr2 := gethcmn.Address{0xA2}
	addr3 := gethcmn.Address{0xA3}
	addr4 := gethcmn.Address{0xA4}

	blk1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		TxWithAddr(gethcmn.Hash{0xC1, 0x01}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC1, 0x02}, addr2, addr3).
		TxWithAddr(gethcmn.Hash{0xC1, 0x03}, addr3, addr4).
		Build()
	blk2 := testutils.NewMdbBlockBuilder().
		Height(2).Hash(gethcmn.Hash{0xB2}).
		TxWithAddr(gethcmn.Hash{0xC2, 0x01}, addr1, addr4).
		TxWithAddr(gethcmn.Hash{0xC2, 0x02}, addr2, addr3).
		TxWithAddr(gethcmn.Hash{0xC2, 0x03}, addr3, addr2).
		Build()

	_app.StoreBlocks(blk1, blk2)
	_app.WaitMS(100)

	testCases := []struct {
		queryBy  string
		addr     gethcmn.Address
		startH   gethrpc.BlockNumber
		endH     gethrpc.BlockNumber
		txHashes []gethcmn.Hash
	}{
		// startH <= endH
		{"src", addr1, 1, 2, []gethcmn.Hash{{0xC1, 0x01}, {0xC2, 0x01}}},
		{"src", addr1, 1, -1, []gethcmn.Hash{{0xC1, 0x01}, {0xC2, 0x01}}},
		{"src", addr1, -1, -1, []gethcmn.Hash{{0xC2, 0x01}}},
		{"dst", addr2, 1, 2, []gethcmn.Hash{{0xC1, 0x01}, {0xC2, 0x03}}},
		{"dst", addr2, 1, -1, []gethcmn.Hash{{0xC1, 0x01}, {0xC2, 0x03}}},
		{"dst", addr2, -1, -1, []gethcmn.Hash{{0xC2, 0x03}}},
		{"addr", addr1, 1, 1, []gethcmn.Hash{{0xC1, 0x01}}},
		{"addr", addr2, 1, 1, []gethcmn.Hash{{0xC1, 0x01}, {0xC1, 0x02}}},
		{"addr", addr3, 1, 1, []gethcmn.Hash{{0xC1, 0x02}, {0xC1, 0x03}}},
		{"addr", addr4, 1, 2, []gethcmn.Hash{{0xC1, 0x03}, {0xC2, 0x01}}},

		// startH > endH
		{"src", addr1, 2, 1, []gethcmn.Hash{{0xC2, 0x01}, {0xC1, 0x01}}},
		{"dst", addr2, 2, 1, []gethcmn.Hash{{0xC2, 0x03}, {0xC1, 0x01}}},
		{"addr", addr4, 2, 1, []gethcmn.Hash{{0xC2, 0x01}, {0xC1, 0x03}}},
	}
	for _, testCase := range testCases {
		switch testCase.queryBy {
		case "src":
			txs, err := _api.QueryTxBySrc(testCase.addr, testCase.startH, testCase.endH, 0)
			require.NoError(t, err)
			require.Len(t, txs, len(testCase.txHashes))
			for i, tx := range txs {
				require.Equal(t, testCase.addr, tx.From)
				require.Equal(t, testCase.txHashes[i], tx.Hash)
			}
		case "dst":
			txs, err := _api.QueryTxByDst(testCase.addr, testCase.startH, testCase.endH, 0)
			require.NoError(t, err)
			require.Len(t, txs, len(testCase.txHashes))
			for i, tx := range txs {
				require.Equal(t, testCase.addr, *tx.To)
				require.Equal(t, testCase.txHashes[i], tx.Hash)
			}
		default:
			txs, err := _api.QueryTxByAddr(testCase.addr, testCase.startH, testCase.endH, 0)
			require.NoError(t, err)
			require.Len(t, txs, len(testCase.txHashes))
			for i, tx := range txs {
				require.True(t, testCase.addr == tx.From || testCase.addr == *tx.To)
				require.Equal(t, testCase.txHashes[i], tx.Hash)
			}
		}
	}
}

func TestQueryTxBySrcDstAddr_limit(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createSbchAPI(_app)

	addr1 := gethcmn.Address{0xA1}
	addr2 := gethcmn.Address{0xA2}
	blk1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		TxWithAddr(gethcmn.Hash{0xC1}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC2}, addr2, addr1).
		TxWithAddr(gethcmn.Hash{0xC3}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC4}, addr2, addr1).
		TxWithAddr(gethcmn.Hash{0xC5}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC6}, addr2, addr1).
		TxWithAddr(gethcmn.Hash{0xC7}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC8}, addr2, addr1).
		TxWithAddr(gethcmn.Hash{0xC9}, addr1, addr2).
		Build()

	_app.StoreBlocks(blk1)

	testCases := []struct {
		queryBy string
		addr    gethcmn.Address
		startH  gethrpc.BlockNumber
		endH    gethrpc.BlockNumber
		limit   hexutil.Uint64
		nRet    int
	}{
		{"src", addr1, 1, 1, 0, 5},
		{"src", addr1, 1, 1, 3, 3},
		{"dst", addr1, 1, 1, 0, 4},
		{"dst", addr1, 1, 1, 2, 2},
		{"addr", addr1, 1, 1, 0, 9},
		{"addr", addr1, 1, 1, 7, 7},
	}

	for _, testCase := range testCases {
		var txs []*rpctypes.Transaction
		var err error
		switch testCase.queryBy {
		case "src":
			txs, err = _api.QueryTxBySrc(addr1, testCase.startH, testCase.endH, testCase.limit)
		case "dst":
			txs, err = _api.QueryTxByDst(addr1, testCase.startH, testCase.endH, testCase.limit)
		case "addr":
			txs, err = _api.QueryTxByAddr(addr1, testCase.startH, testCase.endH, testCase.limit)
		}

		require.NoError(t, err)
		require.Len(t, txs, testCase.nRet)
	}
}

func TestQueryTxByAddr(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createSbchAPI(_app)

	addr1 := gethcmn.Address{0xAD, 0x01}
	addr2 := gethcmn.Address{0xAD, 0x02}
	addr3 := gethcmn.Address{0xAD, 0x03}
	addr4 := gethcmn.Address{0xAD, 0x04}

	blk1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1, 0x23}).
		TxWithAddr(gethcmn.Hash{0xC1}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC2}, addr2, addr3).
		TxWithAddr(gethcmn.Hash{0xC3}, addr3, addr4).
		Build()

	_app.StoreBlocks(blk1)
	_app.WaitMS(100)

	txs, err := _api.QueryTxByAddr(addr4, 1, 1, 0)
	require.NoError(t, err)
	for _, tx := range txs {
		require.Contains(t, []gethcmn.Address{tx.From, *tx.To}, addr4)
	}
	require.Len(t, txs, 1)
}

func TestGetTxListByHeight(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createSbchAPI(_app)

	addr1 := gethcmn.Address{0xAD, 0x01}
	addr2 := gethcmn.Address{0xAD, 0x02}
	addr3 := gethcmn.Address{0xAD, 0x03}
	addr4 := gethcmn.Address{0xAD, 0x04}

	blk1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1, 0x23}).
		TxWithAddr(gethcmn.Hash{0xC1}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC2}, addr1, addr3).
		TxWithAddr(gethcmn.Hash{0xC3}, addr1, addr4).
		Build()

	blk2 := testutils.NewMdbBlockBuilder().
		Height(2).Hash(gethcmn.Hash{0xB2, 0x34}).
		TxWithAddr(gethcmn.Hash{0xC4}, addr2, addr4).
		TxWithAddr(gethcmn.Hash{0xC5}, addr2, addr3).
		Build()

	blk3 := testutils.NewMdbBlockBuilder().
		Height(3).Hash(gethcmn.Hash{0xB3, 0x45}).
		TxWithAddr(gethcmn.Hash{0xC6}, addr3, addr4).
		Build()

	_app.StoreBlocks(blk1, blk2, blk3)
	_app.WaitMS(100)

	txs, err := _api.GetTxListByHeight(1)
	require.NoError(t, err)
	require.Len(t, txs, 3)

	txs, err = _api.GetTxListByHeight(2)
	require.NoError(t, err)
	require.Len(t, txs, 2)

	txs, err = _api.GetTxListByHeight(3)
	require.NoError(t, err)
	require.Len(t, txs, 1)
}

func TestGetTxListByHeightWithRange(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createSbchAPI(_app)

	addr1 := gethcmn.Address{0xAD, 0x01}
	addr2 := gethcmn.Address{0xAD, 0x02}
	blk1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1, 0x23}).
		TxWithAddr(gethcmn.Hash{0xC1}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC2}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC3}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC4}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC5}, addr1, addr2).
		TxWithAddr(gethcmn.Hash{0xC6}, addr1, addr2).
		Build()

	_app.StoreBlocks(blk1)
	_app.WaitMS(100)

	txs, err := _api.GetTxListByHeightWithRange(1, 0, 3)
	require.NoError(t, err)
	require.Len(t, txs, 3)

	txs, err = _api.GetTxListByHeightWithRange(1, 1, 3)
	require.NoError(t, err)
	require.Len(t, txs, 2)

	txs, err = _api.GetTxListByHeightWithRange(1, 2, 1)
	require.NoError(t, err)
	require.Len(t, txs, 1)

	txs, err = _api.GetTxListByHeightWithRange(1, 2, 0)
	require.NoError(t, err)
	require.Len(t, txs, 4)

	txs, err = _api.GetTxListByHeightWithRange(1, 9, 10)
	require.NoError(t, err)
	require.Len(t, txs, 0)
}

func TestGetToAddressCount(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	key3, addr3 := testutils.GenKeyAndAddr()
	key4, addr4 := testutils.GenKeyAndAddr()

	_app := testutils.CreateTestApp(key1, key2, key3, key4)
	defer _app.Destroy()
	_api := createSbchAPI(_app)

	_app.MakeAndExecTxInBlock(key2, addr1, 123, nil)
	_app.MakeAndExecTxInBlock(key3, addr1, 234, nil)
	_app.MakeAndExecTxInBlock(key4, addr1, 345, nil)
	_app.WaitMS(200)
	require.Equal(t, hexutil.Uint64(3), _api.GetAddressCount("to", addr1))
	require.Equal(t, hexutil.Uint64(0), _api.GetAddressCount("to", addr2))
	require.Equal(t, hexutil.Uint64(0), _api.GetAddressCount("to", addr3))
	require.Equal(t, hexutil.Uint64(0), _api.GetAddressCount("to", addr4))
	require.Equal(t, hexutil.Uint64(3), _api.GetAddressCount("both", addr1))
	require.Equal(t, hexutil.Uint64(1), _api.GetAddressCount("both", addr2))
	require.Equal(t, hexutil.Uint64(1), _api.GetAddressCount("both", addr3))
	require.Equal(t, hexutil.Uint64(1), _api.GetAddressCount("both", addr4))
}

func TestQueryLogs(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createSbchAPI(_app)

	addr1 := gethcmn.Address{0xA1}
	addr2 := gethcmn.Address{0xA2}
	tx1Hash := gethcmn.Hash{0xC1}
	tx2Hash := gethcmn.Hash{0xC2}
	tx3Hash := gethcmn.Hash{0xC3}
	tx4Hash := gethcmn.Hash{0xC4}
	tx5Hash := gethcmn.Hash{0xC5}
	tx6Hash := gethcmn.Hash{0xC6}
	topic1 := gethcmn.Hash{0xD1}
	topic2 := gethcmn.Hash{0xD2}
	topic3 := gethcmn.Hash{0xD3}
	topic4 := gethcmn.Hash{0xD4}
	topic5 := gethcmn.Hash{0xD5}

	blk1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(tx1Hash, motypes.Log{Address: addr1, Topics: [][32]byte{topic1, topic2, topic3}}).
		Tx(tx2Hash, motypes.Log{Address: addr1, Topics: [][32]byte{topic1, topic4, topic5}}).
		Tx(tx3Hash, motypes.Log{Address: addr2, Topics: [][32]byte{topic2, topic3, topic4}}).
		Build()
	blk2 := testutils.NewMdbBlockBuilder().
		Height(2).Hash(gethcmn.Hash{0xB2}).
		Tx(tx4Hash, motypes.Log{Address: addr2, Topics: [][32]byte{topic2, topic4, topic5}}).
		Tx(tx5Hash, motypes.Log{Address: addr1, Topics: [][32]byte{topic1, topic3, topic2}}).
		Tx(tx6Hash, motypes.Log{Address: addr1, Topics: [][32]byte{topic1, topic3, topic4}}).
		Build()
	_app.AddBlocksToHistory(blk1, blk2)

	testCases := []struct {
		addr        gethcmn.Address
		topics      []gethcmn.Hash
		startH      gethrpc.BlockNumber
		endH        gethrpc.BlockNumber
		logTxHashes []gethcmn.Hash
	}{
		{addr1, []gethcmn.Hash{topic2, topic3}, 1, 3, []gethcmn.Hash{tx1Hash, tx5Hash}},
		{addr1, []gethcmn.Hash{topic2, topic3}, 3, 1, []gethcmn.Hash{tx5Hash, tx3Hash}},
	}

	for _, testCase := range testCases {
		logs, err := _api.QueryLogs(testCase.addr, testCase.topics, testCase.startH, testCase.endH, 0)
		require.NoError(t, err)
		require.Len(t, logs, len(testCase.logTxHashes))
	}
}

func TestQueryLogs_limit(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createSbchAPI(_app)

	addr1 := gethcmn.Address{0xA1}
	topic1 := gethcmn.Hash{0xD1}
	topic2 := gethcmn.Hash{0xD2}
	topic3 := gethcmn.Hash{0xD3}
	blk1 := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, motypes.Log{Address: addr1, Topics: [][32]byte{topic1}}).
		Tx(gethcmn.Hash{0xC2}, motypes.Log{Address: addr1, Topics: [][32]byte{topic1, topic2}}).
		Tx(gethcmn.Hash{0xC3}, motypes.Log{Address: addr1, Topics: [][32]byte{topic2, topic1}}).
		Tx(gethcmn.Hash{0xC4}, motypes.Log{Address: addr1, Topics: [][32]byte{topic1, topic2, topic3}}).
		Tx(gethcmn.Hash{0xC5}, motypes.Log{Address: addr1, Topics: [][32]byte{topic1, topic3, topic2}}).
		Tx(gethcmn.Hash{0xC6}, motypes.Log{Address: addr1, Topics: [][32]byte{topic2, topic1, topic3}}).
		Tx(gethcmn.Hash{0xC7}, motypes.Log{Address: addr1, Topics: [][32]byte{topic2, topic3, topic1}}).
		Tx(gethcmn.Hash{0xC8}, motypes.Log{Address: addr1, Topics: [][32]byte{topic3, topic1, topic2}}).
		Tx(gethcmn.Hash{0xC9}, motypes.Log{Address: addr1, Topics: [][32]byte{topic3, topic2, topic1}}).
		Build()
	_app.AddBlocksToHistory(blk1)

	logs, err := _api.QueryLogs(addr1, []gethcmn.Hash{topic1, topic2}, 1, 2, 0)
	require.NoError(t, err)
	require.Len(t, logs, 8)

	logs, err = _api.QueryLogs(addr1, []gethcmn.Hash{topic1, topic2}, 1, 2, 5)
	require.NoError(t, err)
	require.Len(t, logs, 5)
}

func TestQueryLogs_OneTx(t *testing.T) {
	_app := testutils.CreateTestApp()
	defer _app.Destroy()
	_api := createSbchAPI(_app)

	blk := testutils.NewMdbBlockBuilder().
		Height(0x222ef).
		Hash(gethcmn.HexToHash("0x7b61ffc31c9cbf2365d76d406976cd00694879bdb4ecd7aaa2bde0a11bdf1a4b")).
		Tx(gethcmn.HexToHash("0x652e16e6f6d7c473488f6b95995dfe68ebb3b413d29f6422e676576eabf261b7"),
			motypes.Log{Address: gethcmn.HexToAddress("0xc801a4862e5c877e46065d8547fdb3220ff441f5"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
					gethcmn.HexToHash("0x0000000000000000000000002c4487b596b6034d6a8634616a8fd9934434d20b"),
					gethcmn.HexToHash("0x000000000000000000000000a112caaefecb231b91779a9e68c12080672fcc81"),
				}},
			motypes.Log{Address: gethcmn.HexToAddress("0xc801a4862e5c877e46065d8547fdb3220ff441f5"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"),
					gethcmn.HexToHash("0x0000000000000000000000002c4487b596b6034d6a8634616a8fd9934434d20b"),
					gethcmn.HexToHash("0x000000000000000000000000a207d13e6f65799c9ab42ade81ed49f05b3b6f5d"),
				}},
			motypes.Log{Address: gethcmn.HexToAddress("0x4272d9d470e71f00adb91fbf0ea8276959e4e15d"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
					gethcmn.HexToHash("0x0000000000000000000000002c4487b596b6034d6a8634616a8fd9934434d20b"),
					gethcmn.HexToHash("0x000000000000000000000000a112caaefecb231b91779a9e68c12080672fcc81"),
				}},
			motypes.Log{Address: gethcmn.HexToAddress("0x4272d9d470e71f00adb91fbf0ea8276959e4e15d"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"),
					gethcmn.HexToHash("0x0000000000000000000000002c4487b596b6034d6a8634616a8fd9934434d20b"),
					gethcmn.HexToHash("0x000000000000000000000000a207d13e6f65799c9ab42ade81ed49f05b3b6f5d"),
				}},
			motypes.Log{Address: gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
					gethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
					gethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
				}},
			motypes.Log{Address: gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
					gethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
					gethcmn.HexToHash("0x0000000000000000000000002c4487b596b6034d6a8634616a8fd9934434d20b"),
				}},
			motypes.Log{Address: gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1"),
				}},
			motypes.Log{Address: gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81"),
				Topics: [][32]byte{
					gethcmn.HexToHash("0x4c209b5fc8ad50758f13e2e1088ba56a560dff690a1c6fef26394f4c03821c4f"),
					gethcmn.HexToHash("0x000000000000000000000000a207d13e6f65799c9ab42ade81ed49f05b3b6f5d"),
				}},
		).
		Build()
	_app.AddBlocksToHistory(blk)

	logs, err := _api.QueryLogs(gethcmn.HexToAddress("0xa112caaefecb231b91779a9e68c12080672fcc81"),
		[]gethcmn.Hash{}, 1, 0x222ef+1, 0)
	require.NoError(t, err)
	require.Len(t, logs, 4)

	//logsJson, _ := json.Marshal(logs)
	//require.Equal(t, "?", string(logsJson))
}

func createSbchAPI(_app *testutils.TestApp) SbchAPI {
	backend := api.NewBackend(nil, _app.App)
	return newSbchAPI(backend, _app.Logger())
}
