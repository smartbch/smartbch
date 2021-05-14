package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/coinexchain/randsrc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"github.com/smartbch/moeingads/indextree"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/param"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

type RocksDB = indextree.RocksDB

const (
	adsDir   = "./testdbdata"
	modbDir  = "./modbdata"
	blockDir = "./blkdata"
)

var num1e18 = uint256.NewInt().SetUint64(1_000_000_000_000_000_000)
var initBalance = uint256.NewInt().Mul(num1e18, num1e18)

type Block struct {
	AppHash []byte
	TxList  [][]byte
}

type BlockDB struct {
	rdb    *RocksDB
	height uint32
}

func NewBlockDB(dir string) *BlockDB {
	db, err := indextree.NewRocksDB("rocksdb", dir)
	if err != nil {
		panic(err)
	}
	return &BlockDB{db, 1}
}

func (db *BlockDB) SaveBlock(blk *Block) {
	key := []byte("B1234")
	binary.LittleEndian.PutUint32(key[1:], db.height)
	bz, err := rlp.EncodeToBytes(blk)
	if err != nil {
		panic(err)
	}
	db.rdb.SetSync(key, bz)
	db.height++
}

func (db *BlockDB) LoadBlock(height uint32) *Block {
	key := []byte("B1234")
	binary.LittleEndian.PutUint32(key[1:], height)
	bz := db.rdb.Get(key)
	if len(bz) == 0 {
		return nil
	}
	var blk Block
	err := rlp.DecodeBytes(bz, &blk)
	if err != nil {
		panic(err)
	}
	return &blk
}

func ReplayBlocks(_app *testutils.TestApp, db *BlockDB, endHeight uint32) {
	for h := uint32(1); h <= endHeight; h++ {
		blk := db.LoadBlock(h)
		appHash := ExecTxsInOneBlock(_app, int64(h), blk.TxList)
		if !bytes.Equal(appHash[:], blk.AppHash[:]) {
			panic("Incorrect AppHash")
		}
	}
}

// ==================================================

func GenKeysToFile(fname string, count int) {
	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for i := 0; i < count; i++ {
		key, _ := crypto.GenerateKey()
		fmt.Fprintln(f, hex.EncodeToString(crypto.FromECDSA(key)))
	}
}

func ReadKeysFromFile(fname string) (res []string) {
	f, err := os.Open("test.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	res = make([]string, 0, 8192)
	for scanner.Scan() {
		txt := scanner.Text()
		_, err := crypto.HexToECDSA(txt)
		if err != nil {
			panic(err)
		}
		res = append(res, txt)
	}
	return
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

var creationBytecode = testutils.HexToBytes(`
	608060405234801561001057600080fd5b506102f1806100206000396000f3fe6080604052600436106100295760003560e01c8063381fd1901461002e578063d8a26e3a14610043575b600080fd5b61004161003c3660046101d4565b610079565b005b34801561004f57600080fd5b5061006361005e36600461020a565b6101bc565b604051610070919061026e565b60405180910390f35b604080516000815260208101918290526001600160a01b038416916123289134916100a49190610235565b600060405180830381858888f193505050503d80600081146100e2576040519150601f19603f3d011682016040523d82523d6000602084013e6100e7565b606091505b505050602081811c63ffffffff81811660009081529283905260408084205491851684529283902054849384901c91606085901c91608086901c9160a087901c9160029134916101379190610277565b6101419190610277565b61014b919061029b565b63ffffffff808616600090815260208190526040808220939093558482168152828120549186168152919091205460029134916101889190610277565b6101929190610277565b61019c919061029b565b63ffffffff90911660009081526020819052604090205550505050505050565b63ffffffff1660009081526020819052604090205490565b600080604083850312156101e6578182fd5b82356001600160a01b03811681146101fc578283fd5b946020939093013593505050565b60006020828403121561021b578081fd5b813563ffffffff8116811461022e578182fd5b9392505050565b60008251815b81811015610255576020818601810151858301520161023b565b818111156102635782828501525b509190910192915050565b90815260200190565b6000821982111561029657634e487b7160e01b81526011600452602481fd5b500190565b6000826102b657634e487b7160e01b81526012600452602481fd5b50049056fea2646970667358221220e66a2e809beccf7e9d31ac11ec547ae76cd13f0c56d68a51d2da6224c706fdd864736f6c63430008000033
`)

func GetDeployTxAndAddrList(_app *testutils.TestApp, privKeys []string, creationBytecode []byte) ([][]byte, []common.Address) {
	txList := make([][]byte, len(privKeys))
	contractAddrList := make([]common.Address, len(privKeys))
	for i := range privKeys {
		tx, addr := _app.MakeAndSignTx(privKeys[i], nil, 0, creationBytecode, 0)
		txList[i] = testutils.MustEncodeTx(tx)
		contractAddrList[i] = crypto.CreateAddress(addr, tx.Nonce())
	}
	return txList, contractAddrList
}

func ExecTxsInOneBlock(_app *testutils.TestApp, height int64, txs [][]byte) (appHash []byte) {
	_app.BeginBlock(abci.RequestBeginBlock{
		Header: tmproto.Header{
			Height:          height,
			Time:            time.Now(),
			ProposerAddress: _app.TestPubkey.Address(),
		},
	})
	for _, tx := range txs {
		_app.DeliverTx(abci.RequestDeliverTx{
			Tx: tx,
		})
	}
	_app.EndBlock(abci.RequestEndBlock{Height: height})
	responseCommit := _app.Commit()
	return responseCommit.Data
}

func GetTx(_app *testutils.TestApp, key string, contractAddr, toAddr common.Address, value int64, slots [6]uint32) []byte {
	param := big.NewInt(int64(slots[0]))
	param.Lsh(param, 32)
	param.Or(param, big.NewInt(int64(slots[1])))
	param.Lsh(param, 32)
	param.Or(param, big.NewInt(int64(slots[2])))
	param.Lsh(param, 32)
	param.Or(param, big.NewInt(int64(slots[3])))
	param.Lsh(param, 32)
	param.Or(param, big.NewInt(int64(slots[4])))
	param.Lsh(param, 32)
	param.Or(param, big.NewInt(int64(slots[5])))
	calldata := testAddABI.MustPack("run", toAddr, param)
	tx, _ := _app.MakeAndSignTx(key, &contractAddr, value, calldata, 2 /*gasprice*/)
	return testutils.MustEncodeTx(tx)
}

func GenInitStorageTxList(_app *testutils.TestApp, fromKeys []string, contractAddrs, toAddrs []common.Address, j int) [][]byte {
	if len(fromKeys) != len(contractAddrs) || len(toAddrs)%len(fromKeys) != 0 {
		panic("Invalid lengths")
	}
	res := make([][]byte, 0, len(toAddrs))
	for i, fromKey := range fromKeys {
		slots := [6]uint32{uint32(2 * j), 0, 0, uint32(2*j + 1), 0, 0}
		tx := GetTx(_app, fromKey, contractAddrs[i], toAddrs[i], 10000, slots)
		res = append(res, tx)
	}
	return res
}

func GenRandTxList(_app *testutils.TestApp, rs randsrc.RandSrc, txList [][]byte, fromKeys []string, contractAddrs, toAddrs []common.Address) {
	if len(fromKeys) != len(contractAddrs) || len(toAddrs)%len(fromKeys) != 0 {
		panic("Invalid lengths")
	}
	touchedFrom := make(map[int]struct{}, len(txList))
	for i := range txList {
		x := int(rs.GetUint32()) % len(fromKeys)
		for {
			if _, ok := touchedFrom[x]; !ok {
				touchedFrom[x] = struct{}{}
				break
			}
			x = int(rs.GetUint32()) % len(fromKeys)
		}
		y := int(rs.GetUint32()) % len(contractAddrs)
		z := int(rs.GetUint32()) % len(toAddrs)
		value := int64(rs.GetUint32()) % 2100_0000
		var slots [6]uint32
		for j := 0; j < len(slots); j++ {
			slots[j] = rs.GetUint32() % uint32(len(toAddrs))
		}
		txList[i] = GetTx(_app, fromKeys[x], contractAddrs[y], toAddrs[z], value, slots)
	}
}

func KeyToAddr(keyStr string) common.Address {
	key, err := crypto.HexToECDSA(keyStr)
	if err != nil {
		panic(err)
	}
	return crypto.PubkeyToAddress(key.PublicKey)
}

func CreateTestApp(testInitAmt *uint256.Int, keys []string) *testutils.TestApp {
	_ = os.RemoveAll(adsDir)
	_ = os.RemoveAll(modbDir)
	params := param.DefaultConfig()
	params.AppDataPath = adsDir
	params.ModbDataPath = modbDir
	params.UseLiteDB = true
	testValidatorPubKey := ed25519.GenPrivKey().PubKey()
	_app := app.NewApp(params, bigutils.NewU256(1), log.NewNopLogger())
	genesisData := app.GenesisData{
		Alloc: testutils.KeysToGenesisAlloc(testInitAmt, keys),
	}
	testValidator := &stakingtypes.Validator{}
	copy(testValidator.Address[:], testValidatorPubKey.Address().Bytes())
	copy(testValidator.Pubkey[:], testValidatorPubKey.Bytes())
	testValidator.VotingPower = 1
	genesisData.Validators = append(genesisData.Validators, testValidator)
	appStateBytes, _ := json.Marshal(genesisData)

	_app.InitChain(abci.RequestInitChain{AppStateBytes: appStateBytes})
	_app.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{
		ProposerAddress: testValidatorPubKey.Address(),
	}})
	_app.Commit()
	return &testutils.TestApp{App: _app, TestPubkey: testValidatorPubKey}
}

func RecordBlocks(db *BlockDB, rs randsrc.RandSrc, totalNum int, keys []string, fromSize, toSize int) {
	if toSize%fromSize != 0 {
		panic("Invalid sizes")
	}
	n := toSize / fromSize
	if len(keys) < fromSize+toSize {
		panic("not enough keys")
	}
	toAddrs := make([]common.Address, toSize)
	for i := range toAddrs {
		toAddrs[i] = KeyToAddr(keys[fromSize+i])
	}
	keys = keys[:fromSize]
	_app := CreateTestApp(initBalance, keys)
	txList, contractAddrs := GetDeployTxAndAddrList(_app, keys, creationBytecode)
	blk := &Block{}
	for len(txList) != 0 {
		length := 512
		if len(txList) < length {
			length = len(txList)
		}
		blk.TxList = txList[:length]
		txList = txList[length:]
		blk.AppHash = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
		db.SaveBlock(blk)
	}
	for j := 0; j < n; j++ {
		start, end := fromSize*j, fromSize*(j+1)
		blk.TxList = GenInitStorageTxList(_app, keys, contractAddrs, toAddrs[start:end], j)
		blk.AppHash = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
		db.SaveBlock(blk)
	}

	blk.TxList = make([][]byte, 8192)
	for i := 0; i < totalNum; i++ {
		GenRandTxList(_app, rs, blk.TxList, keys, contractAddrs, toAddrs)
		blk.AppHash = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
		db.SaveBlock(blk)
	}
}

func RunRecordBlocks(totalNum, fromSize, toSize int) {
	randFilename := os.Getenv("RANDFILE")
	if len(randFilename) == 0 {
		fmt.Printf("No RANDFILE specified. Exiting...")
		return
	}
	rs := randsrc.NewRandSrcFromFile(randFilename)
	keys := ReadKeysFromFile("keys1M.txt")
	blkDB := NewBlockDB(blockDir)
	RecordBlocks(blkDB, rs, totalNum, keys, fromSize, toSize)
}

func RunReplayBlocks(endHeight uint32, fromSize int) {
	blkDB := NewBlockDB(blockDir)
	keys := ReadKeysFromFile("keys1M.txt")
	_app := CreateTestApp(initBalance, keys[:fromSize])
	ReplayBlocks(_app, blkDB, endHeight)
}

// =================

func main() {
	//GenKeysToFile("keys1M.txt", 1000*1000)

	RunRecordBlocks(1000, 10, 1000)
	//RunReplayBlocks(1000, 10)
}
