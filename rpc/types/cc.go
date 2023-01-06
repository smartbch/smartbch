package types

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type OperatorInfo struct {
	Address gethcmn.Address `json:"address"`
	Pubkey  hexutil.Bytes   `json:"pubkey"`
	RpcUrl  string          `json:"rpcUrl"`
	Intro   string          `json:"intro"`
}

type MonitorInfo struct {
	Address gethcmn.Address `json:"address"`
	Pubkey  hexutil.Bytes   `json:"pubkey"`
	Intro   string          `json:"intro"`
}

type CcInfo struct {
	MonitorsWithPauseCommand   []string        `json:"monitorsWithPauseCommand"`
	Operators                  []*OperatorInfo `json:"operators"`
	Monitors                   []*MonitorInfo  `json:"monitors"`
	OldOperators               []*OperatorInfo `json:"oldOperators"`
	OldMonitors                []*MonitorInfo  `json:"oldMonitors"`
	LastCovenantAddress        string          `json:"lastCovenantAddress"`
	CurrCovenantAddress        string          `json:"currCovenantAddress"`
	LastRescannedHeight        uint64          `json:"lastRescannedHeight"`
	RescannedHeight            uint64          `json:"rescannedHeight"`
	RescanTime                 int64           `json:"rescanTime"`
	UTXOAlreadyHandled         bool            `json:"utxoAlreadyHandled"`
	LatestEpochHandled         int64           `json:"latestEpochHandled"`
	CovenantAddrLastChangeTime int64           `json:"covenantAddrLastChangeTime"`
	Signature                  hexutil.Bytes   `json:"signature"`
}

type UtxoInfo struct {
	OwnerOfLost      gethcmn.Address `json:"ownerOfLost"`
	CovenantAddr     gethcmn.Address `json:"covenantAddr"`
	IsRedeemed       bool            `json:"isRedeemed"`
	RedeemTarget     gethcmn.Address `json:"redeemTarget"`
	ExpectedSignTime int64           `json:"expectedSignTime"`
	Txid             gethcmn.Hash    `json:"txid"`
	Index            uint32          `json:"index"`
	Amount           hexutil.Uint64  `json:"amount"` // in satoshi
	TxSigHash        hexutil.Bytes   `json:"txSigHash"`
}

type UtxoInfos struct {
	Infos     []*UtxoInfo   `json:"infos"`
	Signature hexutil.Bytes `json:"signature"`
}
