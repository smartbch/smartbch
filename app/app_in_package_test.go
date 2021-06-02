package app

import (
	"math/big"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/moeingevm/evmwrap/testcase"
	"github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/param"
)

var p *param.ChainConfig

func init() {
	p = param.DefaultConfig()
	p.ModbDataPath = "./testDb"
	p.AppDataPath = "./testAppDb"
}

func removeTestDB(_app *App) {
	_app.Stop()
	_ = os.RemoveAll(p.ModbDataPath)
	_ = os.RemoveAll(p.AppDataPath)
}

func TestAppReload(t *testing.T) {
	_app := NewApp(p, uint256.NewInt().SetUint64(1), log.NewNopLogger())
	defer removeTestDB(_app)
	_app.block = &types.Block{
		Number:           1,
		Hash:             [32]byte{0x01},
		ParentHash:       [32]byte{0x00},
		LogsBloom:        [256]byte{},
		TransactionsRoot: [32]byte{0x10},
		StateRoot:        [32]byte{0x11},
		Miner:            [20]byte{0x12},
		Size:             10,
		GasUsed:          100,
		Timestamp:        666,
		Transactions:     nil,
	}
	_app.reload()
	_app.mtx.Lock()
	_app.mtx.Unlock()
	bi := _app.blockInfo.Load().(*types.BlockInfo)
	require.Equal(t, _app.block.Number, bi.Number)
	require.Equal(t, _app.block.Timestamp, bi.Timestamp)
	require.Equal(t, _app.block.Hash, bi.Hash)
}

func TestAppInfo(t *testing.T) {
	_app := NewApp(p, uint256.NewInt().SetUint64(1), log.NewNopLogger())
	defer removeTestDB(_app)
	_app.block.Number = 1
	res := _app.Info(abcitypes.RequestInfo{})
	require.Equal(t, _app.block.Number, res.LastBlockHeight)
	require.Equal(t, _app.root.GetRootHash(), res.LastBlockAppHash)
}

func TestCheckTx(t *testing.T) {
	_app := NewApp(p, uint256.NewInt().SetUint64(1), log.NewNopLogger())
	_app.signer = &testcase.DumbSigner{}
	defer removeTestDB(_app)

	//test sigCache
	addr := common.Address{0x01}
	tx := ethutils.NewTx(0, &addr, big.NewInt(100), 100000, big.NewInt(10), nil)
	signedTx, _ := tx.WithSignature(_app.signer, addr.Bytes())
	data, _ := ethutils.EncodeTx(signedTx)
	r := abcitypes.RequestCheckTx{
		Tx:   data,
		Type: abcitypes.CheckTxType_New,
	}
	_app.CheckTx(r)
	require.Equal(t, _app.sigCache[signedTx.Hash()].Sender, addr)

	//test recheck counter
	r.Type = abcitypes.CheckTxType_Recheck
	_app.CheckTx(r)
	require.Equal(t, 1, _app.recheckCounter)

	//test refuse new tx
	_app.recheckThreshold = 0
	r.Type = abcitypes.CheckTxType_New
	res := _app.CheckTx(r)
	require.Equal(t, MempoolBusy, res.Code)

	//test sigCache clear
	_app.sigCacheSize = 0
	tx = ethutils.NewTx(1, &addr, big.NewInt(100), 100000, big.NewInt(10), nil)
	signedTx, _ = tx.WithSignature(_app.signer, addr.Bytes())
	data, _ = ethutils.EncodeTx(signedTx)
	r.Tx = data
	_app.CheckTx(r)
	require.Equal(t, 1, len(_app.sigCache))

	//test gas too large
	_app.recheckThreshold = 10
	tx = ethutils.NewTx(2, &addr, big.NewInt(100), ebp.MaxTxGasLimit+1, big.NewInt(10), nil)
	signedTx, _ = tx.WithSignature(_app.signer, addr.Bytes())
	data, _ = ethutils.EncodeTx(signedTx)
	r.Tx = data
	res = _app.CheckTx(r)
	require.Equal(t, GasLimitInvalid, res.Code)
}
