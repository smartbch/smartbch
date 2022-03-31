//go:build !params_testnet && !params_amber
// +build !params_testnet,!params_amber

package param

const (
	BlockMaxGas             int64  = 1_000_000_000  // 1Billion
	DefaultMinGasPrice      uint64 = 10_000_000_000 // 10gwei
	MaxActiveValidatorCount int    = 50
	IsAmber                 bool   = false
	XHedgeForkBlock         int64  = 70000000
	ShaGateForkBlock        int64  = 80000000
	ChainID                 uint64 = 1000
)
