package watcher

import (
	"fmt"
	"github.com/tendermint/tendermint/libs/log"

	stakingtypes "github.com/smartbch/smartbch/staking/types"
	"github.com/smartbch/smartbch/watcher/types"
)

var _ types.RpcClient = (*ParallelRpcClient)(nil)

type ParallelRpcClient struct {
	client       *RpcClient
	latestHeight int64
	preGetCount  int64
	preGetMaxH   int64
	preGetBlocks map[int64]chan *types.BCHBlock

	logger log.Logger
}

func NewParallelRpcClient(url, user, password string, logger log.Logger) types.RpcClient {
	return &ParallelRpcClient{
		client:      NewRpcClient(url, user, password, "text/plain;", logger),
		preGetCount: 10,
		logger:      logger,
	}
}

func (c *ParallelRpcClient) GetBlockByHash(hash [32]byte) *types.BCHBlock {
	return c.client.GetBlockByHash(hash)
}

func (c *ParallelRpcClient) GetLatestHeight() int64 {
	c.latestHeight = c.client.GetLatestHeight()
	return c.latestHeight
}

func (c *ParallelRpcClient) GetEpochs(start, end uint64) []*stakingtypes.Epoch {
	return c.client.GetEpochs(start, end)
}

func (c *ParallelRpcClient) GetBlockByHeight(height int64) *types.BCHBlock {
	if height+c.preGetCount*2 < c.latestHeight {
		if ch, found := c.preGetBlocks[height]; found {
			delete(c.preGetBlocks, height)
			return <-ch
		}

		if c.preGetMaxH < height {
			c.preGetMaxH = height + c.preGetCount
			fmt.Printf("pre fetch bch blocks: #%d ~ #%d, latest: #%d\n",
				height+1, c.preGetMaxH, c.latestHeight)
			c.getBlocksAsync(height+1, c.preGetCount)
		}
	}

	return c.client.GetBlockByHeight(height)
}

func (c *ParallelRpcClient) getBlocksAsync(hStart, n int64) {
	c.preGetBlocks = map[int64]chan *types.BCHBlock{}

	for i := int64(0); i < n; i++ {
		h := hStart + i
		ch := make(chan *types.BCHBlock)
		c.preGetBlocks[h] = ch

		go func() {
			ch <- c.client.GetBlockByHeight(h)
		}()
	}
}
