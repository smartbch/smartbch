package crosschain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/smartbch/moeingads/store"
	"github.com/smartbch/moeingads/store/rabbit"
	mtypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/crosschain/types"
)

func TestRedeem(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := mtypes.NewContext(&r, nil)
	// prepare cc context
	context := types.CCContext{
		LastRescannedHeight: 1,
		PendingBurning:      [32]byte{0x1},
	}
	SaveCCContext(ctx, context)
	// prepare alice account
	alice := common.Address{0x01}
	acc := mtypes.ZeroAccountInfo()
	acc.UpdateBalance(uint256.NewInt(100))
	ctx.SetAccount(alice, acc)
	// prepare redeemable utxo
	txid := [32]byte{0x1}
	vout := uint32(1)
	amount := uint256.NewInt(10).Bytes32()
	record := types.UTXORecord{
		OwnerOfLost:      [20]byte{},
		CovenantAddr:     [20]byte{},
		IsRedeemed:       false,
		RedeemTarget:     [20]byte{},
		ExpectedSignTime: 0,
		Txid:             txid,
		Index:            vout,
		Amount:           amount,
	}
	SaveUTXORecord(ctx, record)
	txData := PackRedeem(big.NewInt(0).SetBytes(txid[:]), big.NewInt(int64(vout)), alice)
	// normal
	status, logs, _, outdata := redeem(ctx, &mtypes.BlockInfo{Timestamp: 0}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			From:  alice,
			Value: amount,
			Data:  txData,
		},
	})
	require.Equal(t, StatusSuccess, status)
	require.Equal(t, 1, len(logs))
	require.Equal(t, 0, len(outdata))
	ccAcc := ctx.GetAccount(CCContractAddress)
	require.Equal(t, uint256.NewInt(0).SetBytes(amount[:]).Uint64(), ccAcc.Balance().Uint64())
	// already redeemed
	status, logs, _, outdata = redeem(ctx, &mtypes.BlockInfo{Timestamp: 0}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			From:  alice,
			Value: amount,
			Data:  txData,
		},
	})
	require.Equal(t, StatusFailed, status)
	require.Equal(t, AlreadyRedeemed.Error(), string(outdata))
	// refresh record
	DeleteUTXORecord(ctx, txid, vout)
	SaveUTXORecord(ctx, record)
	// test lost and found utxo not found
	status, logs, _, outdata = redeem(ctx, &mtypes.BlockInfo{Timestamp: 0}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			From:  alice,
			Value: uint256.NewInt(0).Bytes32(),
			Data:  txData,
		},
	})
	require.Equal(t, StatusFailed, status)
	require.Equal(t, NotLostAndFound.Error(), string(outdata))
	// test redeem amount not match
	status, logs, _, outdata = redeem(ctx, &mtypes.BlockInfo{Timestamp: 0}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			From:  alice,
			Value: uint256.NewInt(1).Bytes32(),
			Data:  txData,
		},
	})
	require.Equal(t, StatusFailed, status)
	require.Equal(t, AmountNotMatch.Error(), string(outdata))
	// test lost and found
	// refresh record
	DeleteUTXORecord(ctx, txid, vout)
	record.OwnerOfLost = alice
	SaveUTXORecord(ctx, record)
	// test lost and found utxo not found
	status, logs, _, outdata = redeem(ctx, &mtypes.BlockInfo{Timestamp: 0}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			From:  alice,
			Value: uint256.NewInt(0).Bytes32(),
			Data:  txData,
		},
	})
	require.Equal(t, StatusSuccess, status)
	loadU := LoadUTXORecord(ctx, txid, vout)
	require.Equal(t, [20]byte(alice), loadU.RedeemTarget)
}

func TestHandleUTXOs(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := mtypes.NewContext(&r, nil)
	// prepare cc context
	context := types.CCContext{
		LastRescannedHeight: 1,
		PendingBurning:      [32]byte{0x1},
	}
	SaveCCContext(ctx, context)
	// prepare alice account
	alice := common.Address{0x01}
	// prepare redeemable utxo
	txid := [32]byte{0x1}
	vout := uint32(1)
	amount := uint256.NewInt(10).Bytes32()
	executor := CcContractExecutor{
		UTXOCollectDone: make(chan []*types.CCTransferInfo),
		Voter:           &MockVoteContract{},
	}
	info := types.CCTransferInfo{
		Type: types.TransferType,
		UTXO: types.UTXO{
			TxID:   txid,
			Index:  vout,
			Amount: amount,
		},
		Receiver: alice,
	}
	var infos []*types.CCTransferInfo
	infos = append(infos, &info)
	go func(exe *CcContractExecutor) {
		exe.UTXOCollectDone <- infos
	}(&executor)
	status, logs, _, outdata := executor.handleUTXOs(ctx, &mtypes.BlockInfo{Timestamp: UTXOHandleDelay + 1}, nil)
	require.Equal(t, StatusSuccess, status)
	require.Equal(t, 1, len(logs))
	require.Equal(t, 0, len(outdata))
	record := LoadUTXORecord(ctx, txid, vout)
	require.Equal(t, [20]byte(alice), record.OwnerOfLost)
	loadCtx := LoadCCContext(ctx)
	require.Equal(t, true, loadCtx.UTXOAlreadyHandled)
}

func TestHandleConvertTypeUTXO(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := mtypes.NewContext(&r, nil)
	// prepare cc context
	context := types.CCContext{
		LastRescannedHeight: 1,
		PendingBurning:      uint256.NewInt(2).Bytes32(),
	}
	SaveCCContext(ctx, context)
	// prepare utxo
	prevTxid := [32]byte{0x1}
	prevVout := uint32(1)
	txid := [32]byte{0x02}
	vout := uint32(2)
	prevAmount := uint256.NewInt(10).Bytes32()
	amount := uint256.NewInt(9).Bytes32()
	record := types.UTXORecord{
		Txid:   prevTxid,
		Index:  prevVout,
		Amount: prevAmount,
	}
	SaveUTXORecord(ctx, record)
	info := types.CCTransferInfo{
		Type: types.ConvertType,
		PrevUTXO: types.UTXO{
			TxID:   prevTxid,
			Index:  prevVout,
			Amount: prevAmount,
		},
		UTXO: types.UTXO{
			TxID:   txid,
			Index:  vout,
			Amount: amount,
		},
	}
	logs := handleConvertTypeUTXO(ctx, &context, &info)
	require.Equal(t, uint64(1), uint256.NewInt(0).SetBytes32(context.PendingBurning[:]).Uint64())
	loadRecord := LoadUTXORecord(ctx, txid, vout)
	require.Equal(t, vout, loadRecord.Index)
	loadRecord = LoadUTXORecord(ctx, prevTxid, prevVout)
	require.Nil(t, loadRecord)
	require.Equal(t, 1, len(logs))
}

func TestHandleRedeemOrLostAndFoundTypeUTXO(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := mtypes.NewContext(&r, nil)
	// prepare cc context
	context := types.CCContext{
		LastRescannedHeight: 1,
	}
	SaveCCContext(ctx, context)
	// prepare utxo
	prevTxid := [32]byte{0x1}
	prevVout := uint32(1)
	prevAmount := uint256.NewInt(10).Bytes32()
	record := types.UTXORecord{
		IsRedeemed: true,
		Txid:       prevTxid,
		Index:      prevVout,
		Amount:     prevAmount,
	}
	SaveUTXORecord(ctx, record)
	info := types.CCTransferInfo{
		Type: types.ConvertType,
		PrevUTXO: types.UTXO{
			TxID:   prevTxid,
			Index:  prevVout,
			Amount: prevAmount,
		},
	}
	logs := handleRedeemOrLostAndFoundTypeUTXO(ctx, &context, &info)
	loadRecord := LoadUTXORecord(ctx, prevTxid, prevVout)
	require.Nil(t, loadRecord)
	require.Equal(t, 1, len(logs))
}

func TestStartRescan(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := mtypes.NewContext(&r, nil)
	// prepare cc context
	context := types.CCContext{
		RescanHeight: 1,
	}
	SaveCCContext(ctx, context)
	txData := PackStartRescan(big.NewInt(2))
	// normal
	executor := CcContractExecutor{
		Voter:            &MockVoteContract{IsM: true},
		StartUTXOCollect: make(chan types.UTXOCollectParam),
	}
	go func(exe *CcContractExecutor) {
		<-exe.StartUTXOCollect
	}(&executor)
	status, logs, _, outdata := executor.startRescan(ctx, &mtypes.BlockInfo{Timestamp: 100}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			Data: txData,
		},
	})
	require.Equal(t, StatusSuccess, status)
	require.Equal(t, 0, len(outdata))
	require.Equal(t, 0, len(logs))
	loadCtx := LoadCCContext(ctx)
	require.Equal(t, int64(100), loadCtx.RescanTime)
	require.Equal(t, uint64(1), loadCtx.LastRescannedHeight)
	require.Equal(t, uint64(2), loadCtx.RescanHeight)
	require.Equal(t, false, loadCtx.UTXOAlreadyHandled)
}

func TestPause(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := mtypes.NewContext(&r, nil)
	// prepare cc context
	context := types.CCContext{
		RescanHeight: 1,
	}
	SaveCCContext(ctx, context)
	txData := PackPause()
	// normal
	executor := CcContractExecutor{
		Voter: &MockVoteContract{IsM: true},
	}
	status, logs, _, outdata := executor.pause(ctx, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			Data: txData,
		},
	})
	require.Equal(t, StatusSuccess, status)
	require.Equal(t, 0, len(outdata))
	require.Equal(t, 0, len(logs))
	loadCtx := LoadCCContext(ctx)
	require.Equal(t, true, loadCtx.IsPaused)
}

func TestHandleOperatorOrMonitorSetChanged(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := mtypes.NewContext(&r, nil)
	// prepare cc context
	context := types.CCContext{}
	SaveCCContext(ctx, context)
	// normal
	executor := CcContractExecutor{
		Voter: &MockVoteContract{IsM: true},
	}
	executor.handleOperatorOrMonitorSetChanged(ctx, &context)
}
