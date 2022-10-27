package client

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/smartbch/smartbch/rpc/types"
)

// Client extends ethclient.Client and adds smartBCH specific APIs.
type Client struct {
	*ethclient.Client
	rpcClient *rpc.Client
}

func Dial(rawUrl string) (*Client, error) {
	return DialContext(context.Background(), rawUrl)
}

func DialContext(ctx context.Context, rawUrl string) (*Client, error) {
	c, err := rpc.DialContext(ctx, rawUrl)
	if err != nil {
		return nil, err
	}

	return &Client{
		Client:    ethclient.NewClient(c),
		rpcClient: c,
	}, nil
}

func (c *Client) CcInfo(ctx context.Context) (*types.CcInfo, error) {
	var result types.CcInfo
	err := c.rpcClient.CallContext(ctx, &result, "sbch_getCcInfo")
	return &result, err
}

func (c *Client) RedeemingUtxosForMonitors(ctx context.Context) ([]*types.UtxoInfo, error) {
	var results []*types.UtxoInfo
	err := c.rpcClient.CallContext(ctx, &results, "sbch_getRedeemingUtxosForMonitors")
	return results, err
}

func (c *Client) RedeemingUtxosForOperators(ctx context.Context) ([]*types.UtxoInfo, error) {
	var results []*types.UtxoInfo
	err := c.rpcClient.CallContext(ctx, &results, "sbch_getRedeemingUtxosForOperators")
	return results, err
}

func (c *Client) ToBeConvertedUtxosForMonitors(ctx context.Context) ([]*types.UtxoInfo, error) {
	var results []*types.UtxoInfo
	err := c.rpcClient.CallContext(ctx, &results, "sbch_getToBeConvertedUtxosForMonitors")
	return results, err
}

func (c *Client) ToBeConvertedUtxosForOperators(ctx context.Context) ([]*types.UtxoInfo, error) {
	var results []*types.UtxoInfo
	err := c.rpcClient.CallContext(ctx, &results, "sbch_getToBeConvertedUtxosForOperators")
	return results, err
}
