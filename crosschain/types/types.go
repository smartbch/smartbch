package types

//go:generate msgp

type UTXO struct {
	TxID   [32]byte `msgp:"txid"`
	Index  uint32   `msgp:"index"`
	Amount int64    `msgp:"amount"`
}

type CCTransferInfo struct {
	Type     UTXOType `msgp:"type"`
	PrevUTXO UTXO     `msgp:"prev_utxo"`
	UTXO     UTXO     `msgp:"utxo"`
	Receiver [20]byte `msgp:"receiver"`
}

type CCInfo struct {
	GenesisMainnetBlockHeight int64 `msgp:"genesis_mainnet_block_height"`
	CurrEpochNum              int64 `msgp:"curr_epoch_num"`
}

type ScriptInfo struct {
	PrevP2sh         [20]byte `msgp:"prev_p2sh"`
	PrevRedeemScript []byte   `msgp:"prev_redeem_script"`
	P2sh             [20]byte `msgp:"p2sh"`
	RedeemScript     []byte   `msgp:"redeem_script"`
}

type UTXOType byte
type UTXOParam byte

var (
	TransferType             = UTXOType(0)
	ConvertType              = UTXOType(1)
	MonitorCancelRedeemType  = UTXOType(2)
	MonitorCancelConvertType = UTXOType(3)
)

var (
	LatestUTXO = UTXOParam(0)
	AllUTXO    = UTXOParam(1)
)

type UTXORecord struct {
	OwnerOfLost      [20]byte `msgp:"owner_of_lost"`
	CovenantAddr     [20]byte `msgp:"covenant_addr"`
	IsRedeemed       bool     `msgp:"is_redeemed"`
	RedeemTarget     [32]byte `msgp:"redeem_target"`
	ExpectedSignTime int64    `msgp:"expected_sign_time"`
	Txid             [32]byte `msgp:"txid"`
	Index            uint32   `msgp:"index"`
	Amount           [32]byte `msgp:"amount"`
}

type CCContext struct {
	IsPaused              bool     `msgp:"is_paused"`
	RescanTime            int64    `msgp:"rescan_time"`
	RescanHint            [32]byte `msgp:"rescan_hint"`
	LastRescannedHint     [32]byte `msgp:"last_rescanned_hint"`
	UTXOAlreadyHandle     bool     `msgp:"utxo_already_handle"`
	TotalBurntOnMainChain [32]byte `msgp:"total_burnt_on_main_chain"`
	PendingBurning        [32]byte `msgp:"pending_burning"`
	LastCovenantAddr      [20]byte `msgp:"last_covenant_addr"`
	CurrCovenantAddr      [20]byte `msgp:"curr_covenant_addr"`
}
