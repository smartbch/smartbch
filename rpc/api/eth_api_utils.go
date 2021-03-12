package api

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/moeing-chain/MoeingEVM/ebp"
	"github.com/moeing-chain/MoeingEVM/types"
	"github.com/moeing-chain/moeing-chain/internal/bigutils"
	rpctypes "github.com/moeing-chain/moeing-chain/rpc/internal/ethapi"
)

func createGethTxFromSendTxArgs(args rpctypes.SendTxArgs) (*gethtypes.Transaction, error) {
	var (
		nonce    uint64
		gasLimit uint64
	)

	amount := (*big.Int)(args.Value)
	gasPrice := (*big.Int)(args.GasPrice)

	if args.GasPrice == nil {
		gasPrice = big.NewInt(DefaultGasPrice)
	}

	if args.Nonce == nil {
		return nil, errors.New("no nonce")
	} else {
		nonce = (uint64)(*args.Nonce)
	}

	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return nil, errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}

	var input []byte
	if args.Input != nil {
		input = *args.Input
	} else if args.Data != nil {
		input = *args.Data
	}

	if args.To == nil && len(input) == 0 {
		return nil, fmt.Errorf("contract creation without any data provided")
	}
	if args.Gas == nil {
		//return nil, errors.New("no gas limit")
		gasLimit = DefaultRPCGasLimit
	} else {
		gasLimit = (uint64)(*args.Gas)
	}

	var tx *gethtypes.Transaction
	if args.To != nil {
		tx = gethtypes.NewTransaction(nonce, *args.To, amount, gasLimit, gasPrice, input)
	} else {
		tx = gethtypes.NewContractCreation(nonce, amount, gasLimit, gasPrice, input)
	}

	return tx, nil
}

func blockToRpcResp(block *types.Block) map[string]interface{} {
	return map[string]interface{}{
		"number":           hexutil.Uint64(block.Number),
		"hash":             hexutil.Bytes(block.Hash[:]),
		"parentHash":       hexutil.Bytes(block.ParentHash[:]),
		"nonce":            hexutil.Uint64(0), // PoW specific
		"sha3Uncles":       gethcmn.Hash{},    // No uncles in Tendermint
		"logsBloom":        gethtypes.Bloom{},
		"transactionsRoot": hexutil.Bytes(block.TransactionsRoot[:]),
		"stateRoot":        hexutil.Bytes(block.StateRoot[:]),
		"miner":            hexutil.Bytes(block.Miner[:]),
		"mixHash":          gethcmn.Hash{},
		"difficulty":       hexutil.Uint64(0),
		"totalDifficulty":  hexutil.Uint64(0),
		"extraData":        hexutil.Uint64(0),
		"size":             hexutil.Uint64(block.Size),
		"gasLimit":         hexutil.Uint64(0), // Static gas limit
		"gasUsed":          hexutil.Uint64(block.GasUsed),
		"timestamp":        hexutil.Uint64(block.Timestamp),
		"transactions":     types.ToGethHashes(block.Transactions),
		"uncles":           []string{},
		"receiptsRoot":     gethcmn.Hash{},
	}
}

func txToRpcResp(tx *types.Transaction) *rpctypes.Transaction {
	idx := hexutil.Uint64(tx.TransactionIndex)
	resp := &rpctypes.Transaction{
		BlockHash:        &gethcmn.Hash{},
		BlockNumber:      (*hexutil.Big)(big.NewInt(tx.BlockNumber)),
		From:             tx.From,
		Gas:              hexutil.Uint64(tx.Gas),
		GasPrice:         (*hexutil.Big)(bigutils.U256FromSlice32(tx.GasPrice[:]).ToBig()),
		Hash:             tx.Hash,
		Input:            tx.Input,
		Nonce:            hexutil.Uint64(tx.Nonce),
		To:               &gethcmn.Address{},
		TransactionIndex: &idx,
		Value:            (*hexutil.Big)(bigutils.U256FromSlice32(tx.Value[:]).ToBig()),
		//V:
		//R:
		//S:
	}
	copy(resp.BlockHash[:], tx.BlockHash[:])
	copy(resp.To[:], tx.To[:])
	return resp
}

func txToReceiptRpcResp(tx *types.Transaction) map[string]interface{} {
	resp := map[string]interface{}{
		"transactionHash":   gethcmn.Hash(tx.Hash),
		"transactionIndex":  hexutil.Uint64(tx.TransactionIndex),
		"blockHash":         gethcmn.Hash(tx.BlockHash),
		"blockNumber":       hexutil.Uint64(tx.BlockNumber),
		"from":              gethcmn.Address(tx.From),
		"to":                gethcmn.Address(tx.To),
		"cumulativeGasUsed": hexutil.Uint64(tx.CumulativeGasUsed),
		"gasUsed":           hexutil.Uint64(tx.GasUsed),
		"logs":              types.ToGethLogs(tx.Logs),
		"logsBloom":         hexutil.Bytes(tx.LogsBloom[:]),
		"status":            hexutil.Uint(tx.Status),
	}
	if !isZeroAddress(tx.ContractAddress) {
		resp["contractAddress"] = gethcmn.Address(tx.ContractAddress)
	}
	return resp
}

func isZeroAddress(addr [20]byte) bool {
	for _, b := range addr {
		if b != 0 {
			return false
		}
	}
	return true
}

func txsToRpcResp(txs []*types.Transaction) []*rpctypes.Transaction {
	rpcTxs := make([]*rpctypes.Transaction, len(txs))
	for i, tx := range txs {
		rpcTxs[i] = txToRpcResp(tx)
	}
	return rpcTxs
}

func toCallErr(statusCode int, retData []byte) error {
	statusStr := ebp.StatusToStr(statusCode)

	switch statusStr {
	case "revert":
		return newRevertError(retData)
	case "invalid-instruction":
		return callError{code: defaultErrorCode, msg: "invalid opcode"}
	default:
		return callError{code: defaultErrorCode, msg: statusStr}
	}
}
