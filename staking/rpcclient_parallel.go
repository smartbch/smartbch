package staking

import (
	"fmt"

	"github.com/smartbch/smartbch/staking/types"
)

var _ types.RpcClient = (*ParallelRpcClient)(nil)

type ParallelRpcClient struct {
	client       *RpcClient
	latestHeight int64
	preGetCount  int64
	preGetMaxH   int64
	preGetBlocks map[int64]chan *types.BCHBlock
}

func NewParallelRpcClient(url, user, password string) types.RpcClient {
	return &ParallelRpcClient{
		client:      NewRpcClient(url, user, password, "text/plain;"),
		preGetCount: 10,
	}
}

func (c *ParallelRpcClient) GetBlockByHash(hash [32]byte) *types.BCHBlock {
	return c.client.GetBlockByHash(hash)
}

func (c *ParallelRpcClient) GetLatestHeight() int64 {
	c.latestHeight = c.client.GetLatestHeight()
	return c.latestHeight
}

func (c *ParallelRpcClient) GetEpochs(start, end uint64) []*types.Epoch {
	return c.client.getEpochs(start, end)
}

func (c *ParallelRpcClient) GetBlockByHeight(height int64) *types.BCHBlock {
	if height+c.preGetCount*2 < c.latestHeight {
		if ch, found := c.preGetBlocks[height]; found {
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
