package api

import gethrpc "github.com/ethereum/go-ethereum/rpc"

const defaultErrorCode = -32000

var _ gethrpc.Error = callError{}

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
