package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/cli"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/tempfile"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"

	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/internal/ethutils"
	stakingtypes "github.com/smartbch/smartbch/staking/types"
)

const (
	flagAddress      = "validator-address"
	flagPubkey       = "consensus-pubkey"
	flagVotingPower  = "voting-power"
	flagStakingCoin  = "staking-coin"
	flagIntroduction = "introduction"
	flagKey          = "validator-key"
	flagNonce        = "nonce"
	flagChainId      = "chain-id"
	flagGasPrice     = "gas-price"
	flagVerbose      = "verbose"
)

func GenerateConsensusKeyInfoCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-consensus-key-info",
		Short: "Generate and print genesis validator consensus key info",
		Args:  cobra.ExactArgs(0),
		Example: `
smartbchd generate-consensus-key-info
`,
		RunE: func(_ *cobra.Command, args []string) error {
			c := ctx.Config
			c.SetRoot(viper.GetString(cli.HomeFlag))
			pk := ed25519.GenPrivKey()
			fpv := privval.FilePVKey{
				Address: pk.PubKey().Address(),
				PubKey:  pk.PubKey(),
				PrivKey: pk,
			}
			jsonBytes, err := tmjson.MarshalIndent(fpv, "", "  ")
			if err != nil {
				return err
			}
			err = tempfile.WriteFileAtomic("./priv_validator_key.json", jsonBytes, 0600)
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", hex.EncodeToString(pk.PubKey().Bytes()))
			return nil
		},
	}
	return cmd
}

// GenerateGenesisValidatorCmd returns add-genesis-validator cobra Command.
func GenerateGenesisValidatorCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-genesis-validator",
		Short: "Generate and print genesis validator info",
		Example: `
smartbchd generate-genesis-validator 
--validator-address= 
--consensus-pubkey= 
--voting-power= 
--staking-coin=10000000000000 
--introduction="freeman node"
`,
		RunE: func(_ *cobra.Command, args []string) error {
			c := ctx.Config
			c.SetRoot(viper.GetString(cli.HomeFlag))
			// get validator address
			addr := common.HexToAddress(viper.GetString(flagAddress))
			// get pubkey
			pubKey, _, err := ethutils.HexToPubKey(viper.GetString(flagPubkey))
			if err != nil {
				return errors.New("pubkey error: " + err.Error())
			}
			// get staking coin
			sCoin, success := bigutils.ParseU256(viper.GetString(flagStakingCoin))
			if !success {
				return errors.New("staking coin parse failed")
			}
			// generate new genesis validator
			genVal := stakingtypes.Validator{
				Address:      addr,
				RewardTo:     addr,
				VotingPower:  viper.GetInt64(flagVotingPower),
				Introduction: viper.GetString(flagIntroduction),
				StakedCoins:  sCoin.Bytes32(),
				IsRetiring:   false,
			}
			copy(genVal.Pubkey[:], pubKey)
			// print validator info, add this to genesis manually
			info, _ := json.Marshal(genVal)
			//fmt.Printf("%s\n", info)
			out := hex.EncodeToString(info)
			fmt.Printf("%s\n", out)
			return nil
		},
	}
	cmd.Flags().String(flagAddress, "", "validator address")
	cmd.Flags().String(flagPubkey, "", "consensus pubkey")
	cmd.Flags().Int64(flagVotingPower, 0, "voting power")
	cmd.Flags().String(flagStakingCoin, "0", "staking coin")
	cmd.Flags().String(flagIntroduction, "genesis validator", "introduction")
	return cmd
}

func AddGenesisValidatorCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-genesis-validator [validator_json_string]",
		Short: "Add genesis validator to genesis.json",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			config := ctx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))
			// get new validator info
			s := strings.TrimSpace(args[0])
			// check
			v := stakingtypes.Validator{}
			info, err := hex.DecodeString(s)
			if err != nil {
				return err
			}
			err = json.Unmarshal(info, &v)
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
			gData.Validators = append(gData.Validators, app.FromStakingValidator(&v))
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
