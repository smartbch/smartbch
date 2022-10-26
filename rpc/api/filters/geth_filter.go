// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package filters

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/smartbch/smartbch/api"
)

type Backend = api.FilterService

func includes(addresses []common.Address, a common.Address) bool {
	for _, addr := range addresses {
		if addr == a {
			return true
		}
	}

	return false
}

func filterFunc(addr common.Address, topics []common.Hash, addrList []common.Address, topicsList [][]common.Hash) (ok bool) {
	if len(addrList) > 0 && !includes(addrList, addr) {
		return false
	}
	// If the to filtered topicsList is greater than the amount of topics in logs, skip.
	if len(topicsList) > len(topics) {
		return false
	}
	for i, sub := range topicsList {
		match := len(sub) == 0 // empty rule set == wildcard
		for _, topic := range sub {
			if topics[i] == topic {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

// filterLogs creates a slice of logs matching the given criteria.
func filterLogs(logs []*types.Log, fromBlock, toBlock *big.Int, addresses []common.Address, topics [][]common.Hash) []*types.Log {
	var ret []*types.Log
	for _, log := range logs {
		if fromBlock != nil && fromBlock.Int64() >= 0 && fromBlock.Uint64() > log.BlockNumber {
			continue
		}
		if toBlock != nil && toBlock.Int64() >= 0 && toBlock.Uint64() < log.BlockNumber {
			continue
		}

		if filterFunc(log.Address, log.Topics, addresses, topics) {
			ret = append(ret, log)
		}
	}
	return ret
}
