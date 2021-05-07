package api

import (
	"encoding/json"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"

	"github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/app"
)

type GenesisDataForRPC struct {
	Validators []ValidatorForRPC     `json:"validators"`
	Alloc      gethcore.GenesisAlloc `json:"alloc"`
}

type ValidatorForRPC struct {
	Address      gethcmn.Address `json:"address"`
	Pubkey       gethcmn.Hash    `json:"pubkey"` // bytes32
	RewardTo     gethcmn.Address `json:"reward_to"`
	VotingPower  int64           `json:"voting_power"`
	Introduction string          `json:"introduction"`
	StakedCoins  gethcmn.Hash    `json:"staked_coins"` // bytes32
	IsRetiring   bool            `json:"is_retiring"`
}

func marshalNodeInfo(nodeInfo api.Info) json.RawMessage {
	castAppState(&nodeInfo)
	bytes, _ := json.Marshal(nodeInfo)
	return bytes
}

func castAppState(nodeInfo *api.Info) {
	genesisData := app.GenesisData{}
	err := json.Unmarshal(nodeInfo.AppState, &genesisData)
	if err != nil {
		return
	}

	genesisDataForRPC := GenesisDataForRPC{
		Alloc:      genesisData.Alloc,
		Validators: make([]ValidatorForRPC, len(genesisData.Validators)),
	}
	for i, v := range genesisData.Validators {
		genesisDataForRPC.Validators[i] = ValidatorForRPC{
			Address:      v.Address,
			Pubkey:       v.Pubkey,
			RewardTo:     v.RewardTo,
			VotingPower:  v.VotingPower,
			Introduction: v.Introduction,
			StakedCoins:  v.StakedCoins,
			IsRetiring:   v.IsRetiring,
		}
	}

	appState, err := json.Marshal(genesisDataForRPC)
	if err != nil {
		return
	}

	nodeInfo.AppState = appState
}
