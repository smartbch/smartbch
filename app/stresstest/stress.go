package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
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

func (db *BlockDB) Close() {
	db.rdb.Close()
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

//Replay the blockes stored in db onto _app
func ReplayBlocks(_app *testutils.TestApp, db *BlockDB, endHeight uint32) {
	for h := uint32(1); h < endHeight; h++ {
		fmt.Printf("Height %d %d\n", h, time.Now().UnixNano())
		blk := db.LoadBlock(h)
		appHash := ExecTxsInOneBlock(_app, int64(h), blk.TxList)
		if !bytes.Equal(appHash, blk.AppHash) {
			fmt.Printf("ref %#v imp %#v\n", appHash, blk.AppHash)
			panic("Incorrect AppHash")
		}
	}
}

// generate several private keys and output them to a file
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

// read private keys from a file
func ReadKeysFromFile(fname string) (res []string) {
	f, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	res = make([]string, 0, 8192)
	for scanner.Scan() {
		//fmt.Printf("Now read %d\n", len(res))
		txt := scanner.Text()
		//_, err := crypto.HexToECDSA(txt)
		//if err != nil {
		//	panic(err)
		//}
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

// get a transaction that calls the `run` function
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

var creationBytecode = testutils.HexToBytes(`
	608060405234801561001057600080fd5b506102f1806100206000396000f3fe6080604052600436106100295760003560e01c8063381fd1901461002e578063d8a26e3a14610043575b600080fd5b61004161003c3660046101d4565b610079565b005b34801561004f57600080fd5b5061006361005e36600461020a565b6101bc565b604051610070919061026e565b60405180910390f35b604080516000815260208101918290526001600160a01b038416916123289134916100a49190610235565b600060405180830381858888f193505050503d80600081146100e2576040519150601f19603f3d011682016040523d82523d6000602084013e6100e7565b606091505b505050602081811c63ffffffff81811660009081529283905260408084205491851684529283902054849384901c91606085901c91608086901c9160a087901c9160029134916101379190610277565b6101419190610277565b61014b919061029b565b63ffffffff808616600090815260208190526040808220939093558482168152828120549186168152919091205460029134916101889190610277565b6101929190610277565b61019c919061029b565b63ffffffff90911660009081526020819052604090205550505050505050565b63ffffffff1660009081526020819052604090205490565b600080604083850312156101e6578182fd5b82356001600160a01b03811681146101fc578283fd5b946020939093013593505050565b60006020828403121561021b578081fd5b813563ffffffff8116811461022e578182fd5b9392505050565b60008251815b81811015610255576020818601810151858301520161023b565b818111156102635782828501525b509190910192915050565b90815260200190565b6000821982111561029657634e487b7160e01b81526011600452602481fd5b500190565b6000826102b657634e487b7160e01b81526012600452602481fd5b50049056fea2646970667358221220e66a2e809beccf7e9d31ac11ec547ae76cd13f0c56d68a51d2da6224c706fdd864736f6c63430008000033
`)

// Each sender in privKeys deploy a smart contract
func GetDeployTxAndAddrList(_app *testutils.TestApp, privKeys []string, creationBytecode []byte) ([][]byte, []common.Address) {
	txList := make([][]byte, len(privKeys))
	contractAddrList := make([]common.Address, len(privKeys))
	for i := range privKeys {
		tx, addr := _app.MakeAndSignTx(privKeys[i], nil, 0 /*value*/, creationBytecode, 0 /*gas price*/)
		txList[i] = testutils.MustEncodeTx(tx)
		contractAddrList[i] = crypto.CreateAddress(addr, tx.Nonce())
	}
	return txList, contractAddrList
}

// Apply the transaction in txs onto _app, at the `height`-th block. Caller makes sure the heights are increasing.
func ExecTxsInOneBlock(_app *testutils.TestApp, height int64, txs [][]byte) (appHash []byte) {
	_app.BeginBlock(abci.RequestBeginBlock{
		Header: tmproto.Header{
			Height:          height,
			Time:            time.Unix(height*5, 0),
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

// Generate some transactions to write two slots in each contract and create to-addresses.
func GenFanoutTxList(_app *testutils.TestApp, fromKeys []string, contractAddrs, toAddrs []common.Address, fanoutID int) [][]byte {
	if len(fromKeys) != len(contractAddrs) || len(fromKeys) != len(toAddrs) {
		panic("Invalid lengths")
	}
	res := make([][]byte, 0, len(toAddrs))
	for i, fromKey := range fromKeys {
		a := uint32(math.MaxUint32)
		slots := [6]uint32{uint32(2 * fanoutID), a, a, uint32(2*fanoutID + 1), a, a}
		tx := GetTx(_app, fromKey, contractAddrs[i], toAddrs[i], 10000 /*value*/, slots)
		res = append(res, tx)
	}
	return res
}

// fill txList with random transactions
func GenRandTxList(_app *testutils.TestApp, rs randsrc.RandSrc, txList [][]byte, fromKeys []string, contractAddrs, toAddrs []common.Address, lastTouched map[int]struct{}) map[int]struct{} {
	if len(fromKeys) != len(contractAddrs) || len(toAddrs)%len(fromKeys) != 0 {
		panic("Invalid lengths")
	}
	fanoutSize := len(toAddrs) / len(fromKeys)
	touchedFrom := make(map[int]struct{}, len(txList)) //each from-address is only used once for correct nonce
	for i := range txList {
		x := int(rs.GetUint32()) % len(fromKeys)
		for {
			_, touched := touchedFrom[x]
			_, lastTouched := lastTouched[x]
			if !touched && !lastTouched {
				touchedFrom[x] = struct{}{}
				break
			}
			x = int(rs.GetUint32()) % len(fromKeys)
		}
		y := int(rs.GetUint32()) % len(contractAddrs)
		z := int(rs.GetUint32()) % len(toAddrs)
		value := int64(rs.GetUint32()) % 2100_0000
		var slots [6]uint32
		for fanoutID := 0; fanoutID < len(slots); fanoutID++ {
			slots[fanoutID] = rs.GetUint32() % uint32(2*fanoutSize)
		}
		txList[i] = GetTx(_app, fromKeys[x], contractAddrs[y], toAddrs[z], value, slots)
	}
	return touchedFrom
}

func KeyToAddr(keyStr string) common.Address {
	key, err := crypto.HexToECDSA(keyStr)
	if err != nil {
		panic(err)
	}
	return crypto.PubkeyToAddress(key.PublicKey)
}

func CreateTestApp(testInitAmt *uint256.Int, keys []string) *testutils.TestApp {
	//fmt.Printf("CreateTestApp keys %d %#v\n", len(keys), keys)
	params := param.DefaultConfig()
	params.AppDataPath = adsDir
	params.ModbDataPath = modbDir
	params.UseLiteDB = true
	testValidatorPubKey := ed25519.GenPrivKeyFromSecret([]byte("stress")).PubKey()
	_app := app.NewApp(params, bigutils.NewU256(0x2711), log.NewNopLogger())
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

// record `randBlocks` blocks into db for later replay
func RecordBlocks(db *BlockDB, rs randsrc.RandSrc, randBlocks int, keys []string, fromSize, toSize, txPerBlock int) {
	if toSize%fromSize != 0 {
		panic("Invalid sizes")
	}
	fanoutSize := toSize / fromSize
	if len(keys) < fromSize+toSize {
		panic("not enough keys")
	}
	toAddrs := make([]common.Address, toSize) // to-addresses
	for i := range toAddrs {
		if i%10000 == 0 && i != 0 {
			fmt.Printf("Now we get %d to-addresses\n", i)
		}
		toAddrs[i] = KeyToAddr(keys[fromSize+i])
	}
	keys = keys[:fromSize] // from-priv-keys
	_app := CreateTestApp(initBalance, keys)
	txList, contractAddrs := GetDeployTxAndAddrList(_app, keys, creationBytecode)
	blk := &Block{}
	for len(txList) != 0 {
		fmt.Printf("Remaining contracts: %d\n", len(txList))
		length := 512 // at most 512 transactions in a block
		if len(txList) < length {
			length = len(txList)
		}
		blk.TxList = txList[:length]
		txList = txList[length:]
		blk.AppHash = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
		db.SaveBlock(blk)
	}
	blk.TxList = nil
	blk.AppHash = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
	db.SaveBlock(blk)
	// fanout transactions for initializing storage slots and to-addresses
	for fanoutID := 0; fanoutID < fanoutSize; fanoutID++ {
		fmt.Printf("fanoutID: %d\n", fanoutID)
		start, end := fromSize*fanoutID, fromSize*(fanoutID+1)
		half := len(keys) / 2
		mid := (start + end) / 2
		// only half of the from-addresses per block, to avoid incorrect nonce
		blk.TxList = GenFanoutTxList(_app, keys[:half], contractAddrs[:half], toAddrs[start:mid], fanoutID)
		blk.AppHash = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
		db.SaveBlock(blk)
		blk.TxList = GenFanoutTxList(_app, keys[half:], contractAddrs[half:], toAddrs[mid:end], fanoutID)
		blk.AppHash = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
		db.SaveBlock(blk)
	}

	blk.TxList = nil
	blk.AppHash = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
	db.SaveBlock(blk)
	//ShowSlots(_app, toAddrs[0], contractAddrs, fanoutSize)
	//ShowBalances(_app, keys, toAddrs)

	// generate blocks with `txPerBlock` random transactions
	blk.TxList = make([][]byte, txPerBlock)
	lastTouched := make(map[int]struct{})
	for i := 0; i < randBlocks; i++ {
		fmt.Printf("RandBlock %d\n", db.height)
		lastTouched = GenRandTxList(_app, rs, blk.TxList, keys, contractAddrs, toAddrs, lastTouched)
		blk.AppHash = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
		db.SaveBlock(blk)
	}
	_app.WaitLock()

	//ShowSlots(_app, toAddrs[0], contractAddrs, fanoutSize)
	//ShowBalances(_app, keys, toAddrs)
}

func ShowBalances(_app *testutils.TestApp, fromKeys []string, toAddrs []common.Address) {
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)
	var show = func(prefix string, addr common.Address) {
		if acc := ctx.GetAccount(addr); acc != nil {
			fmt.Printf("%s %s : %d\n", prefix, addr, acc.Balance().ToBig())
		} else {
			fmt.Printf("%s %s : ZERO\n", prefix, addr)
		}
	}
	for i, key := range fromKeys {
		show(fmt.Sprintf("FROM-%d", i), KeyToAddr(key))
	}
	for i, addr := range toAddrs {
		show(fmt.Sprintf("TO-%d", i), addr)
	}
}

func ShowSlots(_app *testutils.TestApp, caller common.Address, contractAddrs []common.Address, fanoutSize int) {
	for addrSN, addr := range contractAddrs {
		for i := 0; i < fanoutSize*2; i++ {
			calldata := testAddABI.MustPack("get", uint32(i))
			status, statusStr, retData := _app.Call(caller, addr, calldata)
			n := big.NewInt(0)
			n.SetBytes(retData[:])
			if status != 0 || statusStr != "success" {
				panic("invalid status")
			}
			fmt.Printf("Slot %s(%d) %d %d\n", addr, addrSN, i, n.Int64())
		}
	}
}

func RunRecordBlocks(randBlocks, fromSize, toSize, txPerBlock int) {
	os.RemoveAll(adsDir)
	os.RemoveAll(modbDir)
	os.RemoveAll(blockDir)
	os.Mkdir(modbDir, 0700)
	os.Mkdir(blockDir, 0700)

	randFilename := os.Getenv("RANDFILE")
	if len(randFilename) == 0 {
		fmt.Printf("No RANDFILE specified. Exiting...")
		return
	}
	rs := randsrc.NewRandSrcFromFile(randFilename)
	keys := ReadKeysFromFile("keys1M.txt")
	fmt.Printf("keys loaded\n")
	blkDB := NewBlockDB(blockDir)
	RecordBlocks(blkDB, rs, randBlocks, keys, fromSize, toSize, txPerBlock)
	fmt.Printf("Finished at height %d\n", blkDB.height)
	blkDB.Close()
}

func RunReplayBlocks(endHeight uint32, fromSize int) {
	os.RemoveAll(adsDir)
	os.RemoveAll(modbDir)
	os.Mkdir(modbDir, 0700)

	blkDB := NewBlockDB(blockDir)
	keys := ReadKeysFromFile("keys1M.txt")
	_app := CreateTestApp(initBalance, keys[:fromSize])
	ReplayBlocks(_app, blkDB, endHeight)
	blkDB.Close()
}

// =================

func main() {
	//randBlocks, fromSize, toSize, txPerBlock, endHeight := 100, 10, 100, 4, 124
	randBlocks, fromSize, toSize, txPerBlock, endHeight := 100, 5000, 5000_00, 1024, 312
	//randBlocks, fromSize, toSize, txPerBlock, endHeight := 1000, 50000, 50000_000, 10000,
	if os.Args[1] == "gen" {
		RunRecordBlocks(randBlocks, fromSize, toSize, txPerBlock)
	} else if os.Args[1] == "replay" {
		RunReplayBlocks(uint32(endHeight), fromSize)
	} else if os.Args[1] == "genkeys" {
		GenKeysToFile("keys1M.txt", 1000*1000)
	} else {
		panic("invalid argument")
	}
}
