package app_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/internal/ethutils"
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
	_app := testutils.CreateTestApp0(time.Now(),
		ed25519.GenPrivKey().PubKey(), bigutils.NewU256(1000000000), key)
	defer _app.Destroy()

	// see testdata/basic/contracts/EventEmitter.sol
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
	// see testdata/basic/contracts/TestAdd.sol
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

	// testdata/basic/contracts/BlockHash2.sol
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
