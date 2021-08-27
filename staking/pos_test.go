package staking_test

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/staking"
)

// testdata/staking/contracts/XHedgeStorage.sol
var xhedgeStorageCreationBytecode = testutils.HexToBytes(`
608060405234801561001057600080fd5b5061023e806100206000396000f3fe
608060405234801561001057600080fd5b50600436106100415760003560e01c
806335aa2e4414610046578063430ce4f5146100765780639176355f146100a6
575b600080fd5b610060600480360381019061005b9190610158565b6100c256
5b60405161006d91906101cc565b60405180910390f35b610090600480360381
019061008b9190610158565b6100e6565b60405161009d91906101cc565b6040
5180910390f35b6100c060048036038101906100bb9190610181565b6100fe56
5b005b608881815481106100d257600080fd5b90600052602060002001600091
5090505481565b60876020528060005260406000206000915090505481565b60
8882908060018154018082558091505060019003906000526020600020016000
9091909190915055806087600084815260200190815260200160002081905550
5050565b600081359050610152816101f1565b92915050565b60006020828403
121561016a57600080fd5b600061017884828501610143565b91505092915050
565b6000806040838503121561019457600080fd5b60006101a2858286016101
43565b92505060206101b385828601610143565b9150509250929050565b6101
c6816101e7565b82525050565b60006020820190506101e160008301846101bd
565b92915050565b6000819050919050565b6101fa816101e7565b8114610205
57600080fd5b5056fea2646970667358221220e597c15f1e93800bfc8807c5d1
ffa0d278feb8ae58a7dece432cfe67a40447bf64736f6c63430008000033
`)

var xhedgeStorageABI = ethutils.MustParseABI(`
[
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "name": "valToVotes",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "name": "validators",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "val",
          "type": "uint256"
        },
        {
          "internalType": "uint256",
          "name": "votes",
          "type": "uint256"
        }
      ],
      "name": "addVal",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    }
  ]
`)

func TestGetAndClearPosVotes(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	tx, _, xhedgeAddr := _app.DeployContractInBlock(key, xhedgeStorageCreationBytecode)
	_app.EnsureTxSuccess(tx.Hash())
	xhedgeSeq := _app.GetSeq(xhedgeAddr)
	require.True(t, xhedgeSeq > 0)

	val1Key := big.NewInt(0x1234)
	val2Key := big.NewInt(0xABCD)
	val1Votes := uint256.NewInt(0).Mul(uint256.NewInt(2), staking.CoindayUnit).ToBig()
	val2Votes := uint256.NewInt(0).Mul(uint256.NewInt(3), staking.CoindayUnit).ToBig()

	data := xhedgeStorageABI.MustPack("addVal", val1Key, val1Votes)
	tx, _ = _app.MakeAndExecTxInBlock(key, xhedgeAddr, 0, data)
	_app.EnsureTxSuccess(tx.Hash())

	data = xhedgeStorageABI.MustPack("addVal", val2Key, val2Votes)
	tx, _ = _app.MakeAndExecTxInBlock(key, xhedgeAddr, 0, data)
	_app.EnsureTxSuccess(tx.Hash())

	result := _app.CallWithABI(addr, xhedgeAddr, xhedgeStorageABI, "validators", big.NewInt(0))
	require.Equal(t, []interface{}{val1Key}, result)
	result = _app.CallWithABI(addr, xhedgeAddr, xhedgeStorageABI, "validators", big.NewInt(1))
	require.Equal(t, []interface{}{val2Key}, result)

	result = _app.CallWithABI(addr, xhedgeAddr, xhedgeStorageABI, "valToVotes", val1Key)
	require.Equal(t, []interface{}{val1Votes}, result)
	result = _app.CallWithABI(addr, xhedgeAddr, xhedgeStorageABI, "valToVotes", val2Key)
	require.Equal(t, []interface{}{val2Votes}, result)

	ctx := _app.GetRunTxContext()
	posVotes := staking.GetAndClearPosVotes(ctx, xhedgeSeq)
	ctx.Close(true)
	require.Len(t, posVotes, 2)
	require.Equal(t, map[[32]byte]int64{
		uint256.NewInt(0x1234).Bytes32(): 2,
		uint256.NewInt(0xABCD).Bytes32(): 3,
	}, posVotes)

	_app.ExecTxsInBlock()
	require.Len(t, _app.GetDynamicArray(xhedgeAddr, staking.SlotValatorsArray), 0)
	result = _app.CallWithABI(addr, xhedgeAddr, xhedgeStorageABI, "valToVotes", val1Key)
	require.Equal(t, []interface{}{big.NewInt(1).SetInt64(0)}, result)
	result = _app.CallWithABI(addr, xhedgeAddr, xhedgeStorageABI, "valToVotes", val2Key)
	require.Equal(t, []interface{}{big.NewInt(1).SetInt64(0)}, result)

	ctx = _app.GetRunTxContext()
	posVotes = staking.GetAndClearPosVotes(ctx, xhedgeSeq)
	ctx.Close(true)
	require.Len(t, posVotes, 0)
}
