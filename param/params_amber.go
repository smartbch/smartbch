//go:build params_amber
// +build params_amber

package param

//FILE: consensus configurable params collected here!
const (
	BlockMaxGas             int64  = 1_000_000_000 //1Billion
	DefaultMinGasPrice      uint64 = 1_000_000_000 // 1gwei
	MaxActiveValidatorCount int    = 50
	IsAmber                 bool   = true
	XHedgeForkBlock         int64  = 3088100
	ShaGateForkBlock        int64  = 80000000
	ChainID                 uint64 = 1001
)
