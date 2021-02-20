package main

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/holiman/uint256"
)

var reChainID = regexp.MustCompile("moeing-(\\d+)")

func parseChainID(chainID string) (*uint256.Int, error) {
	if ss := reChainID.FindStringSubmatch(chainID); len(ss) == 2 {
		idU64, err := strconv.ParseUint(ss[1], 10, 64)
		if err == nil {
			idU256 := uint256.NewInt()
			idU256.SetUint64(idU64)
			return idU256, nil
		}
	}
	return nil, errors.New("invalid chain ID: " + chainID)
}
