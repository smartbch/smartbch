package watcher

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/param"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
	"github.com/smartbch/smartbch/watcher/types"
)

type MockBCHNode struct {
	height      int64
	blocks      []*types.BCHBlock
	reorgBlocks map[[32]byte]*types.BCHBlock
}

var testValidatorPubkey1 = [32]byte{0x1}

//var testValidatorPubkey2 = [32]byte{0x2}
//var testValidatorPubkey3 = [32]byte{0x3}

func buildMockBCHNodeWithOnlyValidator1() *MockBCHNode {
	m := &MockBCHNode{
		height: 100,
		blocks: make([]*types.BCHBlock, 100),
	}
	for i := range m.blocks {
		m.blocks[i] = &types.BCHBlock{
			Height:      int64(i + 1),
			Timestamp:   int64(i * 10 * 60),
			HashId:      [32]byte{byte(i + 1)},
			ParentBlk:   [32]byte{byte(i)},
			Nominations: make([]stakingtypes.Nomination, 1),
		}
		m.blocks[i].Nominations[0] = stakingtypes.Nomination{
			Pubkey:         testValidatorPubkey1,
			NominatedCount: 1,
		}
	}
	return m
}

// block at height 99 forked
func buildMockBCHNodeWithReorg() *MockBCHNode {
	m := buildMockBCHNodeWithOnlyValidator1()
	m.blocks[m.height-1] = &types.BCHBlock{
		Height:      100,
		Timestamp:   99 * 10 * 60,
		HashId:      [32]byte{byte(100)},
		ParentBlk:   [32]byte{byte(199)},
		Nominations: make([]stakingtypes.Nomination, 1),
	}
	m.reorgBlocks = make(map[[32]byte]*types.BCHBlock)
	m.reorgBlocks[[32]byte{byte(199)}] = &types.BCHBlock{
		Height:      99,
		Timestamp:   98 * 10 * 60,
		HashId:      [32]byte{byte(199)},
		ParentBlk:   [32]byte{byte(98)},
		Nominations: make([]stakingtypes.Nomination, 1),
	}
	return m
}

type MockRpcClient struct {
	node       *MockBCHNode
	BlockInfos map[int64]*types.BlockInfo
}

func NewMockRpcClient() MockRpcClient {
	return MockRpcClient{BlockInfos: make(map[int64]*types.BlockInfo)}
}

//nolint
func (m MockRpcClient) start() {
	go func() {
		time.Sleep(1 * time.Second)
		m.node.height++
	}()
}

func (m MockRpcClient) Dial() {}

func (m MockRpcClient) Close() {}

func (m MockRpcClient) GetLatestHeight(retry bool) int64 { return m.node.height }

func (m MockRpcClient) GetBlockInfoByHeight(height int64, retry bool) *types.BlockInfo {
	if info, ok := m.BlockInfos[height]; ok {
		return info
	}
	return &types.BlockInfo{}
}

func (m *MockRpcClient) SetBlockInfoByHeight(height int64, info *types.BlockInfo) {
	m.BlockInfos[height] = info
}

func (m MockRpcClient) GetBlockByHeight(height int64, retry bool) *types.BCHBlock {
	if height > m.node.height {
		return nil
	}
	return m.node.blocks[height-1]
}

func (m MockRpcClient) GetBlockByHash(hash [32]byte) *types.BCHBlock {
	height := int64(hash[0])
	if height > m.node.height {
		return m.node.reorgBlocks[hash]
	}
	return m.node.blocks[height-1]
}

func (m MockRpcClient) GetVoteInfoByEpochNumber(start, end uint64) []*types.VoteInfo {
	fmt.Printf("mock Rpc not support get vote info")
	return nil
}

var _ types.RpcClient = MockRpcClient{}

type MockEpochConsumer struct {
	w         *Watcher
	epochList []*stakingtypes.Epoch
}

//nolint
func (m *MockEpochConsumer) consume() {
	for {
		select {
		case e := <-m.w.EpochChan:
			m.epochList = append(m.epochList, e)
		}
	}
}

func TestRun(t *testing.T) {
	w := NewWatcher(log.NewNopLogger(), nil, 0, 0, param.DefaultConfig())
	client := MockRpcClient{node: buildMockBCHNodeWithOnlyValidator1()}
	w.rpcClient = client
	go w.Run()
	w.WaitCatchup()
	time.Sleep(1 * time.Second)
	require.Equal(t, int(100/param.StakingNumBlocksInEpoch), len(w.epochList))
	require.Equal(t, 91, len(w.heightToFinalizedBlock))
	require.Equal(t, int64(91), w.latestFinalizedHeight)
}

func TestRunWithNewEpoch(t *testing.T) {
	w := NewWatcher(log.NewNopLogger(), nil, 0, 0, param.DefaultConfig())
	w.rpcClient = MockRpcClient{node: buildMockBCHNodeWithOnlyValidator1()}
	c := MockEpochConsumer{
		w: w,
	}
	numBlocksInEpoch := 10
	w.SetNumBlocksInEpoch(int64(numBlocksInEpoch))
	go w.Run()
	w.WaitCatchup()
	go c.consume()
	time.Sleep(3 * time.Second)
	//test watcher clear
	//require.Equal(t, 6*int(WatcherNumBlocksInEpoch)-1+10 /*bch finalize block num*/, len(w.hashToBlock))
	require.Equal(t, 6*numBlocksInEpoch, len(w.heightToFinalizedBlock))
	require.Equal(t, 5, len(w.epochList))
	require.Equal(t, int64(91), w.latestFinalizedHeight)
	require.Equal(t, 9, len(c.epochList))
	for i, e := range c.epochList {
		require.Equal(t, int64(i*numBlocksInEpoch)+1, e.StartHeight)
	}
}

func TestRunWithFork(t *testing.T) {
	w := NewWatcher(log.NewNopLogger(), nil, 0, 0, param.DefaultConfig())
	w.rpcClient = MockRpcClient{node: buildMockBCHNodeWithReorg()}
	w.SetNumBlocksInEpoch(1000)
	go w.Run()
	w.WaitCatchup()
	time.Sleep(5 * time.Second)
	require.Equal(t, 0, len(w.epochList))
	require.Equal(t, 91, len(w.heightToFinalizedBlock))
	require.Equal(t, int64(91), w.latestFinalizedHeight)
}

func TestEpochSort(t *testing.T) {
	epoch := &stakingtypes.Epoch{
		Nominations: make([]*stakingtypes.Nomination, 100),
	}
	for i := 0; i < 100; i++ {
		epoch.Nominations[i] = &stakingtypes.Nomination{
			Pubkey:         [32]byte{byte(i)},
			NominatedCount: int64(i/5 + 1),
		}
	}
	sortEpochNominations(epoch)
	epoch.Nominations = epoch.Nominations[:30]
	i := 0
	for j := 1; i < 30 && j < 30; j++ {
		require.True(t, epoch.Nominations[i].NominatedCount > epoch.Nominations[j].NominatedCount ||
			(epoch.Nominations[i].NominatedCount == epoch.Nominations[j].NominatedCount &&
				bytes.Compare(epoch.Nominations[i].Pubkey[:], epoch.Nominations[j].Pubkey[:]) < 0))
		i++
	}
}

func TestGetBCHBlocks(t *testing.T) {
	w := Watcher{}
	c := MockClient{BlockInfos: make(map[int64]*types.BlockInfo)}
	w.rpcClient = c
	for i := int64(0); i < 100; i++ {
		c.SetBlockInfoByHeight(i, &types.BlockInfo{Height: i})
	}
	blks := w.getBCHBlockInfos(0, 30)
	require.Equal(t, 30, len(blks))
	for k, blk := range blks {
		require.Equal(t, int64(k+1), blk.Height)
	}
}
