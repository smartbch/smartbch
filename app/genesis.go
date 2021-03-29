package app

import (
	gethcore "github.com/ethereum/go-ethereum/core"

	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

type GenesisData struct {
	Validators []*stakingtypes.Validator `json:"validators"`
	Alloc      gethcore.GenesisAlloc     `json:"alloc"`
}
