package app

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
)

func TestDeployContract(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key)
	defer DestroyTestApp(_app)

	// testdata/test05_counter
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
	tx := gethtypes.NewContractCreation(0, big.NewInt(0), 100000, big.NewInt(1), creationBytecode)
	tx = ethutils.MustSignTx(tx, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key))

	testutils.ExecTxInBlock(_app, 1, tx)
	contractAddr := gethcrypto.CreateAddress(addr, tx.Nonce())
	code := getCode(_app, contractAddr)
	require.Equal(t, deployedBytecode, code)
}

func TestEmitLogs(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := CreateTestApp0(bigutils.NewU256(1000000000), key)
	defer DestroyTestApp(_app)

	// testdata/test06_events
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
	tx1 := gethtypes.NewContractCreation(0,
		big.NewInt(0), 10000000, big.NewInt(1), creationBytecode)
	tx1 = ethutils.MustSignTx(tx1, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key))
	testutils.ExecTxInBlock(_app, 1, tx1)

	contractAddr := gethcrypto.CreateAddress(addr, tx1.Nonce())
	code := getCode(_app, contractAddr)
	require.True(t, len(code) > 0)

	blk1 := getBlock(_app, 1)
	require.Equal(t, int64(1), blk1.Number)
	require.Len(t, blk1.Transactions, 1)
	txInBlk2 := getTx(_app, blk1.Transactions[0])
	require.Equal(t, gethtypes.ReceiptStatusSuccessful, txInBlk2.Status)
	require.Equal(t, tx1.Hash(), common.Hash(txInBlk2.Hash))

	// call emitEvent1()
	tx2 := gethtypes.NewTransaction(1, contractAddr,
		big.NewInt(0), 10000000, big.NewInt(1), testutils.HexToBytes("990ee412"))
	tx2 = ethutils.MustSignTx(tx2, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key))
	testutils.ExecTxInBlock(_app, 3, tx2)

	time.Sleep(100 * time.Millisecond)
	blk3 := getBlock(_app, 3)
	require.Equal(t, int64(3), blk3.Number)
	require.Len(t, blk3.Transactions, 1)
	txInBlk3 := getTx(_app, blk3.Transactions[0])
	require.Equal(t, gethtypes.ReceiptStatusSuccessful, txInBlk3.Status)
	require.Equal(t, tx2.Hash(), common.Hash(txInBlk3.Hash))
	require.Len(t, txInBlk3.Logs, 1)
	require.Len(t, txInBlk3.Logs[0].Topics, 2)
	require.Equal(t, "d1c6b99eac4e6a0f44c67915eb5195ecb58425668b0c7a46f58908541b5b2899",
		hex.EncodeToString(txInBlk3.Logs[0].Topics[0][:]))
	require.Equal(t, "000000000000000000000000"+hex.EncodeToString(addr[:]),
		hex.EncodeToString(txInBlk3.Logs[0].Topics[1][:]))

	// call emitEvent2()
	tx3 := gethtypes.NewTransaction(2, contractAddr,
		big.NewInt(0), 10000000, big.NewInt(1),
		testutils.HexToBytes("0xfb584c39000000000000000000000000000000000000000000000000000000000000007b"))
	tx3 = ethutils.MustSignTx(tx3, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key))
	testutils.ExecTxInBlock(_app, 5, tx3)

	time.Sleep(100 * time.Millisecond)
	blk5 := getBlock(_app, 5)
	require.Equal(t, int64(5), blk5.Number)
	require.Len(t, blk5.Transactions, 1)
	txInBlk5 := getTx(_app, blk5.Transactions[0])
	require.Equal(t, gethtypes.ReceiptStatusSuccessful, txInBlk5.Status)
	require.Equal(t, tx3.Hash(), common.Hash(txInBlk5.Hash))
	require.Len(t, txInBlk5.Logs, 1)
	require.Len(t, txInBlk5.Logs[0].Topics, 2)
	require.Equal(t, "7a2c2ad471d70e0a88640e6c3f4f5e975bcbccea7740c25631d0b74bb2c1cef4",
		hex.EncodeToString(txInBlk5.Logs[0].Topics[0][:]))
	require.Equal(t, "000000000000000000000000"+hex.EncodeToString(addr[:]),
		hex.EncodeToString(txInBlk5.Logs[0].Topics[1][:]))
	require.Equal(t, "000000000000000000000000000000000000000000000000000000000000007b",
		hex.EncodeToString(txInBlk5.Logs[0].Data))
}

func TestChainID(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key)
	defer DestroyTestApp(_app)

	require.Equal(t, "0x1", _app.ChainID().String())

	// testdata/test07_eip1344
	creationBytecode := testutils.HexToBytes(`
608060405234801561001057600080fd5b5060b58061001f6000396000f3fe60
80604052348015600f57600080fd5b506004361060285760003560e01c806356
4b81ef14602d575b600080fd5b60336047565b604051603e9190605c565b6040
5180910390f35b600046905090565b6056816075565b82525050565b60006020
82019050606f6000830184604f565b92915050565b600081905091905056fea2
64697066735822122071af38cd4ec3657373c5944f6d44becf841a91b5a85545
7dfdabc41dd2e3b50064736f6c63430008000033
`)

	tx1 := gethtypes.NewContractCreation(0, big.NewInt(0), 100000, big.NewInt(1), creationBytecode)
	tx1 = ethutils.MustSignTx(tx1, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key))

	testutils.ExecTxInBlock(_app, 1, tx1)
	contractAddr := gethcrypto.CreateAddress(addr, tx1.Nonce())
	code := getCode(_app, contractAddr)
	require.True(t, len(code) > 0)

	tx2 := gethtypes.NewTransaction(1, contractAddr, big.NewInt(0), 100000, big.NewInt(1),
		testutils.HexToBytes("564b81ef"))

	_, _, output := call(_app, addr, tx2)
	require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000001",
		hex.EncodeToString(output))
}

func TestRevert(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := CreateTestApp(key)
	defer DestroyTestApp(_app)

	// testdata/test08_errors
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

	tx1 := gethtypes.NewContractCreation(0, big.NewInt(0), 1000000, big.NewInt(1), creationBytecode)
	tx1 = ethutils.MustSignTx(tx1, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key))

	testutils.ExecTxInBlock(_app, 1, tx1)
	contractAddr := gethcrypto.CreateAddress(addr, tx1.Nonce())
	code := getCode(_app, contractAddr)
	require.True(t, len(code) > 0)

	// call setN_revert()
	tx2 := gethtypes.NewTransaction(1, contractAddr, big.NewInt(0), 1000000, big.NewInt(1),
		testutils.HexToBytes("0xe0ada09a0000000000000000000000000000000000000000000000000000000000000064"))
	tx2 = ethutils.MustSignTx(tx2, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key))
	testutils.ExecTxInBlock(_app, 3, tx2)

	time.Sleep(100 * time.Millisecond)
	blk3 := getBlock(_app, 3)
	require.Equal(t, int64(3), blk3.Number)
	require.Len(t, blk3.Transactions, 1)
	txInBlk3 := getTx(_app, blk3.Transactions[0])
	require.Equal(t, gethtypes.ReceiptStatusFailed, txInBlk3.Status)
	require.Equal(t, "revert", txInBlk3.StatusStr)

	statusCode, statusStr, retData := call(_app, contractAddr, tx2)
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
	_app := CreateTestApp(key)
	defer DestroyTestApp(_app)

	// testdata/test08_errors
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

	tx1 := gethtypes.NewContractCreation(0, big.NewInt(0), 1000000, big.NewInt(1), creationBytecode)
	tx1 = ethutils.MustSignTx(tx1, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key))

	testutils.ExecTxInBlock(_app, 1, tx1)
	contractAddr := gethcrypto.CreateAddress(addr, tx1.Nonce())
	code := getCode(_app, contractAddr)
	require.True(t, len(code) > 0)

	// call setN_invalidOpcode()
	tx2 := gethtypes.NewTransaction(1, contractAddr, big.NewInt(0), 1000000, big.NewInt(1),
		testutils.HexToBytes("0x12f28d510000000000000000000000000000000000000000000000000000000000000064"))
	tx2 = ethutils.MustSignTx(tx2, _app.chainId.ToBig(), ethutils.MustHexToPrivKey(key))
	testutils.ExecTxInBlock(_app, 3, tx2)

	time.Sleep(100 * time.Millisecond)
	blk3 := getBlock(_app, 3)
	require.Equal(t, int64(3), blk3.Number)
	require.Len(t, blk3.Transactions, 1)
	txInBlk3 := getTx(_app, blk3.Transactions[0])
	require.Equal(t, gethtypes.ReceiptStatusFailed, txInBlk3.Status)
	require.Equal(t, "invalid-instruction", txInBlk3.StatusStr)

	statusCode, statusStr, _ := call(_app, contractAddr, tx2)
	require.Equal(t, 4, statusCode)
	require.Equal(t, "invalid-instruction", statusStr)
}
