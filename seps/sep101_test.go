package seps_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
)

var _sep101ABI = ethutils.MustParseABI(`
[
{
  "inputs": [
	{
	  "internalType": "bytes",
	  "name": "key",
	  "type": "bytes"
	},
	{
	  "internalType": "bytes",
	  "name": "value",
	  "type": "bytes"
	}
  ],
  "name": "set",
  "outputs": [],
  "stateMutability": "nonpayable",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "bytes",
	  "name": "key",
	  "type": "bytes"
	}
  ],
  "name": "get",
  "outputs": [
	{
	  "internalType": "bytes",
	  "name": "",
	  "type": "bytes"
	}
  ],
  "stateMutability": "nonpayable",
  "type": "function"
}
]
`)

// see testdata/sol/contracts/seps/SEP101Proxy_DELEGATECALL.sol
var _sep101ProxyCreationBytecode = testutils.HexToBytes(`
6080604052348015600f57600080fd5b50606780601d6000396000f3fe608060
40526000612712905060405136600082376000803683855af43d806000843e81
60008114602d578184f35b8184fdfea26469706673582212207b1492f6a8ff64
a3c095ea643170fd74565ee96f5e8a8bfea90b5863a23d4f9c64736f6c634300
08000033
`)

func deploySEP101Proxy(t *testing.T, _app *testutils.TestApp, privKey string) gethcmn.Address {
	_, _, contractAddr := _app.DeployContractInBlock(privKey, _sep101ProxyCreationBytecode)
	require.NotEmpty(t, _app.GetCode(contractAddr))
	return contractAddr
}

func TestSEP101(t *testing.T) {
	privKey, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(privKey)
	defer _app.Destroy()

	// deploy proxy
	contractAddr := deploySEP101Proxy(t, _app, privKey)

	key := []byte{0xAB, 0xCD}
	val := bytes.Repeat([]byte{0x12, 0x34}, 500)

	// call set()
	data := _sep101ABI.MustPack("set", key, val)
	tx, _ := _app.MakeAndExecTxInBlock(privKey, contractAddr, 0, data)
	_app.EnsureTxSuccess(tx.Hash())

	// call get()
	data = _sep101ABI.MustPack("get", key)
	statusCode, statusStr, output := _app.Call(addr, contractAddr, data)
	require.Equal(t, 0, statusCode)
	require.Equal(t, "success", statusStr)
	require.Equal(t, []interface{}{val}, _sep101ABI.MustUnpack("get", output))

	// read val through getStorageAt()
	sKey := sha256.Sum256(key)
	sVal := _app.GetStorageAt(contractAddr, sKey[:])
	require.Equal(t, val, sVal)

	// get non-existing key
	data = _sep101ABI.MustPack("get", []byte{9, 9, 9})
	statusCode, statusStr, output = _app.Call(addr, contractAddr, data)
	require.Equal(t, "success", statusStr)
	require.Equal(t, 0, statusCode)
	require.Equal(t, []interface{}{[]byte{}}, _sep101ABI.MustUnpack("get", output))
}

func TestSEP101_setZeroLenKey(t *testing.T) {
	privKey, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(privKey)
	defer _app.Destroy()

	// deploy proxy
	contractAddr := deploySEP101Proxy(t, _app, privKey)

	// set() with zero-len key
	data := _sep101ABI.MustPack("set", []byte{}, []byte{1, 2, 3})
	tx, _ := _app.MakeAndExecTxInBlock(privKey, contractAddr, 0, data)
	_app.EnsureTxFailed(tx.Hash(), "revert")
}

func TestSEP101_setKeyTooLong(t *testing.T) {
	privKey, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(privKey)
	defer _app.Destroy()

	// deploy proxy
	contractAddr := deploySEP101Proxy(t, _app, privKey)

	// set() with looooong key
	data := _sep101ABI.MustPack("set", bytes.Repeat([]byte{39}, 257), []byte{1, 2, 3})
	tx, _ := _app.MakeAndExecTxInBlock(privKey, contractAddr, 0, data)
	_app.EnsureTxFailed(tx.Hash(), "revert")
}

func TestSEP101_setValTooLong(t *testing.T) {
	privKey, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(privKey)
	defer _app.Destroy()

	// deploy proxy
	contractAddr := deploySEP101Proxy(t, _app, privKey)

	// set() with looooong val
	data := _sep101ABI.MustPack("set", []byte{1, 2, 3}, bytes.Repeat([]byte{39}, 24*1024+1))
	tx, _ := _app.MakeAndExecTxInBlock(privKey, contractAddr, 0, data)
	_app.EnsureTxFailed(tx.Hash(), "revert")
}

func TestSEP101_archiveMode(t *testing.T) {
	privKey, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestAppInArchiveMode(privKey)
	defer _app.Destroy()

	// deploy proxy
	contractAddr := deploySEP101Proxy(t, _app, privKey)

	key := []byte{0xAB, 0xCD}
	for i := 0; i < 3; i++ {
		val := []byte{0xEF, byte(i * 100)}
		data := _sep101ABI.MustPack("set", key, val)
		tx, h := _app.MakeAndExecTxInBlock(privKey, contractAddr, 0, data)
		_app.EnsureTxSuccess(tx.Hash())
		require.Equal(t, int64(3+i*2), h) // 3, 5, 7
	}

	data := _sep101ABI.MustPack("get", key)
	require.Equal(t, []byte{0xef, 200}, sep101Get(_app, addr, contractAddr, data, -1))
	require.Equal(t, []byte{0xef, 200}, sep101Get(_app, addr, contractAddr, data, 7))
	require.Equal(t, []byte{0xef, 100}, sep101Get(_app, addr, contractAddr, data, 6))
	require.Equal(t, []byte{0xef, 100}, sep101Get(_app, addr, contractAddr, data, 5))
	require.Equal(t, []byte{0xef, 0}, sep101Get(_app, addr, contractAddr, data, 4))
	require.Equal(t, []byte{0xef, 0}, sep101Get(_app, addr, contractAddr, data, 3))
	require.Equal(t, []byte{}, sep101Get(_app, addr, contractAddr, data, 2))
	require.Equal(t, []byte{}, sep101Get(_app, addr, contractAddr, data, 1))
}

func sep101Get(_app *testutils.TestApp, sender, contractAddr gethcmn.Address, data []byte, height int64) []byte {
	statusCode, statusStr, retData := _app.CallAtHeight(sender, contractAddr, data, height)
	if statusCode != 0 {
		panic(fmt.Errorf("statusCode=%d", statusCode))
	}
	if statusStr != "success" {
		panic(fmt.Errorf("statusStr=%s", statusStr))
	}
	unpacked := _sep101ABI.MustUnpack("get", retData)
	if n := len(unpacked); n != 1 {
		panic(fmt.Errorf("len(unpacked)=%d", n))
	}
	return (unpacked[0]).([]byte)
}
