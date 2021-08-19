package types

//go:generate msgp

type CCTransferInfo struct {
	UTXO         [36]byte
	Amount       uint64
	SenderPubkey [33]byte
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
