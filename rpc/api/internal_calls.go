package api

import (
	"fmt"
	"math/big"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	motypes "github.com/smartbch/moeingevm/types"
)

const (
	callModeStaticCall = 1

	callKindCall         = 0
	callKindDelegateCall = 1
	callKindCallCode     = 2
	callKindCreate       = 3
	callKindCreate2      = 4
)

type InternalTx struct {
	depth          int32
	path           string
	count          int
	CallPath       string           `json:"callPath"`
	From           gethcmn.Address  `json:"from"`
	To             gethcmn.Address  `json:"to"`
	GasLimit       hexutil.Uint64   `json:"gas"`
	Value          *hexutil.Big     `json:"value"`
	Input          hexutil.Bytes    `json:"input"`
	StatusCode     hexutil.Uint64   `json:"status"`
	GasUsed        hexutil.Uint64   `json:"gasUsed"`
	Output         hexutil.Bytes    `json:"output"`
	CreatedAddress *gethcmn.Address `json:"contractAddress,omitempty"`
}

func buildCallList(tx *motypes.Transaction) []*InternalTx {
	callList := make([]*InternalTx, len(tx.InternalTxCalls))
	var callStack []*InternalTx

	for i, call := range tx.InternalTxCalls {
		callType := getCallType(call.Kind, call.Flags)
		callSite := newCallSite(call)
		callList[i] = callSite

		if len(callStack) == 0 { // first call
			callStack = append(callStack, callSite)
			callSite.CallPath = callType + getCallPath(callStack)
			continue
		}

		lastCallSite := callStack[len(callStack)-1]
		if call.Depth == lastCallSite.depth+1 { // depth++
			callStack = append(callStack, callSite)
			callSite.CallPath = callType + getCallPath(callStack)
			continue
		}

		if call.Depth <= lastCallSite.depth { // last calls return
			n := lastCallSite.depth - call.Depth
			for j := int32(0); j <= n; j++ {
				if j == n {
					callSite.count = lastCallSite.count + 1
				}
				addRetInfo(lastCallSite, tx.InternalTxReturns[0])
				tx.InternalTxReturns = tx.InternalTxReturns[1:]
				callStack = callStack[:len(callStack)-1]
				lastCallSite = callStack[len(callStack)-1]
			}

			callStack = append(callStack, callSite)
			callSite.CallPath = callType + getCallPath(callStack)
			continue
		}
	}

	for len(callStack) > 0 {
		callSite := callStack[len(callStack)-1]
		callStack = callStack[:len(callStack)-1]
		addRetInfo(callSite, tx.InternalTxReturns[0])
		tx.InternalTxReturns = tx.InternalTxReturns[1:]
	}

	return callList
}

func newCallSite(call motypes.InternalTxCall) *InternalTx {
	return &InternalTx{
		depth:    call.Depth,
		From:     call.Sender,
		To:       call.Destination,
		GasLimit: hexutil.Uint64(call.Gas),
		Value:    (*hexutil.Big)(big.NewInt(0).SetBytes(call.Value[:])),
		Input:    call.Input,
	}
}
func addRetInfo(callSite *InternalTx, ret motypes.InternalTxReturn) {
	callSite.StatusCode = hexutil.Uint64(ret.StatusCode)
	callSite.GasUsed = callSite.GasLimit - hexutil.Uint64(ret.GasLeft)
	callSite.Output = ret.Output
	if !isZeroAddress(ret.CreateAddress) {
		var addr gethcmn.Address = ret.CreateAddress
		callSite.CreatedAddress = &addr
	}
}

func getCallType(kind int, flags uint32) string {
	if flags == callModeStaticCall {
		return "staticcall"
	}
	switch kind {
	case callKindCall:
		return "call"
	case callKindDelegateCall:
		return "delegatecall"
	case callKindCallCode:
		return "callcode"
	case callKindCreate:
		return "create"
	case callKindCreate2:
		return "create2"
	default:
		return "call"
	}
}

func getCallPath(cs []*InternalTx) string {
	path := ""
	for _, call := range cs {
		path += fmt.Sprintf("_%d", call.count)
	}
	return path
}
