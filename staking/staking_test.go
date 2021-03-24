package staking_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartbch/moeingevm/types"
	"github.com/stretchr/testify/require"

	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/staking"
)

type callEntry struct {
	Address [20]byte
	Tx      *types.TxToRun
}

func buildCreateValCallEntry(sender common.Address, rewardTo byte, introduction byte, pubkey byte) *callEntry {
	c := &callEntry{
		Address: staking.StakingContractAddress,
		Tx:      nil,
	}
	c.Tx = &types.TxToRun{
		BasicTx: types.BasicTx{
			From:     sender,
			To:       c.Address,
		},
	}
	c.Tx.Value[31] = 100
	// createValidator(address rewardTo, bytes32 introduction, bytes32 pubkey)
	// data: (4B selector | 32B rewardTo | 1B introLen | introLen B introContent | 32B pubkey)
	c.Tx.Data = make([]byte, 0, 100)
	c.Tx.Data = append(c.Tx.Data, staking.SelectorCreateValidator[:]...)
	r := [32]byte{rewardTo}
	c.Tx.Data = append(c.Tx.Data, r[:]...)
	i := [31]byte{introduction}
	c.Tx.Data = append(c.Tx.Data, 31)
	c.Tx.Data = append(c.Tx.Data, i[:]...)
	p := [32]byte{pubkey}
	c.Tx.Data = append(c.Tx.Data, p[:]...)
	return c
}

func buildEditValCallEntry(sender common.Address, rewardTo byte, introduction byte) *callEntry {
	c := &callEntry{
		Address: staking.StakingContractAddress,
		Tx:      nil,
	}
	c.Tx = &types.TxToRun{
		BasicTx: types.BasicTx{
			From:     sender,
			To:       c.Address,
		},
	}
	c.Tx.Value[31] = 100
	// editValidator(address rewardTo, bytes32 introduction)
	// data: (4B selector | 32B rewardTo | 1B introLen | introLen B introContent)
	c.Tx.Data = make([]byte, 0, 100)
	c.Tx.Data = append(c.Tx.Data, staking.SelectorEditValidator[:]...)
	r := [32]byte{rewardTo}
	c.Tx.Data = append(c.Tx.Data, r[:]...)
	i := [31]byte{introduction}
	c.Tx.Data = append(c.Tx.Data, 31)
	c.Tx.Data = append(c.Tx.Data, i[:]...)
	return c
}

func buildUnboundValCallEntry(sender common.Address) *callEntry {
	c := &callEntry{
		Address: staking.StakingContractAddress,
		Tx:      nil,
	}
	c.Tx = &types.TxToRun{
		BasicTx: types.BasicTx{
			From:     sender,
			To:       c.Address,
		},
	}
	c.Tx.Value[31] = 100
	// unbound()
	// data: (4B selector)
	c.Tx.Data = make([]byte, 0, 100)
	c.Tx.Data = append(c.Tx.Data, staking.SelectorUnbond[:]...)
	return c
}

func TestStaking(t *testing.T) {
	key, sender := testutils.GenKeyAndAddr()
	_app := app.CreateTestApp(key)
	defer app.DestroyTestApp(_app)

	ctx := _app.GetContext(app.RunTxMode)
	e := &staking.StakingContractExecutor{}
	e.Init(ctx)
	// test create validator
	c := buildCreateValCallEntry(sender, 101, 11, 1)
	require.True(t, e.IsSystemContract(c.Address))
	e.Execute(*ctx, nil, c.Tx)
	// test edit validator
	c = buildEditValCallEntry(sender, 102, 12)
	e.Execute(*ctx, nil, c.Tx)
	// test unbound validator
	c = buildUnboundValCallEntry(sender)
	e.Execute(*ctx, nil, c.Tx)
}
