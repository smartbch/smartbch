package types

//go:generate msgp

type UTXO struct {
	TxID   [32]byte `msgp:"txid"`
	Index  uint32   `msgp:"index"`
	Amount [32]byte `msgp:"amount"`
}

type CCTransferInfo struct {
	Type     UTXOType `msgp:"type"`
	PrevUTXO UTXO     `msgp:"prev_utxo"`
	UTXO     UTXO     `msgp:"utxo"`
	/*
		when Type is TransferType: receiver is side chain address to receive sBCH;
		when Type is ConvertType:  receiver is covenant address in empty;
		when Type is RedeemOrLostAndFoundType:  receiver is locking address used when redeem or pubkey hash address used for lostAndFound;
	*/
	Receiver        [20]byte `msgp:"receiver"`
	CovenantAddress [20]byte `msgp:"covenant_address"`
}

type UTXOType byte
type UTXOParam byte

var (
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
}

type CCContext struct {
	IsPaused              bool     `msgp:"is_paused"`
	RescanTime            int64    `msgp:"rescan_time"`
	RescanHeight          uint64   `msgp:"rescan_hint"`
	LastRescannedHeight   uint64   `msgp:"last_rescanned_hint"`
	UTXOAlreadyHandle     bool     `msgp:"utxo_already_handle"`
	TotalBurntOnMainChain [32]byte `msgp:"total_burnt_on_main_chain"`
	PendingBurning        [32]byte `msgp:"pending_burning"`
	LastCovenantAddr      [20]byte `msgp:"last_covenant_addr"`
	CurrCovenantAddr      [20]byte `msgp:"curr_covenant_addr"`
}

type SourceType uint8

var (
	FromRedeemable   = SourceType(0)
	FromLostAndFound = SourceType(1)
	FromRedeeming    = SourceType(2)
	FromBurnRedeem   = SourceType(9)
)
