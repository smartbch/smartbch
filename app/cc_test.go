package app_test

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/smartbch/crosschain"
	"github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/watcher"
	watchertypes "github.com/smartbch/smartbch/watcher/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	"math/big"
	"testing"
)

var ABI = ethutils.MustParseABI(`
[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "address",
				"name": "oldCovenantAddr",
				"type": "address"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "newCovenantAddr",
				"type": "address"
			}
		],
		"name": "ChangeAddr",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "uint256",
				"name": "prevTxid",
				"type": "uint256"
			},
			{
				"indexed": true,
				"internalType": "uint32",
				"name": "prevVout",
				"type": "uint32"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "oldCovenantAddr",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"indexed": false,
				"internalType": "address",
				"name": "newCovenantAddr",
				"type": "address"
			}
		],
		"name": "Convert",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"indexed": true,
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "covenantAddr",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint8",
				"name": "sourceType",
				"type": "uint8"
			}
		],
		"name": "Deleted",
		"type": "event"
	},
	{
		"inputs": [],
		"name": "events",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "handleUTXOs",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"indexed": true,
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "covenantAddr",
				"type": "address"
			}
		],
		"name": "NewLostAndFound",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"indexed": true,
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "covenantAddr",
				"type": "address"
			}
		],
		"name": "NewRedeemable",
		"type": "event"
	},
	{
		"inputs": [],
		"name": "pause",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "index",
				"type": "uint256"
			},
			{
				"internalType": "address",
				"name": "targetAddress",
				"type": "address"
			}
		],
		"name": "redeem",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"indexed": true,
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "covenantAddr",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint8",
				"name": "sourceType",
				"type": "uint8"
			}
		],
		"name": "Redeem",
		"type": "event"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "mainFinalizedBlockHeight",
				"type": "uint256"
			}
		],
		"name": "startRescan",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]
`)

func PackRedeemFunc(txid, index *big.Int, targetAddress gethcmn.Address) []byte {
	return ABI.MustPack("redeem", txid, index, targetAddress)
}

func PackStartRescanFunc(mainFinalizedBlockHeight *big.Int) []byte {
	return ABI.MustPack("startRescan", mainFinalizedBlockHeight)
}

func PackPauseFunc() []byte {
	return ABI.MustPack("pause")
}

func PackHandleUTXOsFunc() []byte {
	return ABI.MustPack("handleUTXOs")
}

func TestCC(t *testing.T) {
	// init test app with shagate fork enabled
	key, alice := testutils.GenKeyAndAddr()
	initHeight := param.ShaGateForkBlock + 1
	_app := testutils.CreateTestAppWithArgs(testutils.TestAppInitArgs{
		StartHeight: &initHeight,
		InitAmt:     uint256.NewInt(0),
		PrivKeys:    []string{key},
	})
	defer _app.Destroy()
	// reset param
	crosschain.MinCCAmount = 0
	// self define executor
	executor := crosschain.NewCcContractExecutor(log.NewNopLogger(), &crosschain.MockVoteContract{IsM: true})
	ebp.PredefinedContractManager[crosschain.CCContractAddress] = executor
	// self define watcher
	w := watcher.NewWatcher(log.NewNopLogger(), _app.HistoryStore(), param.EpochStartHeightForCC+1, 0, param.DefaultConfig())
	w.SetCCExecutor(executor)
	mockRpc := watcher.MockClient{BlockInfos: make(map[int64]*watchertypes.BlockInfo)}
	w.SetRpcClient(mockRpc)
	// tx params
	txid := uint256.NewInt(100)
	index := big.NewInt(1)
	targetAddress := gethcmn.Address{0x01}
	value := uint256.NewInt(10)
	covenantAddress := [20]byte{0x1}

	txid1 := uint256.NewInt(200)
	index1 := big.NewInt(1)
	value1 := uint256.NewInt(11)

	w.CcContractExecutor.Infos = []*types.CCTransferInfo{
		{
			Type: types.TransferType,
			UTXO: types.UTXO{
				TxID:   txid.Bytes32(),
				Index:  uint32(index.Uint64()),
				Amount: value.Bytes32(),
			},
			Receiver:        alice,
			CovenantAddress: covenantAddress,
		},
		{
			Type: types.TransferType,
			UTXO: types.UTXO{
				TxID:   txid1.Bytes32(),
				Index:  uint32(index1.Uint64()),
				Amount: value1.Bytes32(),
			},
			Receiver:        alice,
			CovenantAddress: covenantAddress,
		},
	}
	// set cc context
	ctx := _app.GetRunTxContext()
	ccCtx := types.CCContext{
		CurrCovenantAddr: covenantAddress,
	}
	crosschain.SaveCCContext(ctx, ccCtx)
	// set cc account
	acc := ctx.GetAccount(crosschain.CCContractAddress)
	ccOriginValue := uint256.NewInt(100)
	ccBalance := ccOriginValue
	acc.UpdateBalance(ccBalance)
	ctx.SetAccount(crosschain.CCContractAddress, acc)
	ctx.Close(true)
	// call handleUTXO
	txData := PackHandleUTXOsFunc()
	tx, _ := _app.MakeAndExecTxInBlock(key, crosschain.CCContractAddress, int64(value.Uint64()), txData)
	_app.EnsureTxSuccess(tx.Hash())

	ids := _app.HistoryStore().GetAllUtxoIds()
	require.Equal(t, 2, len(ids))
	require.Equal(t, txid.Uint64(), uint256.NewInt(0).SetBytes(ids[0][:32]).Uint64())
	redeemableUtxos := _app.HistoryStore().GetRedeemableUtxoIds()
	require.Equal(t, 2, len(redeemableUtxos))
	require.Equal(t, txid.Uint64(), uint256.NewInt(0).SetBytes(redeemableUtxos[0][:32]).Uint64())
	require.Equal(t, txid1.Uint64(), uint256.NewInt(0).SetBytes(redeemableUtxos[1][:32]).Uint64())
	ctx = _app.GetRunTxContext()
	aliceBalance, _ := ctx.GetBalance(alice)
	require.Equal(t, value.Uint64()+value1.Uint64(), aliceBalance.Uint64())
	ccBalance, _ = ctx.GetBalance(crosschain.CCContractAddress)
	require.Equal(t, ccOriginValue.Uint64()-value1.Uint64()-value.Uint64(), ccBalance.Uint64())
	ctx.Close(false)

	// call redeem
	txData = PackRedeemFunc(txid.ToBig(), index, targetAddress)
	tx, _ = _app.MakeAndExecTxInBlock(key, crosschain.CCContractAddress, int64(value.Uint64()), txData)
	_app.EnsureTxSuccess(tx.Hash())

	ids = _app.HistoryStore().GetAllUtxoIds()
	require.Equal(t, 2, len(ids))
	redeemableUtxos = _app.HistoryStore().GetRedeemableUtxoIds()
	require.Equal(t, 1, len(redeemableUtxos))
	redeemingU := _app.HistoryStore().GetRedeemingUtxoIds()
	require.Equal(t, 1, len(redeemingU))
	require.Equal(t, txid.Uint64(), uint256.NewInt(0).SetBytes(redeemingU[0][:32]).Uint64())

	ctx = _app.GetRunTxContext()
	aliceBalance, _ = ctx.GetBalance(alice)
	require.Equal(t, value1.Uint64(), aliceBalance.Uint64())
	ccBalance, _ = ctx.GetBalance(crosschain.CCContractAddress)
	require.Equal(t, ccOriginValue.Uint64()-value1.Uint64(), ccBalance.Uint64())
	ctx.Close(false)

	// call handleUTXO to deal convert tx
	txid2 := uint256.NewInt(300)
	index2 := big.NewInt(1)
	value2 := uint256.NewInt(10)
	covenantAddress1 := [20]byte{0x2}

	w.CcContractExecutor.Infos = []*types.CCTransferInfo{
		{
			Type: types.ConvertType,
			UTXO: types.UTXO{
				TxID:   txid2.Bytes32(),
				Index:  uint32(index2.Uint64()),
				Amount: value2.Bytes32(),
			},
			PrevUTXO: types.UTXO{
				TxID:   txid1.Bytes32(),
				Index:  uint32(index1.Uint64()),
				Amount: value1.Bytes32(),
			},
			CovenantAddress: covenantAddress1,
		},
		{
			Type: types.RedeemOrLostAndFoundType,
			PrevUTXO: types.UTXO{
				TxID:   txid.Bytes32(),
				Index:  uint32(index.Uint64()),
				Amount: value.Bytes32(),
			},
		},
	}
	txData = PackHandleUTXOsFunc()
	tx, _ = _app.MakeAndExecTxInBlock(key, crosschain.CCContractAddress, int64(value.Uint64()), txData)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", crosschain.UTXOAlreadyHandled.Error())

	// reset context
	// set cc context
	ctx = _app.GetRunTxContext()
	ccCtx = types.CCContext{
		CurrCovenantAddr:   covenantAddress1,
		LastCovenantAddr:   covenantAddress,
		UTXOAlreadyHandled: false,
	}
	crosschain.SaveCCContext(ctx, ccCtx)
	ctx.Close(true)
	tx, _ = _app.MakeAndExecTxInBlock(key, crosschain.CCContractAddress, int64(value.Uint64()), txData)
	_app.EnsureTxFailedWithOutData(tx.Hash(), "failure", crosschain.PendingBurningNotEnough.Error())

	// increase enough pending burning
	ctx = _app.GetRunTxContext()
	blackHoleContractAddress := [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		byte('b'), byte('l'), byte('a'), byte('c'), byte('k'), byte('h'), byte('o'), byte('l'), byte('e')}
	blackAcc := ctx.GetAccount(blackHoleContractAddress)
	blackBalance := uint256.NewInt(1000)
	blackAcc.UpdateBalance(blackBalance)
	ctx.SetAccount(blackHoleContractAddress, blackAcc)
	ctx.Close(true)
	// test convert and delete redeeming
	tx, _ = _app.MakeAndExecTxInBlock(key, crosschain.CCContractAddress, int64(value.Uint64()), txData)
	_app.EnsureTxSuccess(tx.Hash())
	ids = _app.HistoryStore().GetAllUtxoIds()
	require.Equal(t, 1, len(ids))
	require.Equal(t, txid2.Uint64(), uint256.NewInt(0).SetBytes(ids[0][:32]).Uint64())
	redeemableUtxos = _app.HistoryStore().GetRedeemableUtxoIds()
	require.Equal(t, 1, len(ids))
	require.Equal(t, txid2.Uint64(), uint256.NewInt(0).SetBytes(redeemableUtxos[0][:32]).Uint64())
	ctx = _app.GetRunTxContext()
	ccC := crosschain.LoadCCContext(ctx)
	require.Equal(t, value1.Uint64()-value2.Uint64(), uint256.NewInt(0).SetBytes32(ccC.TotalBurntOnMainChain[:]).Uint64())
	ctx.Close(false)
}
