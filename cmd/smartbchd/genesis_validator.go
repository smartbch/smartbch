package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/staking"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"
	"os"
)

/*
type Validator struct {
	Address      [20]byte `msgp:"address"`   // Validator's address in moeing chain
	Pubkey       [32]byte `msgp:"pubkey"`    // Validator's pubkey for tendermint
	RewardTo     [20]byte `msgp:"reward_to"` // where validator's reward goes into
	VotingPower  int64    `msgp:"voting_power"`
	Introduction string   `msgp:"introduction"` // a short introduction
	StakedCoins  [32]byte `msgp:"staked_coins"`
	IsRetiring   bool     `msgp:"is_retiring"` // whether this validator is in a retiring process
}
*/
// GenerateGenesisValidatorCmd returns add-genesis-validator cobra Command.
func GenerateGenesisValidatorCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-genesis-validator [operator_private_key]",
		Short: "Generate and print genesis validator info",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			c := ctx.Config
			c.SetRoot(app.DefaultNodeHome)
			// get private key
			privateKey, _, err := ethutils.HexToPrivKey(args[0])
			if err != nil {
				return err
			}
			// get validator address
			addr := ethutils.PrivKeyToAddr(privateKey)
			// generate new genesis validator
			genVal := stakingtypes.Validator{
				Address:      addr,
				RewardTo:     [20]byte{},
				VotingPower:  1,
				Introduction: "genesis_validator",
				StakedCoins:  staking.InitialStakingAmount.Bytes32(),
				IsRetiring:   false,
			}
			copy(genVal.Pubkey[:], privval.LoadFilePV(c.PrivValidatorKeyFile(), c.PrivValidatorStateFile()).Key.PubKey.Bytes())
			// print validator info, add this to genesis manually
			info, _ := json.Marshal(genVal)
			//fmt.Printf("%s\n", info)
			out := hex.EncodeToString(info)
			fmt.Printf("%s\n", out)
			return nil
		},
	}
	return cmd
}

func AddGenesisValidatorCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-genesis-validator [validator_json_string]",
		Short: "Add genesis validator to genesis.json",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			config := ctx.Config
			config.SetRoot(app.DefaultNodeHome)
			// get new validator info
			s := args[0]
			// check
			v := stakingtypes.Validator{}
			info, err := hex.DecodeString(s)
			if err != nil {
				return err
			}
			err = json.Unmarshal([]byte(info), &v)
			if err != nil {
				return err
			}
			genFile := config.GenesisFile()
			genDoc := &types.GenesisDoc{}
			if _, err := os.Stat(genFile); err != nil {
				if !os.IsNotExist(err) {
					return err
				}
			} else {
				genDoc, err = types.GenesisDocFromFile(genFile)
				if err != nil {
					return err
				}
			}
			gData := app.GenesisData{}
			err = json.Unmarshal(genDoc.AppState, &gData)
			if err != nil {
				return err
			}
			gData.Validators = append(gData.Validators, &v)
			genDoc.AppState, err = json.Marshal(gData)
			if err != nil {
				return err
			}
			if err := ExportGenesisFile(genDoc, genFile); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
