package staking

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/smartbch/smartbch/staking/types"
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
			Nominations: make([]types.Nomination, 1),
		}
		m.blocks[i].Nominations[0] = types.Nomination{
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
		Nominations: make([]types.Nomination, 1),
	}
	m.reorgBlocks = make(map[[32]byte]*types.BCHBlock)
	m.reorgBlocks[[32]byte{byte(199)}] = &types.BCHBlock{
		Height:      99,
		Timestamp:   98 * 10 * 60,
		HashId:      [32]byte{byte(199)},
		ParentBlk:   [32]byte{byte(98)},
		Nominations: make([]types.Nomination, 1),
	}
	return m
}

type MockRpcClient struct {
	node *MockBCHNode
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

func (m MockRpcClient) GetLatestHeight() int64 { return m.node.height }

func (m MockRpcClient) GetBlockByHeight(height int64) *types.BCHBlock {
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

func (m MockRpcClient) GetEpochs(start, end uint64) []*types.Epoch {
	fmt.Printf("mock Rpc not support get Epoch")
	return nil
}

var _ types.RpcClient = MockRpcClient{}

type MockEpochConsumer struct {
	w         *Watcher
	epochList []*types.Epoch
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
	client := MockRpcClient{node: buildMockBCHNodeWithOnlyValidator1()}
	w := NewWatcher(0, client, "", 0, false)
	catchupChan := make(chan bool, 1)
	go w.Run(catchupChan)
	<-catchupChan
	time.Sleep(1 * time.Second)
	require.Equal(t, 3, len(w.epochList))
	require.Equal(t, 100, len(w.hashToBlock))
	for h, b := range w.hashToBlock {
		require.True(t, int64(h[0]) == b.Height)
		require.True(t, h == b.HashId)
	}
	require.Equal(t, 90, len(w.heightToFinalizedBlock))
	require.Equal(t, int64(90), w.latestFinalizedHeight)
}

func TestRunWithNewEpoch(t *testing.T) {
	client := MockRpcClient{node: buildMockBCHNodeWithOnlyValidator1()}
	w := NewWatcher(0, client, "", 0, false)
	c := MockEpochConsumer{
		w: w,
	}
	NumBlocksInEpoch = 10
	catchupChan := make(chan bool, 1)
	go w.Run(catchupChan)
	<-catchupChan
	go c.consume()
	time.Sleep(3 * time.Second)
	//test watcher clear
	require.Equal(t, 6*int(NumBlocksInEpoch)-1+10 /*bch finalize block num*/, len(w.hashToBlock))
	require.Equal(t, 6*int(NumBlocksInEpoch)-1, len(w.heightToFinalizedBlock))
	require.Equal(t, 5, len(w.epochList))
	for h, b := range w.hashToBlock {
		require.True(t, int64(h[0]) == b.Height)
		require.True(t, h == b.HashId)
	}
	require.Equal(t, int64(90), w.latestFinalizedHeight)
	require.Equal(t, 9, len(c.epochList))
	for i, e := range c.epochList {
		require.Equal(t, int64(i)*NumBlocksInEpoch+1, e.StartHeight)
		if i == 0 {
			require.Equal(t, 60*10*(NumBlocksInEpoch-1), e.Duration)
		} else {
			require.Equal(t, 60*10*NumBlocksInEpoch, e.Duration)
		}
	}
}

func TestRunWithFork(t *testing.T) {
	client := MockRpcClient{node: buildMockBCHNodeWithReorg()}
	w := NewWatcher(0, client, "", 0, false)
	NumBlocksToClearMemory = 100
	NumBlocksInEpoch = 1000
	catchupChan := make(chan bool, 1)
	go w.Run(catchupChan)
	<-catchupChan
	time.Sleep(5 * time.Second)
	require.Equal(t, 0, len(w.epochList))
	require.Equal(t, int(0), len(w.hashToBlock))
	require.Equal(t, 90, len(w.heightToFinalizedBlock))
	require.Equal(t, int64(90), w.latestFinalizedHeight)
}
