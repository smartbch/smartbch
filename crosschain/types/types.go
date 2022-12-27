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
	OwnerOfLost      [20]byte `msgp:"owner_of_lost"`
	CovenantAddr     [20]byte `msgp:"covenant_addr"`
	IsRedeemed       bool     `msgp:"is_redeemed"`
	RedeemTarget     [20]byte `msgp:"redeem_target"`
	ExpectedSignTime int64    `msgp:"expected_sign_time"`
	Txid             [32]byte `msgp:"txid"`
	Index            uint32   `msgp:"index"`
	Amount           [32]byte `msgp:"amount"`
	BornTime         int64    `msgp:"born_time"`
}

type CCContext struct {
	MonitorsWithPauseCommand   [][20]byte `msgp:"monitor_with_pause_command"`
	RescanTime                 int64      `msgp:"rescan_time"`                    // last startRescan block timestamp, init is max int64
	RescanHeight               uint64     `msgp:"rescan_height"`                  // main chain block height used as rescan end height, init is shaGate enabling height
	LastRescannedHeight        uint64     `msgp:"last_rescanned_height"`          // main chain block height used as rescan start height, init is 0
	UTXOAlreadyHandled         bool       `msgp:"utxo_already_handled"`           // set when call handleUtxo, unset when call startRescan, init is true
	TotalBurntOnMainChain      [32]byte   `msgp:"total_burnt_on_main_chain"`      // init is totalBurnt BCH when shaGate enabling
	TotalMinerFeeForConvertTx  [32]byte   `msgp:"total_miner_fee_for_convert_tx"` // init is zero, the accumulative miner fee used to utxo convert tx, which is covered by side chain black hole balance
	LastCovenantAddr           [20]byte   `msgp:"last_covenant_addr"`             // init is zero address
	CurrCovenantAddr           [20]byte   `msgp:"curr_covenant_addr"`             // init is genesis covenant address
	LatestEpochHandled         int64      `msgp:"latest_epoch_handled"`           // init is zero, the latest epoch number handled for operator or monitor election
	CovenantAddrLastChangeTime int64      `msgp:"covenant_addr_last_change_time"` // init is zero, the latest covenant addr change side chain block timestamp
}

type CCInternalInfosForTest struct {
	TotalRedeemAmountS2M       [32]byte `msgp:"total_redeem_amount_s2m"`
	TotalRedeemNumsS2M         uint64   `msgp:"total_redeem_nums_s2m"`
	TotalLostAndFoundAmountS2M [32]byte `msgp:"total_lost_and_found_amount_s2m"`
	TotalLostAndFoundNumsS2M   uint64   `msgp:"total_lost_and_found_nums_s2m"`
	TotalTransferAmountM2S     [32]byte `msgp:"total_transfer_amount_m2s"`
	TotalTransferNumsM2S       uint64   `msgp:"total_transfer_nums_m2s"`
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
	TotalRedeemAmountS2M       string `json:"totalRedeemAmountS2M"`
	TotalRedeemNumsS2M         uint64 `json:"totalRedeemNumsS2M"`
	TotalLostAndFoundAmountS2M string `json:"totalLostAndFoundAmountS2M"`
	TotalLostAndFoundNumsS2M   uint64 `json:"totalLostAndFoundNumsS2M"`
	TotalTransferAmountM2S     string `json:"totalTransferAmountM2S"`
	TotalTransferNumsM2S       uint64 `json:"totalTransferNumsM2S"`
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
	Number      int64         `msgp:"number"` // same with epoch number, start from param.StartEpochNumberForCC
	StartHeight int64         `msgp:"start_height"`
	EndTime     int64         `msgp:"end_time"`
	Nominations []*Nomination `msgp:"nominations"`
}

type Nomination struct {
	Pubkey         [33]byte `msgp:"pubkey"` // The monitor's compressed pubkey used in main chain
	NominatedCount int64    `msgp:"nominated_count"`
}

type UTXOCollectParam struct {
	BeginHeight            int64
	EndHeight              int64
	CurrentCovenantAddress [20]byte
	PrevCovenantAddress    [20]byte
}
