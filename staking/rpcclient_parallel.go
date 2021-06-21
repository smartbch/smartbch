package staking

import (
	"fmt"
	"sync"

	"github.com/smartbch/smartbch/staking/types"
)

var _ types.RpcClient = (*ParallelRpcClient)(nil)

type ParallelRpcClient struct {
	client       *RpcClient
	rwLock       sync.RWMutex
	latestHeight int64
	preGetMaxH   int64
	preGetBlocks map[int64]*types.BCHBlock
}

func NewParallelRpcClient(url, user, password string) types.RpcClient {
	return &ParallelRpcClient{client: NewRpcClient(url, user, password)}
}

func (c *ParallelRpcClient) GetBlockByHash(hash [32]byte) *types.BCHBlock {
	return c.client.GetBlockByHash(hash)
}

func (c *ParallelRpcClient) GetLatestHeight() int64 {
	c.latestHeight = c.client.GetLatestHeight()
	return c.latestHeight
}

func (c *ParallelRpcClient) GetBlockByHeight(height int64) *types.BCHBlock {
	if height+20 < c.latestHeight {
		c.rwLock.RLock()
		block, found := c.preGetBlocks[height]
		c.rwLock.RUnlock()

		if found {
			return block
		}

		if c.preGetMaxH < height {
			c.preGetMaxH = height + 11
			fmt.Printf("pre fetch bch blocks: #%d ~ #%d, latest: #%d",
				height+1, height+11, c.latestHeight)
			go c.getBlocksAsync(height+1, 10)
		}
	}

	return c.client.GetBlockByHeight(height)
}

func (c *ParallelRpcClient) getBlocksAsync(hStart, n int64) {
	c.rwLock.Lock()
	c.preGetBlocks = map[int64]*types.BCHBlock{}
	c.rwLock.Unlock()

	for i := int64(0); i < n; i++ {
		h := hStart + i
		go func() {
			if b := c.client.GetBlockByHeight(h); b != nil {
				c.rwLock.Lock()
				c.preGetBlocks[h] = b
				c.rwLock.Unlock()
			}
		}()
	}
}
