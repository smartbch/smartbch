package types

//go:generate msgp
//msgp:ignore typename UTXO CCTransferInfo CCInfosForTest

type UTXO struct {
	TxID   [32]byte
	Index  uint32
	Amount [32]byte
}

type CCTransferInfo struct {
	Type            UTXOType
	PrevUTXO        UTXO
	UTXO            UTXO
	Receiver        [20]byte
	CovenantAddress [20]byte
}

type UTXOType byte

const (
	TransferType             = UTXOType(0)
	ConvertType              = UTXOType(1)
	RedeemOrLostAndFoundType = UTXOType(2)
)

type UTXORecord struct {
	OwnerOfLost      [20]byte
	CovenantAddr     [20]byte
	IsRedeemed       bool
	RedeemTarget     [20]byte
	ExpectedSignTime int64
	Txid             [32]byte
	Index            uint32
	Amount           [32]byte
	BornTime         int64
}

type CCContext struct {
	MonitorsWithPauseCommand   [][20]byte
	RescanTime                 int64    // last startRescan block timestamp, init is max int64
	RescanHeight               uint64   // main chain block height used as rescan end height, init is shaGate enabling height
	LastRescannedHeight        uint64   // main chain block height used as rescan start height, init is 0
	UTXOAlreadyHandled         bool     // set when call handleUtxo, unset when call startRescan, init is true
	TotalBurntOnMainChain      [32]byte // init is totalBurnt BCH when shaGate enabling
	TotalMinerFeeForConvertTx  [32]byte // init is zero, the accumulative miner fee used to utxo convert tx, which is covered by side chain black hole balance
	LastCovenantAddr           [20]byte // init is zero address
	CurrCovenantAddr           [20]byte // init is genesis covenant address
	LatestEpochHandled         int64    // init is zero, the latest epoch number handled for operator or monitor election
	CovenantAddrLastChangeTime int64    // init is zero, the latest covenant addr change side chain block timestamp
}

type CCInternalInfosForTest struct {
	TotalRedeemAmountS2M       [32]byte
	TotalRedeemNumsS2M         uint64
	TotalLostAndFoundAmountS2M [32]byte
	TotalLostAndFoundNumsS2M   uint64
	TotalTransferAmountM2S     [32]byte
	TotalTransferNumsM2S       uint64
	TotalTransferByBurnAmount  [32]byte
	TotalTransferByBurnNums    uint64
}

type CCInfosForTest struct {
	// fixed param
	MaxAmount             string `json:"maxAmount"`
	MinAmount             string `json:"minAmount"`
	MinPendingBurningLeft string `json:"minPendingBurningLeft"`
	// recalculate every rpc call
	PendingBurning            string `json:"pendingBurning"`
	TotalConsumedOnMainChain  string `json:"totalConsumedOnMainChain"`
	TotalMinerFeeForConvertTx string `json:"totalMinerFeeForConvertTx"`
	TotalBurntOnMainChain     string `json:"totalBurntOnMainChain"`
	// get in ads
	TotalRedeemAmountS2M       string   `json:"totalRedeemAmountS2M"`
	TotalRedeemNumsS2M         uint64   `json:"totalRedeemNumsS2M"`
	TotalLostAndFoundAmountS2M string   `json:"totalLostAndFoundAmountS2M"`
	TotalLostAndFoundNumsS2M   uint64   `json:"totalLostAndFoundNumsS2M"`
	TotalTransferAmountM2S     string   `json:"totalTransferAmountM2S"`
	TotalTransferNumsM2S       uint64   `json:"totalTransferNumsM2S"`
	TotalTransferByBurnAmount  string   `json:"totalTransferByBurnAmount"`
	TotalTransferByBurnNums    uint64   `json:"totalTransferByBurnNums"`
	MonitorsWithPauseCommand   []string `json:"monitorsWithPauseCommand"`
	// fields for LostAndFound Test, recalculate every rpc call
	AmountTriggerLostAndFound string `json:"amountTriggerLostAndFound"`
}

type SourceType uint8

const (
	FromRedeemable   = SourceType(0)
	FromLostAndFound = SourceType(1)
	FromRedeeming    = SourceType(2)
	FromBurnRedeem   = SourceType(9)
)

type MonitorVoteInfo struct {
	Number      int64 // same with epoch number, start from param.StartEpochNumberForCC
	StartHeight int64
	EndTime     int64
	Nominations []*Nomination
}

type Nomination struct {
	Pubkey         [33]byte // The monitor's compressed pubkey used in main chain
	NominatedCount int64
}

type UTXOCollectParam struct {
	BeginHeight            int64
	EndHeight              int64
	CurrentCovenantAddress [20]byte
	PrevCovenantAddress    [20]byte
}
