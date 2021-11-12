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
	CallPath       string           `json:"callPath"`
	From           gethcmn.Address  `json:"from"`
	To             gethcmn.Address  `json:"to"`
	GasLimit       int64            `json:"gasLimit"`
	Value          *hexutil.Big     `json:"value"`
	Input          hexutil.Bytes    `json:"input"`
	StatusCode     int              `json:"statusCode"`
	GasLeft        int64            `json:"gasLeft"`
	Output         hexutil.Bytes    `json:"output"`
	CreatedAddress *gethcmn.Address `json:"createdAddress,omitempty"`
}

func buildCallList(tx *motypes.Transaction) []*InternalTx {
	callList := make([]*InternalTx, len(tx.InternalTxCalls))
	var callStack []*InternalTx

	for i, call := range tx.InternalTxCalls {
		callType := getCallType(call.Kind, call.Flags)
		callSite := newCallSite(call)
		callList[i] = callSite

		if len(callStack) == 0 { // first call
			callSite.path = "_0"
			callSite.CallPath = callType + callSite.path
			callStack = append(callStack, callSite)
			continue
		}

		lastCallSite := callStack[len(callStack)-1]
		if call.Depth == lastCallSite.depth+1 { // depth++
			callSite.path = fmt.Sprintf("%s_%d", lastCallSite.path, call.Depth)
			callSite.CallPath = callType + callSite.path
			callStack = append(callStack, callSite)
			continue
		}

		if call.Depth <= lastCallSite.depth { // last calls return
			n := lastCallSite.depth - call.Depth
			for i := int32(0); i <= n; i++ {
				addRetInfo(lastCallSite, tx.InternalTxReturns[0])
				tx.InternalTxReturns = tx.InternalTxReturns[1:]
				callStack = callStack[:len(callStack)-1]
				lastCallSite = callStack[len(callStack)-1]
			}

			callSite.path = fmt.Sprintf("%s_%d", lastCallSite.path, call.Depth)
			callSite.CallPath = callType + callSite.path
			callStack = append(callStack, callSite)
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
		GasLimit: call.Gas,
		Value:    (*hexutil.Big)(big.NewInt(0).SetBytes(call.Value[:])),
		Input:    call.Input,
	}
}
func addRetInfo(callSite *InternalTx, ret motypes.InternalTxReturn) {
	callSite.StatusCode = ret.StatusCode
	callSite.GasLeft = ret.GasLeft
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
