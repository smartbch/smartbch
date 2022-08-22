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

type MockClient struct {
	BlockInfos map[int64]*watchertypes.BlockInfo
}

func (m MockClient) GetLatestHeight(retry bool) int64 {
	return 0
}

func (m MockClient) GetBlockByHeight(height int64, retry bool) *watchertypes.BCHBlock {
	return nil
}

func (m MockClient) GetVoteInfoByEpochNumber(start, end uint64) []*watchertypes.VoteInfo {
	return nil
}

func (m MockClient) GetBlockInfoByHeight(height int64, retry bool) *watchertypes.BlockInfo {
	if info, ok := m.BlockInfos[height]; ok {
		return info
	}
	return &watchertypes.BlockInfo{}
}

func (m *MockClient) SetBlockInfoByHeight(height int64, info *watchertypes.BlockInfo) {
	m.BlockInfos[height] = info
}

var ABI = ethutils.MustParseABI(`
[
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
				"name": "newCovenantAddr",
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
		"inputs": [],
		"name": "handleUTXOs",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
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
	crosschain.MinCCAmount = 0
	key, alice := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()
	executor := crosschain.NewCcContractExecutor(log.NewNopLogger(), &crosschain.MockVoteContract{IsM: true})
	ebp.PredefinedContractManager[crosschain.CCContractAddress] = executor
	w := watcher.NewWatcher(log.NewNopLogger(), _app.HistoryStore(), param.EpochStartHeightForCC+1, 0, param.DefaultConfig())
	w.SetCCExecutor(executor)
	mockRpc := MockClient{make(map[int64]*watchertypes.BlockInfo)}
	w.SetRpcClient(mockRpc)

	txid := uint256.NewInt(100)
	index := big.NewInt(1)
	targetAddress := gethcmn.Address{0x01}
	value := uint256.NewInt(10)
	covenantAddress := [20]byte{0x1}

	go func() {
		w.CcContractExecutor.UTXOCollectDone <- []*types.CCTransferInfo{
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
		}
	}()

	ctx := _app.GetRunTxContext()
	ccCtx := types.CCContext{
		RescanTime:          0,
		RescanHeight:        0,
		LastRescannedHeight: 0,
		UTXOAlreadyHandled:  false,
		CurrCovenantAddr:    covenantAddress,
	}
	crosschain.SaveCCContext(ctx, ccCtx)
	acc := ctx.GetAccount(crosschain.CCContractAddress)
	ccBalance := uint256.NewInt(100)
	acc.UpdateBalance(ccBalance)
	ctx.SetAccount(crosschain.CCContractAddress, acc)
	ctx.Close(true)

	txData := PackHandleUTXOsFunc()
	tx, _ := _app.MakeAndExecTxInBlock(key, crosschain.CCContractAddress, int64(value.Uint64()), txData)
	_app.EnsureTxSuccess(tx.Hash())

	ids := _app.HistoryStore().GetAllUtxoIds()
	require.Equal(t, 1, len(ids))
	require.Equal(t, uint256.NewInt(0).SetBytes(ids[0][:32]).Uint64(), txid.Uint64())
	redeemableUtxos := _app.HistoryStore().GetRedeemableUtxoIds()
	require.Equal(t, 1, len(redeemableUtxos))

	txData = PackRedeemFunc(txid.ToBig(), index, targetAddress)
	tx, _ = _app.MakeAndExecTxInBlock(key, crosschain.CCContractAddress, int64(value.Uint64()), txData)
	_app.EnsureTxSuccess(tx.Hash())

	ids = _app.HistoryStore().GetAllUtxoIds()
	require.Equal(t, 1, len(ids))
	redeemableUtxos = _app.HistoryStore().GetRedeemableUtxoIds()
	require.Equal(t, 0, len(redeemableUtxos))
	redeemingU := _app.HistoryStore().GetRedeemingUtxoIds()
	require.Equal(t, 1, len(redeemingU))
}
