package testutils

import (
	"math/big"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethfilters "github.com/ethereum/go-ethereum/eth/filters"
)

type FilterBuilder struct {
	crit gethfilters.FilterCriteria
}

func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{}
}

func (fb *FilterBuilder) BlockHash(blockHash gethcmn.Hash) *FilterBuilder {
	fb.crit.BlockHash = &blockHash
	return fb
}

func (fb *FilterBuilder) BlockRange(from, to int64) *FilterBuilder {
	fb.crit.FromBlock = big.NewInt(from)
	fb.crit.ToBlock = big.NewInt(to)
	return fb
}

func (fb *FilterBuilder) Addresses(addrs ...gethcmn.Address) *FilterBuilder {
	fb.crit.Addresses = addrs
	return fb
}

func (fb *FilterBuilder) Topics(topics [][]gethcmn.Hash) *FilterBuilder {
	fb.crit.Topics = topics
	return fb
}

func (fb *FilterBuilder) Build() gethfilters.FilterCriteria {
	return fb.crit
}
