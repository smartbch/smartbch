package app_test

import (
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
	a := createTestApp(config, log.NewNopLogger())
	defer func() {
		_ = os.RemoveAll(os.ExpandEnv("$PWD/app"))
		_ = os.RemoveAll(os.ExpandEnv("$PWD/modb"))
	}()

	time.Sleep(3 * time.Second)
	bk := api.NewBackend(a)
	blk, _ := bk.BlockByNumber(1)
	require.Equal(t, blk.Miner, [20]byte{1})
	blk, _ = bk.BlockByNumber(5)
	require.Equal(t, blk.Miner, [20]byte{5})
	//key := sha256.Sum256(blk.Hash[:])
	//value := bk.GetStorageAt(api.SEP206ContractAddress, string(key[:]), -1)
	//require.Equal(t, value, hexutil.EncodeUint64(uint64(blk.Number)))
	time.Sleep(1 * time.Second)
}

func createTestApp(config *param.ChainConfig, logger log.Logger) *app.App {
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
	go a.Run(0)
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
