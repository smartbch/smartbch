package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/coinexchain/randsrc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/smartbch/moeingads/indextree"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/staking"
)

type RocksDB = indextree.RocksDB

const (
	adsDir   = "./testdbdata"
	modbDir  = "./modbdata"
	blockDir = "./blkdata"
)

var num1e18 = uint256.NewInt(1_000_000_000_000_000_000)
var initBalance = uint256.NewInt(0).Mul(num1e18, num1e18)
var chainId = bigutils.NewU256(0x2711)

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

//Replay the blocks stored in db onto _app
func ReplayBlocks(_app *testutils.TestApp, db *BlockDB) {
	h := uint32(_app.GetLatestBlockNum())
	fmt.Printf("Start from Block Height: %d\n", h)
	last := time.Now().UnixNano()
	for {
		h++
		blk := db.LoadBlock(h)
		if blk == nil {
			break
		}
		appHash, lastGasUsed := ExecTxsInOneBlock(_app, int64(h), blk.TxList)
		t := time.Now().UnixNano()
		fmt.Printf("Height: %d txCount: %d time: %d lastGasUsed: %d gas/s: %f billion\n", h, len(blk.TxList), t-last, lastGasUsed, float64(lastGasUsed)/float64(t-last))
		last = t
		if !bytes.Equal(appHash, blk.AppHash) {
			fmt.Printf("ref %#v imp %#v\n", appHash, blk.AppHash)
			panic("Incorrect AppHash")
		}
		randomPanic(500, 3433)
	}
}

// generate several private keys and output them to a file
func GenKeysToFile(fname string, count int) {
	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	keys := make([]string, 8000)
	for i := 0; i < count; i += 8000 {
		fmt.Printf("\r%d", i)
		parallelRun(8, func(id int) {
			for i := 0; i < 1000; i++ {
				key, err := crypto.GenerateKey()
				if err != nil {
					panic(err)
				}
				keys[id*1000+i] = hex.EncodeToString(crypto.FromECDSA(key))
			}
		})
		for _, key := range keys {
			fmt.Fprintln(f, key)
		}
	}
	fmt.Println()
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
          "internalType": "uint32",
          "name": "offset",
          "type": "uint32"
        }
      ],
      "name": "run0",
      "outputs": [],
      "stateMutability": "payable",
      "type": "function"
    },
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
          "internalType": "address",
          "name": "to",
          "type": "address"
        },
        {
          "internalType": "address",
          "name": "addr1",
          "type": "address"
        },
        {
          "internalType": "address",
          "name": "addr2",
          "type": "address"
        },
        {
          "internalType": "uint256",
          "name": "param",
          "type": "uint256"
        }
      ],
      "name": "run2",
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
      "stateMutability": "view",
      "type": "function"
    }
]
`)

// get a transaction that calls the `run0` function
func GetTx0(_app *testutils.TestApp, key string, contractAddr, toAddr common.Address, value int64, off uint32) []byte {
	calldata := testAddABI.MustPack("run0", toAddr, off)
	tx, _ := _app.MakeAndSignTxWithGas(key, &contractAddr, value, calldata, testutils.DefaultGasLimit, 2 /*gasprice*/)
	return testutils.MustEncodeTx(tx)
}

// get a transaction that calls the `run2` function
func GetTx(_app *testutils.TestApp, key string, contractAddr, a0, a1, a2 common.Address, value int64, slots [6]uint32) []byte {
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
	calldata := testAddABI.MustPack("run2", a0, a1, a2, param)
	tx, _ := _app.MakeAndSignTxWithGas(key, &contractAddr, value, calldata, testutils.DefaultGasLimit, 2 /*gasprice*/)
	return testutils.MustEncodeTx(tx)
}

// Each sender in privKeys deploy a smart contract
func GetDeployTxAndAddrList(_app *testutils.TestApp, privKeys []string, creationBytecode []byte) ([][]byte, []common.Address) {
	txList := make([][]byte, len(privKeys))
	contractAddrList := make([]common.Address, len(privKeys))
	for i := range privKeys {
		tx, addr := _app.MakeAndSignTxWithGas(privKeys[i], nil, 0 /*value*/, creationBytecode, testutils.DefaultGasLimit, 0 /*gas price*/)
		txList[i] = testutils.MustEncodeTx(tx)
		contractAddrList[i] = crypto.CreateAddress(addr, tx.Nonce())
	}
	return txList, contractAddrList
}

// Apply the transaction in txs onto _app, at the `height`-th block. Caller makes sure the heights are increasing.
func ExecTxsInOneBlock(_app *testutils.TestApp, height int64, txs [][]byte) (appHash []byte, lastGasUsed uint64) {
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
	return responseCommit.Data, _app.GetLastGasUsed()
}

// Generate some transactions to write two slots in each contract and create to-addresses.
func GenFanoutTxList(_app *testutils.TestApp, fromKeys []string, contractAddrs, toAddrs []common.Address, fanoutID int) [][]byte {
	if len(fromKeys) != len(contractAddrs) || len(fromKeys) != len(toAddrs) {
		panic("Invalid lengths")
	}
	res := make([][]byte, 0, len(toAddrs))
	for i, fromKey := range fromKeys {
		tx := GetTx0(_app, fromKey, contractAddrs[i], toAddrs[i], 10000 /*value*/, uint32(10*fanoutID))
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
		z0 := int(rs.GetUint32()) % len(toAddrs)
		z1 := int(rs.GetUint32()) % len(contractAddrs)
		for z1 == y {
			z1 = (z1 + 1) % len(contractAddrs)
		}
		z2 := int(rs.GetUint32()) % len(contractAddrs)
		for z2 == y || z2 == z1 {
			z2 = (z2 + 1) % len(contractAddrs)
		}
		value := int64(rs.GetUint32()) % 2100_0000
		var slots [6]uint32
		slots[0] = rs.GetUint32() % uint32(10*fanoutSize)
		slots[1] = rs.GetUint32() % uint32(10*fanoutSize)
		slots[2] = rs.GetUint32() % uint32(10*fanoutSize)
		slots[3] = rs.GetUint32() % uint32(10*fanoutSize)
		slots[4] = rs.GetUint32() % uint32(10*fanoutSize)
		slots[5] = rs.GetUint32() % uint32(10*fanoutSize)
		txList[i] = GetTx(_app, fromKeys[x], contractAddrs[y], toAddrs[z0], contractAddrs[z1], contractAddrs[z2], value, slots)
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
	params.AppConfig.AppDataPath = adsDir
	params.AppConfig.ModbDataPath = modbDir
	params.AppConfig.UseLiteDB = true
	params.AppConfig.NumKeptBlocks = 5
	testValidatorPubKey := ed25519.GenPrivKeyFromSecret([]byte("stress")).PubKey()
	_app := app.NewApp(params, chainId, 0, log.NewNopLogger(), true)
	if _app.GetLatestBlockNum() == 0 {
		genesisData := app.GenesisData{
			Alloc: testutils.KeysToGenesisAlloc(testInitAmt, keys),
		}
		testValidator := &app.Validator{}
		copy(testValidator.Address[:], testValidatorPubKey.Address().Bytes())
		copy(testValidator.Pubkey[:], testValidatorPubKey.Bytes())
		copy(testValidator.StakedCoins[:], staking.MinimumStakingAmount.Bytes())
		testValidator.VotingPower = 1
		genesisData.Validators = append(genesisData.Validators, testValidator)
		appStateBytes, _ := json.Marshal(genesisData)

		_app.InitChain(abci.RequestInitChain{AppStateBytes: appStateBytes})
		_app.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{
			ProposerAddress: testValidatorPubKey.Address(),
		}})
		_app.Commit()
	}
	return &testutils.TestApp{App: _app, TestPubkey: testValidatorPubKey}
}

func parallelRun(workerCount int, fn func(workerID int)) {
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func(i int) {
			fn(i)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

var creationBytecode = testutils.HexToBytes(`
608060405234801561001057600080fd5b50610be0806100206000396000f3fe60806040526004361061003f5760003560e01c806307b8ae3914610044578063381fd19014610060578063754db9cb1461007c578063d8a26e3a14610098575b600080fd5b61005e600480360381019061005991906107d6565b6100d5565b005b61007a60048036038101906100759190610839565b6103fb565b005b61009660048036038101906100919190610875565b61061f565b005b3480156100a457600080fd5b506100bf60048036038101906100ba91906108b1565b61076f565b6040516100cc9190610969565b60405180910390f35b8373ffffffffffffffffffffffffffffffffffffffff166002346100f99190610a2a565b61232890600067ffffffffffffffff81111561013e577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6040519080825280601f01601f1916602001820160405280156101705781602001600182028036833780820191505090505b5060405161017e9190610929565b600060405180830381858888f193505050503d80600081146101bc576040519150601f19603f3d011682016040523d82523d6000602084013e6101c1565b606091505b50505060008082901c90506000602083901c90506000604084901c90506000606085901c90506000608086901c9050600060a087901c90506002346000808863ffffffff1663ffffffff168152602001908152602001600020546000808a63ffffffff1663ffffffff16815260200190815260200160002054610244919061099a565b61024e919061099a565b6102589190610a2a565b6000808663ffffffff1663ffffffff168152602001908152602001600020819055506002346000808563ffffffff1663ffffffff168152602001908152602001600020546000808763ffffffff1663ffffffff168152602001908152602001600020546102c5919061099a565b6102cf919061099a565b6102d99190610a2a565b6000808363ffffffff1663ffffffff168152602001908152602001600020819055508873ffffffffffffffffffffffffffffffffffffffff1663381fd1906003346103249190610a2a565b8a8a6040518463ffffffff1660e01b8152600401610343929190610940565b6000604051808303818588803b15801561035c57600080fd5b505af1158015610370573d6000803e3d6000fd5b50505050508773ffffffffffffffffffffffffffffffffffffffff1663381fd19060093461039e9190610a2a565b8b8a6040518463ffffffff1660e01b81526004016103bd929190610940565b6000604051808303818588803b1580156103d657600080fd5b505af11580156103ea573d6000803e3d6000fd5b505050505050505050505050505050565b8173ffffffffffffffffffffffffffffffffffffffff163461232890600067ffffffffffffffff811115610458577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6040519080825280601f01601f19166020018201604052801561048a5781602001600182028036833780820191505090505b506040516104989190610929565b600060405180830381858888f193505050503d80600081146104d6576040519150601f19603f3d011682016040523d82523d6000602084013e6104db565b606091505b50505060008082901c90506000602083901c90506000604084901c90506000606085901c90506000608086901c9050600060a087901c90506002346000808863ffffffff1663ffffffff168152602001908152602001600020546000808a63ffffffff1663ffffffff1681526020019081526020016000205461055e919061099a565b610568919061099a565b6105729190610a2a565b6000808663ffffffff1663ffffffff168152602001908152602001600020819055506002346000808563ffffffff1663ffffffff168152602001908152602001600020546000808763ffffffff1663ffffffff168152602001908152602001600020546105df919061099a565b6105e9919061099a565b6105f39190610a2a565b6000808363ffffffff1663ffffffff168152602001908152602001600020819055505050505050505050565b8173ffffffffffffffffffffffffffffffffffffffff163461232890600067ffffffffffffffff81111561067c577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6040519080825280601f01601f1916602001820160405280156106ae5781602001600182028036833780820191505090505b506040516106bc9190610929565b600060405180830381858888f193505050503d80600081146106fa576040519150601f19603f3d011682016040523d82523d6000602084013e6106ff565b606091505b50505060008190505b600a8261071591906109f0565b63ffffffff168163ffffffff16101561076a576003346107359190610a2a565b6000808363ffffffff1663ffffffff16815260200190815260200160002081905550808061076290610ada565b915050610708565b505050565b60008060008363ffffffff1663ffffffff168152602001908152602001600020549050919050565b6000813590506107a681610b65565b92915050565b6000813590506107bb81610b7c565b92915050565b6000813590506107d081610b93565b92915050565b600080600080608085870312156107ec57600080fd5b60006107fa87828801610797565b945050602061080b87828801610797565b935050604061081c87828801610797565b925050606061082d878288016107ac565b91505092959194509250565b6000806040838503121561084c57600080fd5b600061085a85828601610797565b925050602061086b858286016107ac565b9150509250929050565b6000806040838503121561088857600080fd5b600061089685828601610797565b92505060206108a7858286016107c1565b9150509250929050565b6000602082840312156108c357600080fd5b60006108d1848285016107c1565b91505092915050565b6108e381610a5b565b82525050565b60006108f482610984565b6108fe818561098f565b935061090e818560208601610aa7565b80840191505092915050565b61092381610a8d565b82525050565b600061093582846108e9565b915081905092915050565b600060408201905061095560008301856108da565b610962602083018461091a565b9392505050565b600060208201905061097e600083018461091a565b92915050565b600081519050919050565b600081905092915050565b60006109a582610a8d565b91506109b083610a8d565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff038211156109e5576109e4610b07565b5b828201905092915050565b60006109fb82610a97565b9150610a0683610a97565b92508263ffffffff03821115610a1f57610a1e610b07565b5b828201905092915050565b6000610a3582610a8d565b9150610a4083610a8d565b925082610a5057610a4f610b36565b5b828204905092915050565b6000610a6682610a6d565b9050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b600063ffffffff82169050919050565b60005b83811015610ac5578082015181840152602081019050610aaa565b83811115610ad4576000848401525b50505050565b6000610ae582610a97565b915063ffffffff821415610afc57610afb610b07565b5b600182019050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b610b6e81610a5b565b8114610b7957600080fd5b50565b610b8581610a8d565b8114610b9057600080fd5b50565b610b9c81610a97565b8114610ba757600080fd5b5056fea2646970667358221220183a8e054a988b975ca4a780ede1d53700bbe19b8ff975f9994f14851cf19c2064736f6c63430008000033`)

// record `randBlocks` blocks into db for later replay
func RecordBlocks(db *BlockDB, rs randsrc.RandSrc, randBlocks int, keys []string, fromSize, toSize, txPerBlockIn int, ignoreFanout bool) {
	numThreads := 8
	if toSize%fromSize != 0 || toSize%numThreads != 0 {
		panic("Invalid sizes")
	}
	fanoutSize := toSize / fromSize
	if len(keys) < fromSize+toSize {
		panic("not enough keys")
	}
	toAddrs := make([]common.Address, toSize) // to-addresses
	parallelRun(numThreads, func(id int) {
		for i := id * toSize / numThreads; i < (id+1)*toSize/numThreads; i++ {
			if i%10000 == 0 && i != 0 {
				fmt.Printf("Now worker-%d get %d-th to-address\n", id, i)
			}
			toAddrs[i] = KeyToAddr(keys[fromSize+i])
		}
	})
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
		blk.AppHash, _ = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
		db.SaveBlock(blk)
	}
	blk.TxList = nil
	blk.AppHash, _ = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
	db.SaveBlock(blk)
	if !ignoreFanout {
		fmt.Printf("================== Contract Created H=%d ===================\n", db.height)
		// fanout transactions for initializing storage slots and to-addresses
		for fanoutID := 0; fanoutID < fanoutSize; fanoutID++ {
			fmt.Printf("fanoutID: %d\n", fanoutID)
			start, end := fromSize*fanoutID, fromSize*(fanoutID+1)
			half := len(keys) / 2
			mid := (start + end) / 2
			// only half of the from-addresses per block, to avoid incorrect nonce
			blk.TxList = GenFanoutTxList(_app, keys[:half], contractAddrs[:half], toAddrs[start:mid], fanoutID)
			blk.AppHash, _ = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
			db.SaveBlock(blk)
			blk.TxList = GenFanoutTxList(_app, keys[half:], contractAddrs[half:], toAddrs[mid:end], fanoutID)
			blk.AppHash, _ = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
			db.SaveBlock(blk)
		}

		blk.TxList = nil
		blk.AppHash, _ = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
		db.SaveBlock(blk)
	}
	fmt.Printf("================== Storage Slots Written H=%d ===================\n", db.height)
	//ShowSlots(_app, toAddrs[0], contractAddrs, fanoutSize)
	//ShowBalances(_app, keys, toAddrs)

	// change negative value to a random value
	lastTouched := make(map[int]struct{})
	for i := 0; i < randBlocks; i++ {
		txPerBlock := txPerBlockIn
		if txPerBlockIn < 0 {
			txPerBlock = 1 + int(rs.GetUint32())%(-txPerBlockIn)
		}
		// generate blocks with `txPerBlock` random transactions
		blk.TxList = make([][]byte, txPerBlock)
		lastTouched = GenRandTxList(_app, rs, blk.TxList, keys, contractAddrs, toAddrs, lastTouched)
		var lastGasUsed uint64
		blk.AppHash, lastGasUsed = ExecTxsInOneBlock(_app, int64(db.height), blk.TxList)
		fmt.Printf("RandBlock h=%d txcount=%d gas=%d\n", db.height, len(blk.TxList), lastGasUsed)
		db.SaveBlock(blk)
	}
	_app.WaitLock()

	//ShowSlots(_app, toAddrs[0], contractAddrs, fanoutSize)
	//ShowBalances(_app, keys, toAddrs)
}

//nolint
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

//nolint
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

func RunRecordBlocks(randBlocks, fromSize, toSize, txPerBlock int, fname string) {
	_ = os.RemoveAll(adsDir)
	_ = os.RemoveAll(modbDir)
	_ = os.RemoveAll(blockDir)
	_ = os.Mkdir(modbDir, 0700)
	_ = os.Mkdir(blockDir, 0700)

	randFilename := os.Getenv("RANDFILE")
	if len(randFilename) == 0 {
		fmt.Printf("No RANDFILE specified. Exiting...")
		return
	}
	ignoreFanout := os.Getenv("IGNOREFANOUT") == "YES"
	rs := randsrc.NewRandSrcFromFile(randFilename)
	keys := testutils.ReadKeysFromFile(fname, fromSize+toSize)
	fmt.Printf("keys loaded\n")
	blkDB := NewBlockDB(blockDir)
	RecordBlocks(blkDB, rs, randBlocks, keys, fromSize, toSize, txPerBlock, ignoreFanout)
	fmt.Printf("!!!Finished at height %d\n", blkDB.height)
	blkDB.Close()
}

func RunReplayBlocks(fromSize int, fname string) {
	_ = os.RemoveAll(modbDir)
	_ = os.Mkdir(modbDir, 0700)

	blkDB := NewBlockDB(blockDir)
	keys := testutils.ReadKeysFromFile(fname, fromSize)
	_app := CreateTestApp(initBalance, keys[:fromSize])
	ReplayBlocks(_app, blkDB)
	blkDB.Close()
}

func randomPanic(baseNumber, primeNumber int64) {
	heightStr := os.Getenv("RANDOMPANIC")
	if heightStr != "YES" {
		return
	}
	go func(sleepMilliseconds int64) {
		time.Sleep(time.Duration(sleepMilliseconds * int64(time.Millisecond)))
		s := fmt.Sprintf("random panic after %d millisecond", sleepMilliseconds)
		fmt.Println(s)
		panic(s)
	}(baseNumber + time.Now().UnixNano()%primeNumber)
}
