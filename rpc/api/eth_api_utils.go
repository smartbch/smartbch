package api

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/param"
	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
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
		tx = ethutils.NewTx(nonce, args.To, amount, gasLimit, gasPrice, input)
	} else {
		tx = ethutils.NewTx(nonce, nil, amount, gasLimit, gasPrice, input)
	}

	return tx, nil
}

func blockToRpcResp(block *types.Block, txs []*types.Transaction, sigs [][65]byte) map[string]interface{} {
	result := map[string]interface{}{
		"number":           hexutil.Uint64(block.Number),
		"hash":             hexutil.Bytes(block.Hash[:]),
		"parentHash":       hexutil.Bytes(block.ParentHash[:]),
		"nonce":            hexutil.Bytes(make([]byte, 8)), // PoW specific
		"sha3Uncles":       gethcmn.Hash{},                 // No uncles in Tendermint
		"logsBloom":        gethtypes.Bloom{},
		"transactionsRoot": hexutil.Bytes(block.TransactionsRoot[:]),
		"stateRoot":        hexutil.Bytes(block.StateRoot[:]),
		"miner":            hexutil.Bytes(block.Miner[:]),
		"mixHash":          gethcmn.Hash{},
		"difficulty":       hexutil.Uint64(0),
		"totalDifficulty":  hexutil.Uint64(0),
		"extraData":        hexutil.Bytes(nil),
		"size":             hexutil.Uint64(block.Size),
		"gasLimit":         hexutil.Uint64(param.BlockMaxGas),
		"gasUsed":          hexutil.Uint64(block.GasUsed),
		"timestamp":        hexutil.Uint64(block.Timestamp),
		"transactions":     types.ToGethHashes(block.Transactions),
		"uncles":           []string{},
		"receiptsRoot":     gethcmn.Hash{},
	}

	if len(txs) > 0 {
		rpcTxs := make([]*rpctypes.Transaction, len(txs))
		for i, tx := range txs {
			rpcTxs[i] = txToRpcResp(tx, sigs[i])
		}
		result["transactions"] = rpcTxs
	}

	return result
}

func txsToRpcResp(txs []*types.Transaction, sigs [][65]byte) []*rpctypes.Transaction {
	rpcTxs := make([]*rpctypes.Transaction, len(txs))
	for i, tx := range txs {
		rpcTxs[i] = txToRpcResp(tx, sigs[i])
	}
	return rpcTxs
}

func txToRpcResp(tx *types.Transaction, rawSig [65]byte) *rpctypes.Transaction {
	v, r, s := app.DecodeVRS(rawSig)
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
		TransactionIndex: &idx,
		Value:            (*hexutil.Big)(bigutils.U256FromSlice32(tx.Value[:]).ToBig()),
		V:                (*hexutil.Big)(v),
		R:                (*hexutil.Big)(r),
		S:                (*hexutil.Big)(s),
	}
	copy(resp.BlockHash[:], tx.BlockHash[:])
	if !isZeroAddress(tx.To) {
		resp.To = &gethcmn.Address{}
		copy(resp.To[:], tx.To[:])
	}
	return resp
}

func txsToReceiptRpcResp(txs []*types.Transaction) []map[string]interface{} {
	rpcTxs := make([]map[string]interface{}, len(txs))
	for i, tx := range txs {
		rpcTxs[i] = txToReceiptRpcResp(tx)
	}
	return rpcTxs
}

func txToReceiptRpcResp(tx *types.Transaction) map[string]interface{} {
	resp := map[string]interface{}{
		"transactionHash":   gethcmn.Hash(tx.Hash),
		"transactionIndex":  hexutil.Uint64(tx.TransactionIndex),
		"blockHash":         gethcmn.Hash(tx.BlockHash),
		"blockNumber":       hexutil.Uint64(tx.BlockNumber),
		"from":              gethcmn.Address(tx.From),
		"to":                nil,
		"cumulativeGasUsed": hexutil.Uint64(tx.CumulativeGasUsed),
		"contractAddress":   nil,
		"gasUsed":           hexutil.Uint64(tx.GasUsed),
		"logs":              types.ToGethLogs(tx.Logs),
		"logsBloom":         hexutil.Bytes(tx.LogsBloom[:]),
		"status":            hexutil.Uint(tx.Status),
	}
	if !isZeroAddress(tx.To) {
		resp["to"] = gethcmn.Address(tx.To)
	}
	if !isZeroAddress(tx.ContractAddress) {
		resp["contractAddress"] = gethcmn.Address(tx.ContractAddress)
	}
	if tx.Status == gethtypes.ReceiptStatusFailed {
		resp["statusStr"] = tx.StatusStr
		resp["outData"] = hex.EncodeToString(tx.OutData)
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
