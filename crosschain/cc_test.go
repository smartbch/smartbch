package crosschain_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/crosschain"
	"github.com/smartbch/smartbch/internal/testutils"
)

func TestCC(t *testing.T) {
	key, sender := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	ctx := _app.GetRunTxContext()
	e := &crosschain.CcContractExecutor{}
	e.Init(ctx)

	// prepare
	senderBalance, _ := ctx.GetBalance(sender)
	acc := ctx.GetAccount(crosschain.CCContractAddress)
	initCCBalance := uint64(1000000)
	acc.UpdateBalance(uint256.NewInt(initCCBalance))
	ctx.SetAccount(crosschain.CCContractAddress, acc)

	txId1 := common.HexToHash("0x1")
	var vOutIndex [4]byte
	binary.BigEndian.PutUint32(vOutIndex[:], 10)
	var utxo1 [36]byte
	copy(utxo1[:], txId1[:])
	copy(utxo1[32:], vOutIndex[:])

	amount1 := uint256.NewInt(100)
	crosschain.SaveUTXO(ctx, utxo1, amount1)

	c := crosschain.PackTransferBCHToMainnet(utxo1)
	tx1 := &types.TxToRun{
		BasicTx: types.BasicTx{
			From:  sender,
			Value: amount1.Bytes32(),
			Data:  c,
		},
	}
	status, logs, _, _ := e.Execute(ctx, nil, tx1)
	require.Equal(t, crosschain.StatusSuccess, status)
	require.Equal(t, amount1.Uint64(), uint256.NewInt(0).SetBytes(logs[0].Data).Uint64())
	require.Equal(t, common.Address(crosschain.CCContractAddress), logs[0].Address)
	require.Equal(t, common.Hash(crosschain.HashOfEventTransferToBch), logs[0].Topics[0])
	require.Equal(t, txId1, logs[0].Topics[1])
	require.Equal(t, binary.BigEndian.Uint32(vOutIndex[:]), binary.BigEndian.Uint32(logs[0].Topics[2].Bytes()[32-4:]))
	require.True(t, bytes.Equal(sender.Bytes(), logs[0].Topics[3].Bytes()[32-20:]))

	amount := crosschain.LoadUTXO(ctx, utxo1)
	require.Equal(t, uint64(0), amount.Uint64())
	senderBalanceAfter, _ := ctx.GetBalance(sender)
	require.Equal(t, amount1.Uint64(), senderBalance.Sub(senderBalance, senderBalanceAfter).Uint64())
	ccBalanceAfter, _ := ctx.GetBalance(crosschain.CCContractAddress)
	require.Equal(t, amount1.Uint64(), ccBalanceAfter.Uint64()-initCCBalance)

	status, _, _, outData := e.Execute(ctx, nil, tx1)
	require.Equal(t, crosschain.StatusFailed, status)
	require.True(t, bytes.Equal([]byte(crosschain.BchAmountNotMatch.Error()), outData))

	txId2 := common.HexToHash("0x1")
	binary.BigEndian.PutUint32(vOutIndex[:], 20)
	var utxo2 [36]byte
	copy(utxo2[:], txId2[:])
	copy(utxo2[32:], vOutIndex[:])
	crosschain.SaveUTXO(ctx, utxo2, amount1)

	c = crosschain.PackBurnBCH(utxo2)
	tx2 := &types.TxToRun{
		BasicTx: types.BasicTx{
			From: sender,
			Data: c,
		},
	}
	_ = ebp.TransferFromSenderAccToBlackHoleAcc(ctx, sender, uint256.NewInt(1000))
	status, logs, _, _ = e.Execute(ctx, nil, tx2)
	require.Equal(t, crosschain.StatusSuccess, status)
	require.Equal(t, amount1.Uint64(), uint256.NewInt(0).SetBytes(logs[0].Data).Uint64())
	require.Equal(t, common.Hash(crosschain.HashOfEventBurn), logs[0].Topics[0])
	require.Equal(t, txId2, logs[0].Topics[1])
	require.Equal(t, binary.BigEndian.Uint32(vOutIndex[:]), binary.BigEndian.Uint32(logs[0].Topics[2].Bytes()[32-4:]))
}
