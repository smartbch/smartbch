package app_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/internal/testutils"
)

func TestDeployContract(t *testing.T) {
	key, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	// see testdata/counter/contracts/Counter.sol
	creationBytecode := testutils.HexToBytes(`
608060405234801561001057600080fd5b5060cc8061001f6000396000f3fe60
80604052348015600f57600080fd5b506004361060325760003560e01c806361
bc221a1460375780636299a6ef146053575b600080fd5b603d607e565b604051
8082815260200191505060405180910390f35b607c6004803603602081101560
6757600080fd5b81019080803590602001909291905050506084565b005b6000
5481565b8060008082825401925050819055505056fea2646970667358221220
37865cfcfd438966956583c78d31220c05c0f1ebfd116aced883214fcb1096c6
64736f6c634300060c0033
`)
	deployedBytecode := testutils.HexToBytes(`
6080604052348015600f57600080fd5b506004361060325760003560e01c8063
61bc221a1460375780636299a6ef146053575b600080fd5b603d607e565b6040
518082815260200191505060405180910390f35b607c60048036036020811015
606757600080fd5b81019080803590602001909291905050506084565b005b60
005481565b8060008082825401925050819055505056fea26469706673582212
2037865cfcfd438966956583c78d31220c05c0f1ebfd116aced883214fcb1096
c664736f6c634300060c0033
`)

	tx, _, contractAddr := _app.DeployContractInBlock(key, creationBytecode)
	require.Equal(t, deployedBytecode, _app.GetCode(contractAddr))
	txGot := _app.GetTx(tx.Hash())
	require.Equal(t, contractAddr, gethcmn.Address(txGot.ContractAddress))
}

func TestEmitLogs(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp0(bigutils.NewU256(1000000000), key)
	defer _app.Destroy()

	// see testdata/basic/contracts/Events.sol
	creationBytecode := testutils.HexToBytes(`
608060405234801561001057600080fd5b506101b6806100206000396000f3fe
608060405234801561001057600080fd5b50600436106100365760003560e01c
8063990ee4121461003b578063fb584c3914610045575b600080fd5b61004361
0061565b005b61005f600480360381019061005a919061010c565b6100a6565b
005b3373ffffffffffffffffffffffffffffffffffffffff167fd1c6b99eac4e
6a0f44c67915eb5195ecb58425668b0c7a46f58908541b5b2899604051604051
80910390a2565b3373ffffffffffffffffffffffffffffffffffffffff167f7a
2c2ad471d70e0a88640e6c3f4f5e975bcbccea7740c25631d0b74bb2c1cef482
6040516100ec9190610144565b60405180910390a250565b6000813590506101
0681610169565b92915050565b60006020828403121561011e57600080fd5b60
0061012c848285016100f7565b91505092915050565b61013e8161015f565b82
525050565b60006020820190506101596000830184610135565b92915050565b
6000819050919050565b6101728161015f565b811461017d57600080fd5b5056
fea2646970667358221220383eba178a868bbea24cfa4e229a163ffcf60cda8e
e7686360ba62da573cfb4864736f6c63430008000033
`)

	// deploy contract
	tx1, _, contractAddr := _app.DeployContractInBlock(key, creationBytecode)
	require.NotEmpty(t, _app.GetCode(contractAddr))
	_app.EnsureTxSuccess(tx1.Hash())

	// call emitEvent1()
	tx2, h2 := _app.MakeAndExecTxInBlock(key,
		contractAddr, 0, testutils.HexToBytes("990ee412"))

	_app.WaitMS(100)
	blk2 := _app.GetBlock(h2)
	require.Equal(t, h2, blk2.Number)
	require.Len(t, blk2.Transactions, 1)
	txInBlk3 := _app.GetTx(blk2.Transactions[0])
	require.Equal(t, gethtypes.ReceiptStatusSuccessful, txInBlk3.Status)
	require.Equal(t, tx2.Hash(), gethcmn.Hash(txInBlk3.Hash))
	require.Len(t, txInBlk3.Logs, 1)
	require.Len(t, txInBlk3.Logs[0].Topics, 2)
	require.Equal(t, "d1c6b99eac4e6a0f44c67915eb5195ecb58425668b0c7a46f58908541b5b2899",
		hex.EncodeToString(txInBlk3.Logs[0].Topics[0][:]))
	require.Equal(t, "000000000000000000000000"+hex.EncodeToString(addr[:]),
		hex.EncodeToString(txInBlk3.Logs[0].Topics[1][:]))

	// call emitEvent2()
	tx3, h3 := _app.MakeAndExecTxInBlock(key,
		contractAddr, 0, testutils.HexToBytes("0xfb584c39000000000000000000000000000000000000000000000000000000000000007b"))

	_app.WaitMS(100)
	blk3 := _app.GetBlock(h3)
	require.Equal(t, h3, blk3.Number)
	require.Len(t, blk3.Transactions, 1)
	txInBlk5 := _app.GetTx(blk3.Transactions[0])
	require.Equal(t, gethtypes.ReceiptStatusSuccessful, txInBlk5.Status)
	require.Equal(t, tx3.Hash(), gethcmn.Hash(txInBlk5.Hash))
	require.Len(t, txInBlk5.Logs, 1)
	require.Len(t, txInBlk5.Logs[0].Topics, 2)
	require.Equal(t, "7a2c2ad471d70e0a88640e6c3f4f5e975bcbccea7740c25631d0b74bb2c1cef4",
		hex.EncodeToString(txInBlk5.Logs[0].Topics[0][:]))
	require.Equal(t, "000000000000000000000000"+hex.EncodeToString(addr[:]),
		hex.EncodeToString(txInBlk5.Logs[0].Topics[1][:]))
	require.Equal(t, "000000000000000000000000000000000000000000000000000000000000007b",
		hex.EncodeToString(txInBlk5.Logs[0].Data))

	// test queryTxByAddr
	txs := _app.GetTxsByAddr(contractAddr)
	require.Equal(t, 2, len(txs))
}

func TestChainID(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	require.Equal(t, "0x1", _app.ChainID().String())

	// see testdata/basic/contracts/ChainID.sol
	creationBytecode := testutils.HexToBytes(`
608060405234801561001057600080fd5b5060b58061001f6000396000f3fe60
80604052348015600f57600080fd5b506004361060285760003560e01c806356
4b81ef14602d575b600080fd5b60336047565b604051603e9190605c565b6040
5180910390f35b600046905090565b6056816075565b82525050565b60006020
82019050606f6000830184604f565b92915050565b600081905091905056fea2
64697066735822122071af38cd4ec3657373c5944f6d44becf841a91b5a85545
7dfdabc41dd2e3b50064736f6c63430008000033
`)

	_, _, contractAddr := _app.DeployContractInBlock(key, creationBytecode)
	require.NotEmpty(t, _app.GetCode(contractAddr))

	_, _, output := _app.Call(addr, contractAddr, testutils.HexToBytes("564b81ef"))
	require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000001",
		hex.EncodeToString(output))
}

func TestRevert(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	// see testdata/basic/contracts/Errors.sol
	creationBytecode := testutils.HexToBytes(`
608060405234801561001057600080fd5b50610190806100206000396000f3fe
608060405234801561001057600080fd5b50600436106100415760003560e01c
806312f28d51146100465780632e52d60614610074578063e0ada09a14610092
575b600080fd5b6100726004803603602081101561005c57600080fd5b810190
80803590602001909291905050506100c0565b005b61007c6100d4565b604051
8082815260200191505060405180910390f35b6100be60048036036020811015
6100a857600080fd5b81019080803590602001909291905050506100da565b00
5b600a81106100ca57fe5b8060008190555050565b60005481565b600a811061
0150576040517f08c379a0000000000000000000000000000000000000000000
0000000000000081526004018080602001828103825260168152602001807f6e
206d757374206265206c657373207468616e2031300000000000000000000081
525060200191505060405180910390fd5b806000819055505056fea264697066
7358221220d21f014f3ec821cd1b466e1c9964010b3eb579a9153a8d63eb5116
b5007928aa64736f6c63430007000033
`)

	_, _, contractAddr := _app.DeployContractInBlock(key, creationBytecode)
	require.NotEmpty(t, _app.GetCode(contractAddr))

	// call setN_revert()
	callData := testutils.HexToBytes("0xe0ada09a0000000000000000000000000000000000000000000000000000000000000064")
	tx, _ := _app.MakeAndExecTxInBlock(key, contractAddr, 0, callData)

	_app.WaitMS(100)
	_app.EnsureTxFailed(tx.Hash(), "revert")

	statusCode, statusStr, retData := _app.Call(addr, contractAddr, callData)
	require.Equal(t, 2, statusCode)
	require.Equal(t, "revert", statusStr)
	require.Equal(t, "08c379a0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000166e206d757374206265206c657373207468616e20313000000000000000000000",
		hex.EncodeToString(retData))

	reason, errUnpack := abi.UnpackRevert(retData)
	require.NoError(t, errUnpack)
	require.Equal(t, "n must be less than 10", reason)
}

func TestInvalidOpcode(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	// see testdata/basic/contracts/Errors.sol
	creationBytecode := testutils.HexToBytes(`
608060405234801561001057600080fd5b50610190806100206000396000f3fe
608060405234801561001057600080fd5b50600436106100415760003560e01c
806312f28d51146100465780632e52d60614610074578063e0ada09a14610092
575b600080fd5b6100726004803603602081101561005c57600080fd5b810190
80803590602001909291905050506100c0565b005b61007c6100d4565b604051
8082815260200191505060405180910390f35b6100be60048036036020811015
6100a857600080fd5b81019080803590602001909291905050506100da565b00
5b600a81106100ca57fe5b8060008190555050565b60005481565b600a811061
0150576040517f08c379a0000000000000000000000000000000000000000000
0000000000000081526004018080602001828103825260168152602001807f6e
206d757374206265206c657373207468616e2031300000000000000000000081
525060200191505060405180910390fd5b806000819055505056fea264697066
7358221220d21f014f3ec821cd1b466e1c9964010b3eb579a9153a8d63eb5116
b5007928aa64736f6c63430007000033
`)

	_, _, contractAddr := _app.DeployContractInBlock(key, creationBytecode)
	require.NotEmpty(t, _app.GetCode(contractAddr))

	// call setN_invalidOpcode()
	callData := testutils.HexToBytes("0x12f28d510000000000000000000000000000000000000000000000000000000000000064")
	tx, _ := _app.MakeAndExecTxInBlock(key, contractAddr, 0, callData)

	_app.WaitMS(100)
	_app.EnsureTxFailed(tx.Hash(), "invalid-instruction")

	statusCode, statusStr, _ := _app.Call(addr, contractAddr, callData)
	require.Equal(t, 4, statusCode)
	require.Equal(t, "invalid-instruction", statusStr)
}

func TestEstimateGas(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	require.Equal(t, "0x1", _app.ChainID().String())

	// see testdata/basic/contracts/ChainID.sol
	creationBytecode := testutils.HexToBytes(`
608060405234801561001057600080fd5b5060b58061001f6000396000f3fe60
80604052348015600f57600080fd5b506004361060285760003560e01c806356
4b81ef14602d575b600080fd5b60336047565b604051603e9190605c565b6040
5180910390f35b600046905090565b6056816075565b82525050565b60006020
82019050606f6000830184604f565b92915050565b600081905091905056fea2
64697066735822122071af38cd4ec3657373c5944f6d44becf841a91b5a85545
7dfdabc41dd2e3b50064736f6c63430008000033
`)

	tx1, _, contractAddr := _app.DeployContractInBlock(key, creationBytecode)
	require.NotEmpty(t, _app.GetCode(contractAddr))

	statusCode, statusStr, gas := _app.EstimateGas(addr, tx1)
	require.Equal(t, 0, statusCode)
	require.Equal(t, "success", statusStr)
	require.True(t, gas > 0)
}
var testAddABI = testutils.MustParseABI(`
[
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "to",
          "type": "address"
        },
        {
          "internalType": "uint256",
          "name": "param",
          "type": "uint256"
        }
      ],
      "name": "run",
      "outputs": [],
      "stateMutability": "payable",
      "type": "function"
    }
]
`)

func TestContractAdd(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()
	// see testdata/basic/contracts/TestAdd.sol
	creationBytecode := testutils.HexToBytes(`
	608060405234801561001057600080fd5b50610264806100206000396000f3fe60806040526004361061001e5760003560e01c8063381fd19014610023575b600080fd5b61003661003136600461017b565b610038565b005b604080516000815260208101918290526001600160a01b0384169161232891349161006391906101b1565b600060405180830381858888f193505050503d80600081146100a1576040519150601f19603f3d011682016040523d82523d6000602084013e6100a6565b606091505b505050602081811c63ffffffff81811660009081529283905260408084205491851684529283902054849384901c91606085901c91608086901c9160a087901c9160029134916100f691906101ea565b61010091906101ea565b61010a919061020e565b63ffffffff8086166000908152602081905260408082209390935584821681528281205491861681529190912054600291349161014791906101ea565b61015191906101ea565b61015b919061020e565b63ffffffff90911660009081526020819052604090205550505050505050565b6000806040838503121561018d578182fd5b82356001600160a01b03811681146101a3578283fd5b946020939093013593505050565b60008251815b818110156101d157602081860181015185830152016101b7565b818111156101df5782828501525b509190910192915050565b6000821982111561020957634e487b7160e01b81526011600452602481fd5b500190565b60008261022957634e487b7160e01b81526012600452602481fd5b50049056fea2646970667358221220e3b7b62d342ecf70509d4193520e8f9d643096c2ff67a083c542c4ae01cbd23a64736f6c63430008000033
`)

	_, _, contractAddr := _app.DeployContractInBlock(key1, creationBytecode)
	require.NotEmpty(t, _app.GetCode(contractAddr))

	param := big.NewInt(1)
	param.Lsh(param, 32); param.Or(param, big.NewInt(2))
	param.Lsh(param, 32); param.Or(param, big.NewInt(3))
	param.Lsh(param, 32); param.Or(param, big.NewInt(4))
	param.Lsh(param, 32); param.Or(param, big.NewInt(5))
	param.Lsh(param, 32); param.Or(param, big.NewInt(6))
	calldata := testAddABI.MustPack("run", addr2, param)
	_app.MakeAndExecTxInBlockWithGasPrice(key1, contractAddr, 100/*value*/, calldata, 2/*gasprice*/)

	ctx := _app.GetCheckTxContext()
	defer ctx.Close(false)
	fmt.Printf("addr1's balance %d\n", ctx.GetAccount(addr1).Balance().Uint64())
	fmt.Printf("addr2's balance %d\n", ctx.GetAccount(addr2).Balance().Uint64())
}




