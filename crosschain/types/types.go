package types

import "github.com/ethereum/go-ethereum/common"

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
		when Type is ConvertType:  receiver and covenant address in empty;
		when Type is RedeemOrLostAndFoundType:  receiver is locking address used when redeem or pubkey hash address used for lostAndFound;
	*/
	Receiver        [20]byte `msgp:"receiver"`
	CovenantAddress [20]byte `msgp:"covenant_address"`
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
	IsPaused              bool     `msgp:"is_paused"`
	RescanTime            int64    `msgp:"rescan_time"`               // last startRescan block timestamp, init is max int64
	RescanHeight          uint64   `msgp:"rescan_hint"`               // main chain block height used as rescan end height, init is shaGate enabling height
	LastRescannedHeight   uint64   `msgp:"last_rescanned_hint"`       // main chain block height used as rescan start height, init is 0
	UTXOAlreadyHandled    bool     `msgp:"utxo_already_handle"`       // set when call handleUtxo, unset when call startRescan, init is true
	TotalBurntOnMainChain [32]byte `msgp:"total_burnt_on_main_chain"` // init is totalBurnt BCH when shaGate enabling
	LastCovenantAddr      [20]byte `msgp:"last_covenant_addr"`        //init is zero address
	CurrCovenantAddr      [20]byte `msgp:"curr_covenant_addr"`        //init is genesis covenant address
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
	CurrentCovenantAddress common.Address
	PrevCovenantAddress    common.Address
}
