package api

import (
	"errors"
	"math/big"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/node"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/moeing-chain/MoeingEVM/types"
	"github.com/moeing-chain/moeing-chain/app"
	"github.com/moeing-chain/moeing-chain/param"
)

var _ BackendService = moeingAPIBackend{}

const (
	// Ethereum Wire Protocol
	// https://github.com/ethereum/devp2p/blob/master/caps/eth.md
	protocolVersion = 63
)

type moeingAPIBackend struct {
	//extRPCEnabled bool
	Node *node.Node
	App  *app.App
	//gpo *gasprice.Oracle
}

func NewBackend(node *node.Node, app *app.App) BackendService {
	return moeingAPIBackend{
		Node: node,
		App:  app,
	}
}

func (backend moeingAPIBackend) GetLogs(blockHash common.Hash) (logs [][]types.Log, err error) {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)

	block, err := ctx.GetBlockByHash(blockHash)
	if err == nil && block != nil {
		for _, txHash := range block.Transactions {
			tx, err := ctx.GetTxByHash(txHash)
			if err == nil && tx != nil {
				logs = append(logs, tx.Logs)
			}
		}
	}
	return
}

//func (m moeingAPIBackend) GetReceipts(hash common.Hash) (*types.Transaction, error) {
//	tx, _, _, _, err := m.GetTransaction(hash)
//	return tx, err
//}

func (backend moeingAPIBackend) ChainId() *big.Int {
	return backend.App.ChainID().ToBig()
}

func (backend moeingAPIBackend) GetStorageAt(address common.Address, key string, blockNumber uint64) []byte {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)

	acc := ctx.GetAccount(address)
	if acc == nil {
		return nil
	}
	if blockNumber == 0 {
		return ctx.GetStorageAt(acc.Sequence(), key)
	}
	return nil
}

func (backend moeingAPIBackend) GetCode(contract common.Address) (bytecode []byte, codeHash []byte) {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)
	info := ctx.GetCode(contract)
	if info != nil {
		bytecode = info.BytecodeSlice()
		codeHash = info.CodeHashSlice()
	}
	return
}

func (backend moeingAPIBackend) GetBalance(owner common.Address, height int64) (*big.Int, error) {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)
	b, err := ctx.GetBalance(owner, height)
	if err != nil {
		return nil, err
	}
	return b.ToBig(), nil
}

func (backend moeingAPIBackend) GetNonce(address common.Address) (uint64, error) {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)
	if acc := ctx.GetAccount(address); acc != nil {
		return acc.Nonce(), nil
	}

	return 0, types.ErrAccNotFound
}

func (backend moeingAPIBackend) GetTransaction(txHash common.Hash) (tx *types.Transaction, blockHash common.Hash, blockNumber uint64, blockIndex uint64, err error) {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)

	if tx, err = ctx.GetTxByHash(txHash); err != nil {
		return
	}
	if tx != nil {
		blockHash = tx.BlockHash
		blockNumber = uint64(tx.BlockNumber)
		blockIndex = uint64(tx.TransactionIndex)
	} else {
		err = errors.New("tx with specific hash not exist")
	}
	return
}

func (backend moeingAPIBackend) BlockByHash(hash common.Hash) (*types.Block, error) {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)
	block, err := ctx.GetBlockByHash(hash)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (backend moeingAPIBackend) BlockByNumber(number int64) (*types.Block, error) {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)
	return ctx.GetBlockByHeight(uint64(number))
}

func (backend moeingAPIBackend) ProtocolVersion() int {
	return protocolVersion
}

func (backend moeingAPIBackend) CurrentBlock() *types.Block {
	return backend.App.CurrentBlock()
}

func (backend moeingAPIBackend) ChainConfig() *param.ChainConfig {
	return backend.App.Config
}

func (backend moeingAPIBackend) SendTx(signedTx *types.Transaction) error {
	panic("implement me")
}

func (backend moeingAPIBackend) SendTx2(signedTx []byte) (common.Hash, error) {
	return backend.broadcastTxSync(signedTx)
}

func (backend moeingAPIBackend) broadcastTxSync(tx tmtypes.Tx) (common.Hash, error) {
	resCh := make(chan *abci.Response, 1)
	err := backend.Node.Mempool().CheckTx(tx, func(res *abci.Response) {
		resCh <- res
	}, mempool.TxInfo{})
	if err != nil {
		return common.Hash{}, err
	}
	res := <-resCh
	r := res.GetCheckTx()
	if r.Code != abci.CodeTypeOK {
		return common.Hash{}, errors.New(r.String())
	}
	return common.BytesToHash(tx.Hash()), nil
}

func (backend moeingAPIBackend) Call(tx *gethtypes.Transaction, sender common.Address) []byte {
	runner, _ := backend.App.RunTxForRpc(tx, sender, false)
	return runner.OutData
}

func (backend moeingAPIBackend) EstimateGas(tx *gethtypes.Transaction, sender common.Address) int64 {
	_, gas := backend.App.RunTxForRpc(tx, sender, true)
	return gas
}

func (backend moeingAPIBackend) QueryLogs(addresses []common.Address, topics [][]common.Hash, startHeight, endHeight uint32) ([]types.Log, error) {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)

	return ctx.QueryLogs(addresses, topics, startHeight, endHeight)
}

func (backend moeingAPIBackend) QueryTxBySrc(addr common.Address, startHeight, endHeight uint32) (tx []*types.Transaction, err error) {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)
	return ctx.QueryTxBySrc(addr, startHeight, endHeight)
}

func (backend moeingAPIBackend) QueryTxByDst(addr common.Address, startHeight, endHeight uint32) (tx []*types.Transaction, err error) {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)
	return ctx.QueryTxByDst(addr, startHeight, endHeight)
}

func (backend moeingAPIBackend) QueryTxByAddr(addr common.Address, startHeight, endHeight uint32) (tx []*types.Transaction, err error) {
	ctx := backend.App.GetContext(app.RpcMode)
	defer ctx.Close(false)
	return ctx.QueryTxByAddr(addr, startHeight, endHeight)
}
