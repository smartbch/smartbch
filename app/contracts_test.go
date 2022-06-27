package app_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
)

func TestDeployContract(t *testing.T) {
	key, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	// see testdata/sol/contracts/basic/Counter.sol
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
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	// see testdata/sol/contracts/basic/EventEmitter.sol
	creationBytecode := testutils.HexToBytes(`
608060405234801561001057600080fd5b506102c2806100206000396000f3fe
608060405234801561001057600080fd5b50600436106100415760003560e01c
80630a13271414610046578063990ee41214610050578063fb584c391461005a
575b600080fd5b61004e610076565b005b6100586100fc565b005b6100746004
80360381019061006f91906101a7565b610141565b005b3373ffffffffffffff
ffffffffffffffffffffffffff167fd1c6b99eac4e6a0f44c67915eb5195ecb5
8425668b0c7a46f58908541b5b289960405160405180910390a260006100fa57
6040517f08c379a0000000000000000000000000000000000000000000000000
0000000081526004016100f19061021f565b60405180910390fd5b565b3373ff
ffffffffffffffffffffffffffffffffffffff167fd1c6b99eac4e6a0f44c679
15eb5195ecb58425668b0c7a46f58908541b5b289960405160405180910390a2
565b3373ffffffffffffffffffffffffffffffffffffffff167f7a2c2ad471d7
0e0a88640e6c3f4f5e975bcbccea7740c25631d0b74bb2c1cef4826040516101
87919061023f565b60405180910390a250565b6000813590506101a181610275
565b92915050565b6000602082840312156101b957600080fd5b60006101c784
828501610192565b91505092915050565b60006101dd60038361025a565b9150
7f7e7e7e00000000000000000000000000000000000000000000000000000000
006000830152602082019050919050565b6102198161026b565b82525050565b
60006020820190508181036000830152610238816101d0565b9050919050565b
60006020820190506102546000830184610210565b92915050565b6000828252
60208201905092915050565b6000819050919050565b61027e8161026b565b81
1461028957600080fd5b5056fea264697066735822122066b324a3c8bb4c14b7
7375d9a272751b053088f4c30d941dfa7727b2e7a3919764736f6c6343000800
0033
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

	// call emitEvent1ThenRevert()
	tx4, _ := _app.MakeAndExecTxInBlock(key,
		contractAddr, 0, testutils.HexToBytes("0a132714"))
	_app.EnsureTxFailed(tx4.Hash(), "revert")
	tx4QueryResult := _app.GetTx(tx4.Hash())
	require.Len(t, tx4QueryResult.Logs, 0)
}

func TestChainID(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	require.Equal(t, "0x1", _app.ChainID().String())

	// see testdata/sol/contracts/basic/ChainID.sol
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

	// see testdata/sol/contracts/basic/Errors.sol
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

	// see testdata/sol/contracts/basic/Errors.sol
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

	// see testdata/sol/contracts/basic/ChainID.sol
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

var testAddABI = ethutils.MustParseABI(`
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
    },
    {
      "inputs": [
        {
          "internalType": "uint32",
          "name": "d",
          "type": "uint32"
        }
      ],
      "name": "get",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "nonpayable",
      "type": "function"
    }
]
`)

func TestContractAdd(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()
	// see testdata/sol/contracts/basic/TestAdd.sol
	creationBytecode := testutils.HexToBytes(`
	608060405234801561001057600080fd5b506102f1806100206000396000f3fe6080604052600436106100295760003560e01c8063381fd1901461002e578063d8a26e3a14610043575b600080fd5b61004161003c3660046101d4565b610079565b005b34801561004f57600080fd5b5061006361005e36600461020a565b6101bc565b604051610070919061026e565b60405180910390f35b604080516000815260208101918290526001600160a01b038416916123289134916100a49190610235565b600060405180830381858888f193505050503d80600081146100e2576040519150601f19603f3d011682016040523d82523d6000602084013e6100e7565b606091505b505050602081811c63ffffffff81811660009081529283905260408084205491851684529283902054849384901c91606085901c91608086901c9160a087901c9160029134916101379190610277565b6101419190610277565b61014b919061029b565b63ffffffff808616600090815260208190526040808220939093558482168152828120549186168152919091205460029134916101889190610277565b6101929190610277565b61019c919061029b565b63ffffffff90911660009081526020819052604090205550505050505050565b63ffffffff1660009081526020819052604090205490565b600080604083850312156101e6578182fd5b82356001600160a01b03811681146101fc578283fd5b946020939093013593505050565b60006020828403121561021b578081fd5b813563ffffffff8116811461022e578182fd5b9392505050565b60008251815b81811015610255576020818601810151858301520161023b565b818111156102635782828501525b509190910192915050565b90815260200190565b6000821982111561029657634e487b7160e01b81526011600452602481fd5b500190565b6000826102b657634e487b7160e01b81526012600452602481fd5b50049056fea2646970667358221220e66a2e809beccf7e9d31ac11ec547ae76cd13f0c56d68a51d2da6224c706fdd864736f6c63430008000033
`)

	_, _, contractAddr := _app.DeployContractInBlock(key1, creationBytecode)
	require.NotEmpty(t, _app.GetCode(contractAddr))

	param := big.NewInt(1)
	param.Lsh(param, 32)
	param.Or(param, big.NewInt(2))
	param.Lsh(param, 32)
	param.Or(param, big.NewInt(3))
	param.Lsh(param, 32)
	param.Or(param, big.NewInt(4))
	param.Lsh(param, 32)
	param.Or(param, big.NewInt(5))
	param.Lsh(param, 32)
	param.Or(param, big.NewInt(6))
	calldata := testAddABI.MustPack("run", addr2, param)
	_app.MakeAndExecTxInBlockWithGas(key1, contractAddr, 120 /*value*/, calldata, testutils.DefaultGasLimit, 2 /*gasprice*/)

	ctx := _app.GetRpcContext()
	//conAcc := ctx.GetAccount(contractAddr)
	//seq := conAcc.Sequence()
	//fmt.Printf("conAcc's Sequence %d\n", seq)
	fmt.Printf("addr1's balance %d\n", ctx.GetAccount(addr1).Balance().Uint64())
	require.Equal(t, uint64(120), ctx.GetAccount(addr2).Balance().Uint64())
	ctx.Close(false)

	res := [7]int64{-1, 60, 0, 0, 60, 0, 0}
	for i := uint32(1); i <= 6; i++ {
		data := testAddABI.MustPack("get", i)
		status, statusStr, retData := _app.Call(addr1, contractAddr, data)
		n := big.NewInt(0)
		n.SetBytes(retData)
		require.Equal(t, 0, status)
		require.Equal(t, "success", statusStr)
		require.Equal(t, res[i], n.Int64())
	}
	_app.ExecTxInBlock(nil)
	_app.MakeAndExecTxInBlockWithGas(key1, contractAddr, 2 /*value*/, calldata, testutils.DefaultGasLimit, 1 /*gasprice*/)
	_app.ExecTxInBlock(nil)
	ctx = _app.GetRpcContext()
	require.Equal(t, uint64(122), ctx.GetAccount(addr2).Balance().Uint64())
	fmt.Printf("addr1's balance %d\n", ctx.GetAccount(addr1).Balance().ToBig())
	ctx.Close(false)
}

func TestEIP3541(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()

	// https://eips.ethereum.org/EIPS/eip-3541#test-cases
	tx, _, _ := _app.DeployContractInBlock(key1, testutils.HexToBytes("0x60ef60005360016000f3"))
	_app.EnsureTxFailed(tx.Hash(), "failure")

	tx, _, _ = _app.DeployContractInBlock(key1, testutils.HexToBytes("0x60ef60005360026000f3"))
	_app.EnsureTxFailed(tx.Hash(), "failure")

	tx, _, _ = _app.DeployContractInBlock(key1, testutils.HexToBytes("0x60ef60005360036000f3"))
	_app.EnsureTxFailed(tx.Hash(), "failure")

	tx, _, _ = _app.DeployContractInBlock(key1, testutils.HexToBytes("0x60ef60005360206000f3"))
	_app.EnsureTxFailed(tx.Hash(), "failure")

	tx, _, addr := _app.DeployContractInBlock(key1, testutils.HexToBytes("0x60fe60005360016000f3"))
	_app.EnsureTxSuccess(tx.Hash())
	require.Equal(t, "fe", hex.EncodeToString(_app.GetCode(addr)))
}

func TestCallPrecompileFromEOA(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1)
	defer _app.Destroy()

	sha256Addr := gethcmn.BytesToAddress([]byte{0x02})
	tx, _ := _app.MakeAndExecTxInBlock(key1, sha256Addr, 0, testutils.HexToBytes("0x1234"))
	_app.EnsureTxSuccess(tx.Hash())
}

func TestGetCodeBug(t *testing.T) {
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

	//key, _ := testutils.GenKeyAndAddr()
	key := "7648adfae1b87581aa90509d64556138b463d8b6dded677455687cb395cf6cfa"

	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	tx, _, contractAddr := _app.DeployContractInBlock(key, creationBytecode)
	_app.EnsureTxSuccess(tx.Hash())
	require.NotEmpty(t, _app.GetCode(contractAddr))
}

func TestBlockHashBug(t *testing.T) {
	_abi := ethutils.MustParseABI(`
[
    {
      "inputs": [],
      "name": "lastBlockHash",
      "outputs": [
        {
          "internalType": "bytes32",
          "name": "",
          "type": "bytes32"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "saveLastBlockHash",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    }
  ]
`)

	// testdata/sol/contracts/basic/BlockHash2.sol
	creationBytecode := testutils.HexToBytes(`
608060405234801561001057600080fd5b50610156806100206000396000f3fe
608060405234801561001057600080fd5b50600436106100365760003560e01c
806336cd93951461003b5780635c0ecfad14610045575b600080fd5b61004361
0063565b005b61004d610079565b60405161005a919061008e565b6040518091
0390f35b60014361007091906100a9565b40600081905550565b60005481565b
610088816100dd565b82525050565b60006020820190506100a3600083018461
007f565b92915050565b60006100b4826100e7565b91506100bf836100e7565b
9250828210156100d2576100d16100f1565b5b828203905092915050565b6000
819050919050565b6000819050919050565b7f4e487b71000000000000000000
00000000000000000000000000000000000000600052601160045260246000fd
fea26469706673582212205b27aaf95d4dba4e9bc245e37a361d0750a786f092
bd0c4e850d868d2f7aa12664736f6c63430008000033
`)

	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	tx, _, contractAddr := _app.DeployContractInBlock(key, creationBytecode)
	_app.EnsureTxSuccess(tx.Hash())

	saveLastBlockHashInput := _abi.MustPack("saveLastBlockHash")
	getLastBlockHashInput := _abi.MustPack("lastBlockHash")

	for i := 0; i < 10; i++ {
		tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, saveLastBlockHashInput)
		_app.EnsureTxSuccess(tx.Hash())
		statusCode, _, retData := _app.Call(addr, contractAddr, getLastBlockHashInput)
		require.Equal(t, 0, statusCode)
		println(hex.EncodeToString(retData))
		require.False(t, uint256.NewInt(0).SetBytes32(retData).IsZero())
	}
}

/*
//SPDX-License-Identifier: Unlicense
pragma solidity ^0.8.0;

contract CCVotingForTest {

	struct OperatorInfo {
		address owner;
		uint    pubkeyPrefix; // 0x02 or 0x03 (TODO: change type to uint8)
		bytes32 pubkeyX;      // x
		bytes32 rpcUrl;       // ip:port
		bytes32 intro;
		uint    votes;
		uint    inOfficeStartTime; // 0 means not in office
	}

	struct VoterInfo {
		uint lastUpdatedTime;
		uint coinDays;
		uint stakedAmount;
	}

	uint currOperatorCount; // set from Go
	OperatorInfo[] operators;
	mapping(address => VoterInfo) voters;

	function addOperator(uint pubkeyPrefix,
		                 bytes32 pubkeyX,
		                 bytes32 rpcUrl,
		                 bytes32 intro) public {
		operators.push(OperatorInfo(msg.sender, pubkeyPrefix, pubkeyX, rpcUrl, intro, 0, 0));
	}

	function getOperator(uint idx) public returns (address, uint, bytes32, bytes32, bytes32, uint, uint) {
		OperatorInfo storage operator = operators[idx];
		return (operator.owner, operator.pubkeyPrefix, operator.pubkeyX, operator.rpcUrl, operator.intro,
			operator.votes, operator.inOfficeStartTime);
	}

}
*/
func TestCCVoting(t *testing.T) {
	_abi := ethutils.MustParseABI(`
[
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "pubkeyPrefix",
          "type": "uint256"
        },
        {
          "internalType": "bytes32",
          "name": "pubkeyX",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "rpcUrl",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "intro",
          "type": "bytes32"
        }
      ],
      "name": "addOperator",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "idx",
          "type": "uint256"
        }
      ],
      "name": "getOperator",
      "outputs": [
        {
          "internalType": "address",
          "name": "",
          "type": "address"
        },
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        },
        {
          "internalType": "bytes32",
          "name": "",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "",
          "type": "bytes32"
        },
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        },
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "nonpayable",
      "type": "function"
    }
  ]
`)

	// CCVotingForTest.sol
	creationBytecode := testutils.HexToBytes(`0x608060405234801561001057600080fd5b5061043d806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c806305f63c8a1461003b578063ee00319414610071575b600080fd5b6100556004803603810190610050919061026b565b61008d565b6040516100689796959493929190610324565b60405180910390f35b61008b60048036038101906100869190610294565b610140565b005b600080600080600080600080600189815481106100d3577f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b906000526020600020906007020190508060000160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16816001015482600201548360030154846004015485600501548660060154975097509750975097509750975050919395979092949650565b60016040518060e001604052803373ffffffffffffffffffffffffffffffffffffffff168152602001868152602001858152602001848152602001838152602001600081526020016000815250908060018154018082558091505060019003906000526020600020906007020160009091909190915060008201518160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506020820151816001015560408201518160020155606082015181600301556080820151816004015560a0820151816005015560c08201518160060155505050505050565b600081359050610250816103d9565b92915050565b600081359050610265816103f0565b92915050565b60006020828403121561027d57600080fd5b600061028b84828501610256565b91505092915050565b600080600080608085870312156102aa57600080fd5b60006102b887828801610256565b94505060206102c987828801610241565b93505060406102da87828801610241565b92505060606102eb87828801610241565b91505092959194509250565b61030081610393565b82525050565b61030f816103a5565b82525050565b61031e816103cf565b82525050565b600060e082019050610339600083018a6102f7565b6103466020830189610315565b6103536040830188610306565b6103606060830187610306565b61036d6080830186610306565b61037a60a0830185610315565b61038760c0830184610315565b98975050505050505050565b600061039e826103af565b9050919050565b6000819050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b6103e2816103a5565b81146103ed57600080fd5b50565b6103f9816103cf565b811461040457600080fd5b5056fea2646970667358221220680049b6bd5fa6da33ba4c0a4124e7ff9dfaeb0636082ed0e87e3676fbdec58c64736f6c63430008040033`)

	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	tx, _, contractAddr := _app.DeployContractInBlock(key, creationBytecode)
	_app.EnsureTxSuccess(tx.Hash())

	addOpInput1 := _abi.MustPack("addOperator",
		big.NewInt(02),
		toBytes32("pubkeyX1"),
		toBytes32("12.34.56.78:9000"),
		toBytes32("intro1"),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOpInput1)
	_app.EnsureTxSuccess(tx.Hash())

	addOpInput2 := _abi.MustPack("addOperator",
		big.NewInt(03),
		toBytes32("pubkeyX2"),
		toBytes32("23.45.67.89:1000"),
		toBytes32("intro2"),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOpInput2)
	_app.EnsureTxSuccess(tx.Hash())

	// read data from Go
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)

	accInfo := ctx.GetAccount(contractAddr)
	seq := accInfo.Sequence()
	operatorsSlot := string(uint256.NewInt(1).PaddedBytes(32))
	operatorsLen := ctx.GetStorageAt(seq, operatorsSlot)
	require.Equal(t, int64(2), big.NewInt(0).SetBytes(operatorsLen).Int64())

	operatorsLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256([]byte(operatorsSlot)))
	// operator1
	op1 := ctx.GetStorageAt(seq, string(operatorsLoc.PaddedBytes(32)))
	require.Equal(t, addr.Bytes(), bytes.TrimLeft(op1, string([]byte{0})))
	prefix1 := ctx.GetStorageAt(seq, string(operatorsLoc.AddUint64(operatorsLoc, 1).PaddedBytes(32)))
	require.Equal(t, int64(2), big.NewInt(0).SetBytes(prefix1).Int64())
	x1 := ctx.GetStorageAt(seq, string(operatorsLoc.AddUint64(operatorsLoc, 1).PaddedBytes(32)))
	require.Equal(t, "pubkeyX1", strings.TrimRight(string(x1), string([]byte{0})))
	url1 := ctx.GetStorageAt(seq, string(operatorsLoc.AddUint64(operatorsLoc, 1).PaddedBytes(32)))
	require.Equal(t, "12.34.56.78:9000", strings.TrimRight(string(url1), string([]byte{0})))
	intro1 := ctx.GetStorageAt(seq, string(operatorsLoc.AddUint64(operatorsLoc, 1).PaddedBytes(32)))
	require.Equal(t, "intro1", strings.TrimRight(string(intro1), string([]byte{0})))
	votes1 := ctx.GetStorageAt(seq, string(operatorsLoc.AddUint64(operatorsLoc, 1).PaddedBytes(32)))
	require.Equal(t, int64(0), big.NewInt(0).SetBytes(votes1).Int64())
	// operator2
	op2 := ctx.GetStorageAt(seq, string(operatorsLoc.AddUint64(operatorsLoc, 2).PaddedBytes(32)))
	require.Equal(t, addr.Bytes(), bytes.TrimLeft(op2, string([]byte{0})))
	prefix2 := ctx.GetStorageAt(seq, string(operatorsLoc.AddUint64(operatorsLoc, 1).PaddedBytes(32)))
	require.Equal(t, int64(3), big.NewInt(0).SetBytes(prefix2).Int64())
	x2 := ctx.GetStorageAt(seq, string(operatorsLoc.AddUint64(operatorsLoc, 1).PaddedBytes(32)))
	require.Equal(t, "pubkeyX2", strings.TrimRight(string(x2), string([]byte{0})))
	url2 := ctx.GetStorageAt(seq, string(operatorsLoc.AddUint64(operatorsLoc, 1).PaddedBytes(32)))
	require.Equal(t, "23.45.67.89:1000", strings.TrimRight(string(url2), string([]byte{0})))
	intro2 := ctx.GetStorageAt(seq, string(operatorsLoc.AddUint64(operatorsLoc, 1).PaddedBytes(32)))
	require.Equal(t, "intro2", strings.TrimRight(string(intro2), string([]byte{0})))
	votes2 := ctx.GetStorageAt(seq, string(operatorsLoc.AddUint64(operatorsLoc, 1).PaddedBytes(32)))
	require.Equal(t, int64(0), big.NewInt(0).SetBytes(votes2).Int64())

	//operators[1].votes = 123
	ctx2 := _app.GetRunTxContext()
	ctx.SetStorageAt(seq, string(operatorsLoc.PaddedBytes(32)), uint256.NewInt(123).PaddedBytes(32))
	ctx2.Close(true)
	votes2 = ctx.GetStorageAt(seq, string(operatorsLoc.PaddedBytes(32)))
	require.Equal(t, int64(123), big.NewInt(0).SetBytes(votes2).Int64())
}

func toBytes32(s string) [32]byte {
	out := [32]byte{}
	copy(out[:], s)
	return out
}

/*
//SPDX-License-Identifier: Unlicense
pragma solidity ^0.8.0;

contract CCVoting2ForTest {

	struct OperatorOrMonitorInfo {
		address addr;              // address
		uint    pubkeyPrefix;      // 0x02 or 0x03 (TODO: change type to uint8)
		bytes32 pubkeyX;           // x
		bytes32 rpcUrl;            // ip:port (not used by monitors)
		bytes32 intro;             // introduction
		uint    totalStakedAmt;    // in BCH
		uint    selfStakedAmt;     // in BCH
		uint    inOfficeStartTime; // 0 means not in office, this field is set from Golang
	}

	struct StakeInfo {
		address staker;
		address monitor;
		address operator;
		uint32  stakedTime;
		uint    stakedAmt;
	}

	uint constant MONITOR_INIT_STAKE = 100_000 ether;
	uint constant OPERATOR_INIT_STAKE = 10_000 ether;
	uint constant MONITOR_MIN_STAKE_PERIOD = 200 days;
	uint constant OPERATOR_MIN_STAKE_PERIOD = 100 days;

	// read by Golang
	OperatorOrMonitorInfo[] monitors;
	OperatorOrMonitorInfo[] operators;

	mapping(address => uint) monitorIdxByAddr;
	mapping(address => uint) operatorIdxByAddr;

	uint lastStakeId;
	mapping(uint => StakeInfo) stakeById;
	mapping(address => uint[]) stakeIdsByAddr;

	function addMonitor(uint pubkeyPrefix,
		                bytes32 pubkeyX,
		                bytes32 rpcUrl,
		                bytes32 intro,
		                uint totalStakedAmt,
		                uint selfStakedAmt) public {
		monitors.push(OperatorOrMonitorInfo(msg.sender,
			pubkeyPrefix, pubkeyX, rpcUrl, intro, totalStakedAmt, selfStakedAmt, 0));
	}

	function addOperator(uint pubkeyPrefix,
		                 bytes32 pubkeyX,
		                 bytes32 rpcUrl,
		                 bytes32 intro,
		                 uint totalStakedAmt,
		                 uint selfStakedAmt) public {
		operators.push(OperatorOrMonitorInfo(msg.sender,
			pubkeyPrefix, pubkeyX, rpcUrl, intro, totalStakedAmt, selfStakedAmt, 0));
	}

}
*/
type OperatorOrMonitorInfo struct {
	addr              gethcmn.Address
	pubkey            []byte // 33 bytes
	rpcUrl            []byte // 32 bytes
	intro             []byte // 32 bytes
	totalStakedAmt    *uint256.Int
	selfStakedAmt     *uint256.Int
	inOfficeStartTime *uint256.Int
}

func readOperatorOrMonitorArr(ctx *types.Context, seq uint64, slot uint64) (result []OperatorOrMonitorInfo) {
	arrSlot := uint256.NewInt(slot).PaddedBytes(32)
	arrLen := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(arrSlot)))
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))

	for i := uint64(0); i < arrLen.ToBig().Uint64(); i++ {
		result = append(result, readOperatorOrMonitorInfo(ctx, seq, arrLoc))
	}
	return
}

func readOperatorOrMonitorInfo(ctx *types.Context, seq uint64, loc *uint256.Int) OperatorOrMonitorInfo {
	addr := gethcmn.BytesToAddress(ctx.GetStorageAt(seq, string(loc.PaddedBytes(32))))
	pubkeyPrefix := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	pubkeyX := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	rpcUrl := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	intro := ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32)))
	selfStakedAmt := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))))
	totalStakedAmt := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(loc.AddUint64(loc, 1).PaddedBytes(32))))
	key := loc.AddUint64(loc, 1).PaddedBytes(32)
	println("inOfficeStartTimeKey:", hex.EncodeToString(key))
	inOfficeStartTime := uint256.NewInt(0).SetBytes(ctx.GetStorageAt(seq, string(key)))
	println("inOfficeStartTime:", inOfficeStartTime.Uint64())
	loc.AddUint64(loc, 1)
	return OperatorOrMonitorInfo{
		addr:              addr,
		pubkey:            append(pubkeyPrefix[31:], pubkeyX...),
		rpcUrl:            rpcUrl[:],
		intro:             intro[:],
		totalStakedAmt:    totalStakedAmt,
		selfStakedAmt:     selfStakedAmt,
		inOfficeStartTime: inOfficeStartTime,
	}
}

func writeOperatorOrMonitorInOfficeStartTime(ctx *types.Context, seq uint64, slot uint64, idx uint64, val uint64) {
	arrSlot := uint256.NewInt(slot).PaddedBytes(32)
	arrLoc := uint256.NewInt(0).SetBytes(crypto.Keccak256(arrSlot))
	itemLoc := uint256.NewInt(0).AddUint64(arrLoc, idx*8)
	fieldLoc := uint256.NewInt(0).AddUint64(itemLoc, 7)
	println("kkk:", hex.EncodeToString(fieldLoc.PaddedBytes(32)), val)
	ctx.SetStorageAt(seq, string(fieldLoc.PaddedBytes(32)),
		uint256.NewInt(val).PaddedBytes(32))
}

func TestCCVoting2(t *testing.T) {
	_abi := ethutils.MustParseABI(`
[
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "pubkeyPrefix",
          "type": "uint256"
        },
        {
          "internalType": "bytes32",
          "name": "pubkeyX",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "rpcUrl",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "intro",
          "type": "bytes32"
        },
        {
          "internalType": "uint256",
          "name": "totalStakedAmt",
          "type": "uint256"
        },
        {
          "internalType": "uint256",
          "name": "selfStakedAmt",
          "type": "uint256"
        }
      ],
      "name": "addMonitor",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "pubkeyPrefix",
          "type": "uint256"
        },
        {
          "internalType": "bytes32",
          "name": "pubkeyX",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "rpcUrl",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "intro",
          "type": "bytes32"
        },
        {
          "internalType": "uint256",
          "name": "totalStakedAmt",
          "type": "uint256"
        },
        {
          "internalType": "uint256",
          "name": "selfStakedAmt",
          "type": "uint256"
        }
      ],
      "name": "addOperator",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    }
  ]
`)

	// CCVoting2ForTest.sol
	creationBytecode := testutils.HexToBytes(`0x608060405234801561001057600080fd5b506103c4806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c8063692ea8021461003b5780639717c20014610057575b600080fd5b610055600480360381019061005091906102c3565b610073565b005b610071600480360381019061006c91906102c3565b610186565b005b60016040518061010001604052803373ffffffffffffffffffffffffffffffffffffffff1681526020018881526020018781526020018681526020018581526020018481526020018381526020016000815250908060018154018082558091505060019003906000526020600020906008020160009091909190915060008201518160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506020820151816001015560408201518160020155606082015181600301556080820151816004015560a0820151816005015560c0820151816006015560e082015181600701555050505050505050565b60006040518061010001604052803373ffffffffffffffffffffffffffffffffffffffff1681526020018881526020018781526020018681526020018581526020018481526020018381526020016000815250908060018154018082558091505060019003906000526020600020906008020160009091909190915060008201518160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506020820151816001015560408201518160020155606082015181600301556080820151816004015560a0820151816005015560c0820151816006015560e082015181600701555050505050505050565b6000813590506102a881610360565b92915050565b6000813590506102bd81610377565b92915050565b60008060008060008060c087890312156102dc57600080fd5b60006102ea89828a016102ae565b96505060206102fb89828a01610299565b955050604061030c89828a01610299565b945050606061031d89828a01610299565b935050608061032e89828a016102ae565b92505060a061033f89828a016102ae565b9150509295509295509295565b6000819050919050565b6000819050919050565b6103698161034c565b811461037457600080fd5b50565b61038081610356565b811461038b57600080fd5b5056fea2646970667358221220e7fb25df78b19bc021c1890ed0410a43b670cfacc9b6486f7ff125722f0d76e664736f6c63430008040033`)

	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	tx, _, contractAddr := _app.DeployContractInBlock(key, creationBytecode)
	_app.EnsureTxSuccess(tx.Hash())

	addMonitor1 := _abi.MustPack("addMonitor",
		big.NewInt(02),
		toBytes32("pubkeyX_m1"),
		toBytes32("12.34.56.78:9001"),
		toBytes32("monitor#1"),
		big.NewInt(1001),
		big.NewInt(1002),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addMonitor1)
	_app.EnsureTxSuccess(tx.Hash())

	addMonitor2 := _abi.MustPack("addMonitor",
		big.NewInt(03),
		toBytes32("pubkeyX_m2"),
		toBytes32("12.34.56.78:9002"),
		toBytes32("monitor#2"),
		big.NewInt(2001),
		big.NewInt(2002),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addMonitor2)
	_app.EnsureTxSuccess(tx.Hash())

	addOperator1 := _abi.MustPack("addOperator",
		big.NewInt(02),
		toBytes32("pubkeyX_o1"),
		toBytes32("12.34.56.78:9011"),
		toBytes32("operator#1"),
		big.NewInt(1011),
		big.NewInt(1012),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOperator1)
	_app.EnsureTxSuccess(tx.Hash())

	addOperator2 := _abi.MustPack("addOperator",
		big.NewInt(03),
		toBytes32("pubkeyX_o2"),
		toBytes32("12.34.56.78:9012"),
		toBytes32("operator#2"),
		big.NewInt(2011),
		big.NewInt(2012),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOperator2)
	_app.EnsureTxSuccess(tx.Hash())

	addOperator3 := _abi.MustPack("addOperator",
		big.NewInt(02),
		toBytes32("pubkeyX_o3"),
		toBytes32("12.34.56.78:9013"),
		toBytes32("operator#3"),
		big.NewInt(3011),
		big.NewInt(3012),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOperator3)
	_app.EnsureTxSuccess(tx.Hash())

	// read data from Go
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)

	accInfo := ctx.GetAccount(contractAddr)
	seq := accInfo.Sequence()

	monitors := readOperatorOrMonitorArr(ctx, seq, 0)
	require.Len(t, monitors, 2)
	require.Equal(t, addr, monitors[0].addr)
	require.Equal(t, "027075626b6579585f6d3100000000000000000000000000000000000000000000",
		hex.EncodeToString(monitors[0].pubkey))
	require.Equal(t, "12.34.56.78:9001",
		strings.TrimRight(string(monitors[0].rpcUrl), string([]byte{0})))
	require.Equal(t, "monitor#1",
		strings.TrimRight(string(monitors[0].intro), string([]byte{0})))
	require.Equal(t, uint64(1001), monitors[0].selfStakedAmt.Uint64())
	require.Equal(t, uint64(1002), monitors[0].totalStakedAmt.Uint64())
	require.Equal(t, uint64(0), monitors[0].inOfficeStartTime.Uint64())

	require.Equal(t, addr, monitors[1].addr)
	require.Equal(t, "037075626b6579585f6d3200000000000000000000000000000000000000000000",
		hex.EncodeToString(monitors[1].pubkey))
	require.Equal(t, "12.34.56.78:9002",
		strings.TrimRight(string(monitors[1].rpcUrl), string([]byte{0})))
	require.Equal(t, "monitor#2",
		strings.TrimRight(string(monitors[1].intro), string([]byte{0})))
	require.Equal(t, uint64(2001), monitors[1].selfStakedAmt.Uint64())
	require.Equal(t, uint64(2002), monitors[1].totalStakedAmt.Uint64())
	require.Equal(t, uint64(0), monitors[1].inOfficeStartTime.Uint64())

	operators := readOperatorOrMonitorArr(ctx, seq, 1)
	require.Len(t, operators, 3)
	require.Equal(t, addr, operators[0].addr)
	require.Equal(t, "027075626b6579585f6f3100000000000000000000000000000000000000000000",
		hex.EncodeToString(operators[0].pubkey))
	require.Equal(t, "12.34.56.78:9011",
		strings.TrimRight(string(operators[0].rpcUrl), string([]byte{0})))
	require.Equal(t, "operator#1",
		strings.TrimRight(string(operators[0].intro), string([]byte{0})))
	require.Equal(t, uint64(1011), operators[0].selfStakedAmt.Uint64())
	require.Equal(t, uint64(1012), operators[0].totalStakedAmt.Uint64())
	require.Equal(t, uint64(0), operators[0].inOfficeStartTime.Uint64())

	require.Equal(t, addr, operators[1].addr)
	require.Equal(t, "037075626b6579585f6f3200000000000000000000000000000000000000000000",
		hex.EncodeToString(operators[1].pubkey))
	require.Equal(t, "12.34.56.78:9012",
		strings.TrimRight(string(operators[1].rpcUrl), string([]byte{0})))
	require.Equal(t, "operator#2",
		strings.TrimRight(string(operators[1].intro), string([]byte{0})))
	require.Equal(t, uint64(2011), operators[1].selfStakedAmt.Uint64())
	require.Equal(t, uint64(2012), operators[1].totalStakedAmt.Uint64())
	require.Equal(t, uint64(0), operators[1].inOfficeStartTime.Uint64())

	require.Equal(t, addr, operators[2].addr)
	require.Equal(t, "027075626b6579585f6f3300000000000000000000000000000000000000000000",
		hex.EncodeToString(operators[2].pubkey))
	require.Equal(t, "12.34.56.78:9013",
		strings.TrimRight(string(operators[2].rpcUrl), string([]byte{0})))
	require.Equal(t, "operator#3",
		strings.TrimRight(string(operators[2].intro), string([]byte{0})))
	require.Equal(t, uint64(3011), operators[2].selfStakedAmt.Uint64())
	require.Equal(t, uint64(3012), operators[2].totalStakedAmt.Uint64())
	require.Equal(t, uint64(0), operators[2].inOfficeStartTime.Uint64())

	//operators[1].votes = 123
	//ctx2 := _app.GetRunTxContext()
	writeOperatorOrMonitorInOfficeStartTime(ctx, seq, 1, 1, 123)
	//ctx2.Close(true)
	operators = readOperatorOrMonitorArr(ctx, seq, 1)
	require.Equal(t, uint64(0), operators[0].inOfficeStartTime.Uint64())
	require.Equal(t, uint64(0), operators[1].inOfficeStartTime.Uint64())
	require.Equal(t, uint64(0), operators[2].inOfficeStartTime.Uint64())
}
