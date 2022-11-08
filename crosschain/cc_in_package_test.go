package crosschain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/smartbch/moeingads/store"
	"github.com/smartbch/moeingads/store/rabbit"
	mtypes "github.com/smartbch/moeingevm/types"
	"github.com/stretchr/testify/require"

	"github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/staking"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

func TestRedeem(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := mtypes.NewContext(&r, nil)
	// prepare cc context
	context := types.CCContext{
		LastRescannedHeight: 1,
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
		BornTime:         1,
		Txid:             txid,
		Index:            vout,
		Amount:           amount,
	}
	SaveUTXORecord(ctx, record)
	txData := PackRedeemFunc(big.NewInt(0).SetBytes(txid[:]), big.NewInt(int64(vout)), alice)
	// normal
	status, logs, _, outdata := redeem(ctx, &mtypes.BlockInfo{Timestamp: MatureTime + 2}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			From:  alice,
			Value: amount,
			Data:  txData,
			Gas:   GasOfCCOp,
		},
	})
	require.Equal(t, StatusSuccess, status)
	require.Equal(t, 1, len(logs))
	require.Equal(t, 0, len(outdata))
	ccAcc := ctx.GetAccount(CCContractAddress)
	require.Equal(t, uint256.NewInt(0).SetBytes(amount[:]).Uint64(), ccAcc.Balance().Uint64())
	// already redeemed
	status, _, _, outdata = redeem(ctx, &mtypes.BlockInfo{Timestamp: 0}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			From:  alice,
			Value: amount,
			Data:  txData,
			Gas:   GasOfCCOp,
		},
	})
	require.Equal(t, StatusFailed, status)
	require.Equal(t, ErrAlreadyRedeemed.Error(), string(outdata))
	// refresh record
	DeleteUTXORecord(ctx, txid, vout)
	SaveUTXORecord(ctx, record)
	// test lost and found utxo not found
	status, _, _, outdata = redeem(ctx, &mtypes.BlockInfo{Timestamp: 0}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			From:  alice,
			Value: uint256.NewInt(0).Bytes32(),
			Data:  txData,
			Gas:   GasOfCCOp,
		},
	})
	require.Equal(t, StatusFailed, status)
	require.Equal(t, ErrNotLostAndFound.Error(), string(outdata))
	// test redeem amount not match
	status, _, _, outdata = redeem(ctx, &mtypes.BlockInfo{Timestamp: 0}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			From:  alice,
			Value: uint256.NewInt(1).Bytes32(),
			Data:  txData,
			Gas:   GasOfCCOp,
		},
	})
	require.Equal(t, StatusFailed, status)
	require.Equal(t, ErrAmountNotMatch.Error(), string(outdata))
	// test lost and found
	// refresh record
	DeleteUTXORecord(ctx, txid, vout)
	record.OwnerOfLost = alice
	SaveUTXORecord(ctx, record)
	// test lost and found utxo not found
	status, _, _, _ = redeem(ctx, &mtypes.BlockInfo{Timestamp: 0}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			From:  alice,
			Value: uint256.NewInt(0).Bytes32(),
			Data:  txData,
			Gas:   GasOfCCOp,
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
		//PendingBurning:      [32]byte{0x1},
	}
	SaveCCContext(ctx, context)
	// prepare alice account
	alice := common.Address{0x01}
	// prepare redeemable utxo
	txid := [32]byte{0x1}
	vout := uint32(1)
	amount := uint256.NewInt(10).Bytes32()
	executor := CcContractExecutor{
		Voter: &MockVoteContract{},
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
	executor.Infos = infos
	status, logs, _, outdata := executor.handleUTXOs(ctx, &mtypes.BlockInfo{Timestamp: UTXOHandleDelay + 1}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			Gas: GasOfCCOp,
		},
	})
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
		//PendingBurning:      uint256.NewInt(2).Bytes32(),
	}
	SaveCCContext(ctx, context)
	// prepare utxo
	prevTxid := [32]byte{0x1}
	prevVout := uint32(1)
	prevAmount := uint256.NewInt(10).Bytes32()

	txid := [32]byte{0x02}
	vout := uint32(2)
	amount := uint256.NewInt(9).Bytes32()

	record := types.UTXORecord{
		Txid:     prevTxid,
		Index:    prevVout,
		Amount:   prevAmount,
		BornTime: 1,
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
	blackHoleContractAddress := [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		byte('b'), byte('l'), byte('a'), byte('c'), byte('k'), byte('h'), byte('o'), byte('l'), byte('e')}
	blackAcc := mtypes.ZeroAccountInfo()
	blackBalance := uint256.NewInt(1000)
	blackAcc.UpdateBalance(blackBalance)
	ctx.SetAccount(blackHoleContractAddress, blackAcc)

	logs := handleConvertTypeUTXO(ctx, &context, &info)
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
	staking.SaveStakingInfo(ctx, stakingtypes.StakingInfo{
		CurrEpochNum: param.StartEpochNumberForCC,
	})
	txData := PackStartRescanFunc(big.NewInt(2))
	// normal
	executor := CcContractExecutor{
		Voter:            &MockVoteContract{IsM: true},
		StartUTXOCollect: make(chan types.UTXOCollectParam),
	}
	go func(exe *CcContractExecutor) {
		<-exe.StartUTXOCollect
	}(&executor)
	status, logs, _, outdata := executor.startRescan(ctx, &mtypes.BlockInfo{Timestamp: UTXOHandleDelay + 1}, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			Data: txData,
			Gas:  GasOfCCOp,
		},
	})
	require.Equal(t, StatusSuccess, status)
	require.Equal(t, 0, len(outdata))
	require.Equal(t, 0, len(logs))
	loadCtx := LoadCCContext(ctx)
	require.Equal(t, int64(UTXOHandleDelay+1), loadCtx.RescanTime)
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
	txData := PackPauseFunc()
	// normal
	executor := CcContractExecutor{
		Voter: &MockVoteContract{IsM: true},
	}
	from := [20]byte{0x01}
	status, logs, _, outdata := executor.pause(ctx, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			Data: txData,
			Gas:  GasOfCCOp,
			From: from,
		},
	})
	require.Equal(t, StatusSuccess, status)
	require.Equal(t, 0, len(outdata))
	require.Equal(t, 0, len(logs))
	loadCtx := LoadCCContext(ctx)
	require.Equal(t, 1, len(loadCtx.MonitorsWithPauseCommand))
	require.Equal(t, from, loadCtx.MonitorsWithPauseCommand[0])
}

func TestResume(t *testing.T) {
	r := rabbit.NewRabbitStore(store.NewMockRootStore())
	ctx := mtypes.NewContext(&r, nil)

	from := [20]byte{0x01}

	// prepare cc context
	context := types.CCContext{
		RescanHeight:             1,
		MonitorsWithPauseCommand: [][20]byte{from},
	}
	SaveCCContext(ctx, context)
	txData := PackResumeFunc()
	// normal
	executor := CcContractExecutor{
		Voter: &MockVoteContract{IsM: true},
	}
	status, logs, _, outdata := executor.resume(ctx, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			Data: txData,
			Gas:  GasOfCCOp,
		},
	})
	require.Equal(t, StatusFailed, status)
	require.Equal(t, ErrMustPauseFirst.Error(), string(outdata))
	require.Equal(t, 0, len(logs))

	status, logs, _, outdata = executor.resume(ctx, &mtypes.TxToRun{
		BasicTx: mtypes.BasicTx{
			Data: txData,
			From: from,
			Gas:  GasOfCCOp,
		},
	})
	require.Equal(t, StatusSuccess, status)
	require.Equal(t, 0, len(outdata))
	require.Equal(t, 0, len(logs))
	loadCtx := LoadCCContext(ctx)
	require.Equal(t, 0, len(loadCtx.MonitorsWithPauseCommand))
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
	executor.handleOperatorOrMonitorSetChanged(ctx, nil, &context)
}
