package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/smartbch/smartbch/rpc/types"
)

// Client extends ethclient.Client and adds smartBCH specific APIs.
type Client struct {
	*ethclient.Client
	rpcClient *rpc.Client
	rpcPubkey []byte
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

func DialHTTP(endpoint string) (*Client, error) {
	return DialHTTPWithClient(endpoint, new(http.Client))
}

func DialHTTPWithClient(endpoint string, client *http.Client) (*Client, error) {
	c, err := rpc.DialHTTPWithClient(endpoint, client)
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
	if err != nil {
		return nil, err
	}
	sig := result.Signature
	result.Signature = nil
	bz, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(bz)
	if len(sig) < 64 {
		return nil, errors.New("invalid signature")
	}
	err = c.getRpcKeyAndVerifySig(ctx, hash[:], sig[:64])
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) verifySigInUtxoInfos(ctx context.Context, infos *types.UtxoInfos) error {
	if infos == nil {
		return errors.New("infos is nil")
	}
	if len(infos.Signature) < 64 {
		return errors.New("invalid signature")
	}
	bz, err := json.Marshal(infos.Infos)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(bz)
	return c.getRpcKeyAndVerifySig(ctx, hash[:], infos.Signature[:64])
}

func (c *Client) RedeemingUtxosForMonitors(ctx context.Context) (*types.UtxoInfos, error) {
	var result *types.UtxoInfos
	err := c.rpcClient.CallContext(ctx, &result, "sbch_getRedeemingUtxosForMonitors")
	if err != nil {
		return nil, err
	}
	return result, c.verifySigInUtxoInfos(ctx, result)
}

func (c *Client) RedeemingUtxosForOperators(ctx context.Context) (*types.UtxoInfos, error) {
	var result *types.UtxoInfos
	err := c.rpcClient.CallContext(ctx, &result, "sbch_getRedeemingUtxosForOperators")
	if err != nil {
		return nil, err
	}
	return result, c.verifySigInUtxoInfos(ctx, result)
}

func (c *Client) RedeemableUtxos(ctx context.Context) (*types.UtxoInfos, error) {
	var result *types.UtxoInfos
	err := c.rpcClient.CallContext(ctx, &result, "sbch_getRedeemableUtxos")
	if err != nil {
		return nil, err
	}
	return result, c.verifySigInUtxoInfos(ctx, result)
}

func (c *Client) LostAndFoundUtxos(ctx context.Context) (*types.UtxoInfos, error) {
	var result *types.UtxoInfos
	err := c.rpcClient.CallContext(ctx, &result, "sbch_getLostAndFoundUtxos")
	if err != nil {
		return nil, err
	}
	return result, c.verifySigInUtxoInfos(ctx, result)
}

func (c *Client) ToBeConvertedUtxosForMonitors(ctx context.Context) (*types.UtxoInfos, error) {
	var result *types.UtxoInfos
	err := c.rpcClient.CallContext(ctx, &result, "sbch_getToBeConvertedUtxosForMonitors")
	if err != nil {
		return nil, err
	}
	return result, c.verifySigInUtxoInfos(ctx, result)
}

func (c *Client) ToBeConvertedUtxosForOperators(ctx context.Context) (*types.UtxoInfos, error) {
	var result *types.UtxoInfos
	err := c.rpcClient.CallContext(ctx, &result, "sbch_getToBeConvertedUtxosForOperators")
	if err != nil {
		return nil, err
	}
	return result, c.verifySigInUtxoInfos(ctx, result)
}

func (c *Client) GetRpcPubkey(ctx context.Context) ([]byte, error) {
	var results string
	err := c.rpcClient.CallContext(ctx, &results, "sbch_getRpcPubkey")
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(results)
}

// this method need rpc server must have a pubkey
func (c *Client) getRpcKeyAndVerifySig(ctx context.Context, hash, sig []byte) error {
	if c.rpcPubkey == nil {
		pubkey, err := c.GetRpcPubkey(ctx)
		if err != nil {
			return err
		}
		c.rpcPubkey = pubkey
	}
	success := crypto.VerifySignature(c.rpcPubkey, hash, sig)
	if !success {
		return errors.New("verify signature failed")
	}
	return nil
}

func (c *Client) CachedRpcPubkey() []byte {
	return c.rpcPubkey
}
