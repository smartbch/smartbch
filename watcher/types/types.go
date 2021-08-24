package types

import (
	cctypes "github.com/smartbch/smartbch/crosschain/types"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

// These functions must be provided by a client connecting to a Bitcoin Cash's fullnode
type RpcClient interface {
	GetLatestHeight() int64
	GetBlockByHeight(height int64) *BCHBlock
	GetBlockByHash(hash [32]byte) *BCHBlock
	GetEpochs(start, end uint64) []*stakingtypes.Epoch
}

// This struct contains the useful information of a BCH block
type BCHBlock struct {
	Height          int64
	Timestamp       int64
	HashId          [32]byte
	ParentBlk       [32]byte
	Nominations     []stakingtypes.Nomination
	CCTransferInfos []*cctypes.CCTransferInfo
}

//not check Nominations
func (b *BCHBlock) Equal(o *BCHBlock) bool {
	return b.Height == o.Height && b.Timestamp == o.Timestamp &&
		b.HashId == o.HashId && b.ParentBlk == o.ParentBlk
}
