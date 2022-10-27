package client

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/smartbch/smartbch/rpc/api"
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

func (c *Client) CcCovenantInfo(ctx context.Context) (*api.CcCovenantInfo, error) {
	var result api.CcCovenantInfo
	err := c.rpcClient.CallContext(ctx, &result, "sbch_getCcCovenantInfo")
	return &result, err
}

func (c *Client) RedeemingUtxosForMonitors(ctx context.Context) ([]*api.UtxoInfo, error) {
	var results []*api.UtxoInfo
	err := c.rpcClient.CallContext(ctx, &results, "sbch_getRedeemingUtxosForMonitors")
	return results, err
}

func (c *Client) RedeemingUtxosForOperators(ctx context.Context) ([]*api.UtxoInfo, error) {
	var results []*api.UtxoInfo
	err := c.rpcClient.CallContext(ctx, &results, "sbch_getRedeemingUtxosForOperators")
	return results, err
}

func (c *Client) ToBeConvertedUtxosForMonitors(ctx context.Context) ([]*api.UtxoInfo, error) {
	var results []*api.UtxoInfo
	err := c.rpcClient.CallContext(ctx, &results, "sbch_getToBeConvertedUtxosForMonitors")
	return results, err
}

func (c *Client) ToBeConvertedUtxosForOperators(ctx context.Context) ([]*api.UtxoInfo, error) {
	var results []*api.UtxoInfo
	err := c.rpcClient.CallContext(ctx, &results, "sbch_getToBeConvertedUtxosForOperators")
	return results, err
}
