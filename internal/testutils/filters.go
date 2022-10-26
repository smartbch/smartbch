package testutils

import (
	"math/big"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethfilters "github.com/ethereum/go-ethereum/eth/filters"
)

func NewBlockHashFilter(hash *gethcmn.Hash, addresses ...gethcmn.Address) gethfilters.FilterCriteria {
	return gethfilters.FilterCriteria{
		BlockHash: hash,
		Addresses: addresses,
	}
}

func NewBlockRangeFilter(from, to int64) gethfilters.FilterCriteria {
	return gethfilters.FilterCriteria{
		FromBlock: big.NewInt(from),
		ToBlock:   big.NewInt(to),
	}
}

func NewAddressFilter(addrs ...gethcmn.Address) gethfilters.FilterCriteria {
	return gethfilters.FilterCriteria{
		Addresses: addrs,
	}
}

func NewTopicsFilter(topics [][]gethcmn.Hash) gethfilters.FilterCriteria {
	return gethfilters.FilterCriteria{
		Topics: topics,
	}
}
