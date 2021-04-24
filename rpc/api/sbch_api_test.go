package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/internal/testutils"
)

func TestQueryTxBySrcDst(t *testing.T) {
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
	blk2 := testutils.NewMdbBlockBuilder().
		Height(2).Hash(gethcmn.Hash{0xB1, 0x23}).
		TxWithAddr(gethcmn.Hash{0xC5}, addr1, addr4).
		TxWithAddr(gethcmn.Hash{0xC6}, addr2, addr3).
		TxWithAddr(gethcmn.Hash{0xC7}, addr3, addr2).
		Build()

	ctx := _app.GetRunTxContext()
	ctx.StoreBlock(blk1)
	ctx.StoreBlock(blk2)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)
	time.Sleep(100 * time.Millisecond)

	testCases := []struct {
		queryBy string
		addr    gethcmn.Address
		startH  gethrpc.BlockNumber
		endH    gethrpc.BlockNumber
		retLen  int
	}{
		{"src", addr1, 1, 2, 2},
		{"src", addr1, 1, -1, 2},
		{"src", addr1, -1, -1, 1},
		{"dst", addr2, 1, 2, 2},
		{"dst", addr2, 1, -1, 2},
		{"dst", addr2, -1, -1, 1},
		{"addr", addr1, 1, 1, 1},
		{"addr", addr2, 1, 1, 2},
		{"addr", addr3, 1, 1, 2},
		//{"addr", addr4, 1, 1, 1},
	}
	for _, testCase := range testCases {
		switch testCase.queryBy {
		case "src":
			txs, err := _api.QueryTxBySrc(testCase.addr, testCase.startH, testCase.endH)
			require.NoError(t, err)
			require.Len(t, txs, testCase.retLen)
			for _, tx := range txs {
				require.Equal(t, testCase.addr, tx.From)
			}
		case "dst":
			txs, err := _api.QueryTxByDst(testCase.addr, testCase.startH, testCase.endH)
			require.NoError(t, err)
			require.Len(t, txs, testCase.retLen)
			for _, tx := range txs {
				require.Equal(t, testCase.addr, *tx.To)
			}
		default:
			txs, err := _api.QueryTxByAddr(testCase.addr, testCase.startH, testCase.endH)
			require.NoError(t, err)
			require.Len(t, txs, testCase.retLen)
			for _, tx := range txs {
				require.True(t, testCase.addr == tx.From || testCase.addr == *tx.To)
			}
		}
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

	ctx := _app.GetRunTxContext()
	ctx.StoreBlock(blk1)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)
	time.Sleep(100 * time.Millisecond)

	txs, err := _api.QueryTxByAddr(addr4, 1, 1)
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

	ctx := _app.GetRunTxContext()
	ctx.StoreBlock(blk1)
	ctx.StoreBlock(blk2)
	ctx.StoreBlock(blk3)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)
	time.Sleep(100 * time.Millisecond)

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

func TestGetToAddressCount(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, _ := testutils.GenKeyAndAddr()
	key3, _ := testutils.GenKeyAndAddr()
	key4, _ := testutils.GenKeyAndAddr()

	_app := testutils.CreateTestApp(key1, key2, key3, key4)
	defer _app.Destroy()
	_api := createSbchAPI(_app)

	_app.MakeAndExecTxInBlock(1, key2, 0, addr1, 123, nil)
	_app.MakeAndExecTxInBlock(3, key3, 1, addr1, 234, nil)
	_app.MakeAndExecTxInBlock(5, key4, 2, addr1, 345, nil)
	require.Equal(t, hexutil.Uint64(3), _api.GetToAddressCount(addr1))
}

func createSbchAPI(_app *testutils.TestApp) SbchAPI {
	backend := api.NewBackend(nil, _app.App)
	return newSbchAPI(backend)
}
