package api

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

const defaultErrorCode = -32000

var _ rpc.Error = callError{}

type callError struct {
	msg  string
	code int
}

func (err callError) Error() string {
	return err.msg
}

func (err callError) ErrorCode() int {
	return err.code
}

// revertError

func newRevertError(retData []byte) *revertError {
	err := errors.New("execution reverted")
	if len(retData) > 0 {
		reason, errUnpack := abi.UnpackRevert(retData)
		if errUnpack == nil {
			err = fmt.Errorf("execution reverted: %v", reason)
		}
	}
	return &revertError{
		error:  err,
		reason: hexutil.Encode(retData),
	}
}

// revertError is an API error that encompassas an EVM revertal with JSON error
// code and a binary data blob.
type revertError struct {
	error
	reason string // revert reason hex encoded
}

// ErrorCode returns the JSON error code for a revertal.
// See: https://eth.wiki/json-rpc/json-rpc-error-codes-improvement-proposal
func (e *revertError) ErrorCode() int {
	return 3
}

// ErrorData returns the hex encoded revert reason.
func (e *revertError) ErrorData() interface{} {
	return e.reason
}
