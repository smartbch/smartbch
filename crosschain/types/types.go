package types

//go:generate msgp

type CCTransferInfo struct {
	PrevUTXO UTXO
	UTXO     UTXO
	Receiver [20]byte
}

type CCEpoch struct {
	Number        int64
	StartHeight   int64
	EndTime       int64
	TransferInfos []*CCTransferInfo
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

type UTXO struct {
	Type   UTXOType `msgp:"type"`
	TxID   [32]byte `msgp:"txid"`
	Index  uint32   `msgp:"index"`
	Amount int64    `msgp:"amount"`
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

type UTXOInfos struct {
	Type    UTXOType
	UtxoSet []UTXO
}
