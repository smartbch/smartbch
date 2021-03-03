package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/moeing-chain/MoeingEVM/types"
	"github.com/moeing-chain/moeing-chain/api"
	"github.com/moeing-chain/moeing-chain/app"
	"github.com/moeing-chain/moeing-chain/internal/testutils"
)

func TestQueryTxBySrcDst(t *testing.T) {
	_app := app.CreateTestApp()
	defer app.DestroyTestApp(_app)
	_api := createMoeAPI(_app)

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

	ctx := _app.GetContext(app.RunTxMode)
	ctx.StoreBlock(blk1)
	ctx.StoreBlock(blk2)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)
	time.Sleep(100 * time.Millisecond)
	_app.SetCurrentBlock(&types.Block{Number: 2})

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

// TODO: fix me
func TestQueryTxByAddr(t *testing.T) {
	_app := app.CreateTestApp()
	defer app.DestroyTestApp(_app)
	_api := createMoeAPI(_app)

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

	ctx := _app.GetContext(app.RunTxMode)
	ctx.StoreBlock(blk1)
	ctx.StoreBlock(nil) // flush previous block
	ctx.Close(true)
	time.Sleep(100 * time.Millisecond)
	_app.SetCurrentBlock(&types.Block{Number: 1})

	txs, err := _api.QueryTxByAddr(addr4, 1, 1)
	require.NoError(t, err)
	for _, tx := range txs {
		require.Contains(t, []gethcmn.Address{tx.From, *tx.To}, addr4)
	}
	require.Len(t, txs, 1)
}

func createMoeAPI(_app *app.App) MoeAPI {
	backend := api.NewBackend(nil, _app)
	return newMoeAPI(backend)
}
