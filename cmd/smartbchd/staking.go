package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/tendermint/tendermint/libs/cli"

	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/staking"
)

var stakingABI = testutils.MustParseABI(`
[
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "rewardTo",
				"type": "address"
			},
			{
				"internalType": "bytes32",
				"name": "introduction",
				"type": "bytes32"
			}
		],
		"name": "editValidator",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]
`)

func StakingCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "staking",
		Short: "call staking contract method",
		Example: `
smartbchd staking 
--validator-key=
--staking-coin=10000000000000 
--introduction="freeman node"
--nonce=
--chain-id=
--gasPrice=
--verbose
`,
		RunE: func(_ *cobra.Command, args []string) error {
			c := ctx.Config
			c.SetRoot(viper.GetString(cli.HomeFlag))
			// get pubkey
			priKey, _, err := ethutils.HexToPrivKey(viper.GetString(flagKey))
			if err != nil {
				return errors.New("private key parse error: " + err.Error())
			}
			addr := ethutils.PrivKeyToAddr(priKey)
			// get staking coin
			sCoin, success := bigutils.ParseU256(viper.GetString(flagStakingCoin))
			if !success {
				return errors.New("staking coin parse failed")
			}
			// generate edit validator info

			var intro [32]byte
			copy(intro[:], viper.GetString(flagIntroduction))
			data := stakingABI.MustPack("editValidator", addr, intro)
			//todo: get nonce in rpc context
			nonce := viper.GetUint64(flagNonce)
			//todo: get chain id in config.toml
			//chainID := ctx.Config.ChainID()
			chainID, err := parseChainID(viper.GetString(flagChainId))
			if err != nil {
				return errors.New(fmt.Sprintf("parse chain id errpr: %s", err.Error()))
			}
			to := common.Address(staking.StakingContractAddress)
			txData := &gethtypes.LegacyTx{
				Nonce:    nonce,
				GasPrice: big.NewInt(viper.GetInt64(flagGasPrice)),
				Gas:      staking.GasOfStakingExternalOp,
				To:       &to,
				Value:    sCoin.ToBig(),
				Data:     data,
			}
			tx := gethtypes.NewTx(txData)
			tx, err = ethutils.SignTx(tx, chainID.ToBig(), priKey)
			if err != nil {
				return errors.New(fmt.Sprintf("sign tx errpr: %s", err.Error()))
			}
			txBytes, err := ethutils.EncodeTx(tx)
			if err != nil {
				return errors.New(fmt.Sprintf("encode tx errpr: %s", err.Error()))
			}
			fmt.Println("0x" + hex.EncodeToString(txBytes))
			if viper.GetBool(flagVerbose) {
				out, _ := tx.MarshalJSON()
				fmt.Println(string(out))
			}
			return nil
		},
	}
	cmd.Flags().String(flagAddress, "", "validator address")
	cmd.Flags().String(flagPubkey, "", "consensus pubkey")
	cmd.Flags().Int64(flagVotingPower, 0, "voting power")
	cmd.Flags().String(flagStakingCoin, "0", "staking coin")
	cmd.Flags().String(flagIntroduction, "genesis validator", "introduction")
	cmd.Flags().Bool(flagVerbose, false, "display verbose information")
	cmd.Flags().Uint64(flagGasPrice, 1, "specify gas price")
	cmd.Flags().String(flagChainId, "", "specify gas price")
	cmd.Flags().Uint64(flagNonce, 0, "specify tx nonce")
	cmd.Flags().String(flagKey, "", "specify from address private key")
	return cmd
}
