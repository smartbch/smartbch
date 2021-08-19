package main

import (
	"errors"
	"strconv"

	"github.com/holiman/uint256"
)

func parseChainID(chainID string) (*uint256.Int, error) {
	if len(chainID) > 2 && chainID[:2] == "0x" {
		idU64, err := strconv.ParseUint(chainID[2:], 16, 64)
		if err == nil {
			idU256 := uint256.NewInt(0)
			idU256.SetUint64(idU64)
			return idU256, nil
		}
	}
	return nil, errors.New("invalid chain ID: " + chainID)
}
