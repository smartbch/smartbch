package app_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/param"
)

func TestNewApp(t *testing.T) {
	config := param.DefaultConfig()
	config.AppConfig.AppDataPath = os.ExpandEnv("$PWD/app")
	config.AppConfig.ModbDataPath = os.ExpandEnv("$PWD/modb")
	config.NodeConfig.RootDir = os.ExpandEnv("$PWD")
	err := os.MkdirAll(os.ExpandEnv("$PWD/config"), os.ModePerm)
	if err != nil {
		panic(err)
	}
	file, err := os.Create(os.ExpandEnv("$PWD/config/genesis.json"))
	if err != nil {
		panic(err)
	}
	_, err = file.Write([]byte(genesisData))
	if err != nil {
		panic(err)
	}
	a := createTestApp(config, true, log.NewNopLogger())
	defer func() {
		_ = os.RemoveAll(os.ExpandEnv("$PWD/app"))
		_ = os.RemoveAll(os.ExpandEnv("$PWD/modb"))
		_ = os.RemoveAll(os.ExpandEnv("$PWD/config"))
	}()
	bk := api.NewBackend(a)
	vals := bk.ValidatorsInfo()
	out, err := json.Marshal(vals)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
	//require.Equal(t, )
	go a.Run(0)
	time.Sleep(2 * time.Second)
	blk, _ := bk.BlockByNumber(1)
	require.Equal(t, blk.Miner, [20]byte{1})
	blk, _ = bk.BlockByNumber(5)
	require.Equal(t, blk.Miner, [20]byte{5})
	//key := sha256.Sum256(blk.Hash[:])
	//value := bk.GetStorageAt(api.SEP206ContractAddress, string(key[:]), -1)
	//require.Equal(t, value, hexutil.EncodeUint64(uint64(blk.Number)))
	time.Sleep(1 * time.Second)
}

func createTestApp(config *param.ChainConfig, isInit bool, logger log.Logger) *app.App {
	a := &app.App{}

	/*------set Config------*/
	a.Config = config
	a.ChainId = uint256.NewInt(param.ChainID)

	/*------set util------*/
	a.Logger = logger.With("module", "app")

	/*------set store------*/
	a.Root, a.Mads = app.CreateRootStore(config)
	a.HistoryStore = app.CreateHistoryStore(config, a.Logger.With("module", "modb"))
	a.StateProducer = &mockRpcClient{maxHeight: 10}
	if isInit {
		a.InitGenesisState()
	}
	return a
}

type mockRpcClient struct {
	height    int64
	maxHeight int64
}

var _ app.IStateProducer = &mockRpcClient{}

func (m *mockRpcClient) GeLatestBlock() int64 {
	m.height++
	if m.height > m.maxHeight {
		return m.maxHeight
	}
	return m.height
}

func (m *mockRpcClient) GetSyncBlock(height uint64) *modbtypes.ExtendedBlock {
	b := types.Block{
		Number:           int64(height),
		Hash:             [32]byte{byte(height)},
		ParentHash:       [32]byte{byte(height)},
		LogsBloom:        [256]byte{byte(height)},
		TransactionsRoot: [32]byte{byte(height)},
		StateRoot:        [32]byte{byte(height)},
		Miner:            [20]byte{byte(height)},
		Size:             int64(height),
		GasUsed:          height,
		Timestamp:        int64(3),
		Transactions:     nil,
	}
	blkInfo, _ := b.MarshalMsg(nil)
	e := modbtypes.ExtendedBlock{
		Block: modbtypes.Block{
			Height:    int64(height),
			BlockHash: [32]byte{byte(height)},
			BlockInfo: blkInfo,
			TxList:    nil,
		},
		Txid2sigMap: nil,
		//UpdateOfADS: make(map[string]string),
	}
	//key := sha256.Sum256(b.Hash[:])
	//e.UpdateOfADS[string(types.GetValueKey(2000, string(key[:])))] = hexutil.EncodeUint64(height)
	blk := &types.Block{}
	_, _ = blk.UnmarshalMsg(e.BlockInfo)
	return &e
}

var genesisData = `{
  "genesis_time": "2021-07-30T04:28:16.955082878Z",
  "chain_id": "0x2710",
  "initial_height": "1",
  "consensus_params": {
    "block": {
      "max_bytes": "22020096",
      "max_gas": "-1",
      "time_iota_ms": "1000"
    },
    "evidence": {
      "max_age_num_blocks": "100000",
      "max_age_duration": "172800000000000",
      "max_bytes": "1048576"
    },
    "validator": {
      "pub_key_types": [
        "ed25519"
      ]
    },
    "version": {}
  },
  "app_hash": "",
  "app_state": {
    "validators": [
      {
        "address": "0x9a6dd2f7ceb71788de691844d16b6b6852f07aa3",
        "pubkey": "0xfbdc5c690ab36319d6a68ed50407a61d95d0ec6a6e9225a0c40d17bd8358010e",
        "reward_to": "0x9a6dd2f7ceb71788de691844d16b6b6852f07aa3",
        "voting_power": 10,
        "introduction": "matrixport",
        "staked_coins": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "is_retiring": false
      },
      {
        "address": "0x7dd41d92235cbbe0d2fe4ebd548cdd29f9befe5e",
        "pubkey": "0x45caa8b683a1838f6cf8c3de60ef826ceaac27351843bc9f8c84cedb7da9a8a0",
        "reward_to": "0x7dd41d92235cbbe0d2fe4ebd548cdd29f9befe5e",
        "voting_power": 1,
        "introduction": "btccom",
        "staked_coins": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "is_retiring": false
      },
      {
        "address": "0xde5ddf2a1101d9501aa3db39750acb1764aa5c5b",
        "pubkey": "0xfc609736388585e77dc106885dd401b1dab7be87e61a3597239db9d0483e9a46",
        "reward_to": "0xde5ddf2a1101d9501aa3db39750acb1764aa5c5b",
        "voting_power": 1,
        "introduction": "viabtc",
        "staked_coins": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "is_retiring": false
      }
    ],
    "alloc": {
      "0x9a6dd2f7ceb71788de691844d16b6b6852f07aa3": {
        "balance": "0x115eec47f6cf7e35000000"
      }
    }
  }
}
`
