package main

import (
	"crypto/ecdsa"
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
	"github.com/smartbch/smartbch/staking"
)

const (
	flagRewardTo = "reward-to"
	flagType     = "type"
)

var stakingABI = ethutils.MustParseABI(`
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
			},
			{
				"internalType": "bytes32",
				"name": "pubkey",
				"type": "bytes32"
			}
		],
		"name": "createValidator",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "decreaseMinGasPrice",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
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
	},
	{
		"inputs": [],
		"name": "increaseMinGasPrice",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "retire",
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
--pubkey=
--reward_to=
--nonce=
--chain-id=
--gasPrice=
--type="create"
--verbose
`,
		RunE: func(_ *cobra.Command, args []string) error {
			c := ctx.Config
			c.SetRoot(viper.GetString(cli.HomeFlag))

			// get private key
			priKey, _, err := ethutils.HexToPrivKey(viper.GetString(flagValKey))
			if err != nil {
				return fmt.Errorf("private key parse error: " + err.Error())
			}
			nonce := viper.GetUint64(flagNonce)
			//todo: get chain id in config.toml
			//chainID := ctx.Config.ChainID()
			chainID, err := parseChainID(viper.GetString(flagChainId))
			if err != nil {
				return fmt.Errorf("parse chain id errpr: %s", err.Error())
			}

			fType := viper.GetString(flagType)
			if fType == "retire" {
				data := stakingABI.MustPack("retire")
				return printSignedTx(big.NewInt(0), data, nonce, priKey, chainID.ToBig())
			}

			// get staking coin
			sCoin, success := bigutils.ParseU256(viper.GetString(flagStakingCoin))
			if !success {
				return fmt.Errorf("staking coin parse failed")
			}
			// generate edit validator info

			var intro [32]byte
			copy(intro[:], viper.GetString(flagIntroduction))

			rewardTo := common.HexToAddress(viper.GetString(flagRewardTo))
			if rewardTo.String() == "" {
				rewardTo = ethutils.PrivKeyToAddr(priKey)
			}
			if fType == "edit" {
				data := stakingABI.MustPack("editValidator", rewardTo, intro)
				return printSignedTx(sCoin.ToBig(), data, nonce, priKey, chainID.ToBig())
			}

			if fType == "create" {
				pubKeyHex := viper.GetString(flagConsPubKey)
				if pubKeyHex == "" {
					return errors.New(flagConsPubKey + " is missing")
				}
				pk, _, err := ethutils.HexToPubKey(pubKeyHex)
				if err != nil {
					return err
				}
				var pubkey [32]byte
				copy(pubkey[:], pk)
				data := stakingABI.MustPack("createValidator", rewardTo, intro, pubkey)
				return printSignedTx(sCoin.ToBig(), data, nonce, priKey, chainID.ToBig())
			}

			return errors.New("invalid staking function type")
		},
	}

	cmd.Flags().String(flagAddress, "", "validator address")
	cmd.Flags().String(flagConsPubKey, "", "consensus pubkey")
	cmd.Flags().Int64(flagVotingPower, 0, "voting power")
	cmd.Flags().String(flagStakingCoin, "0", "staking coin")
	cmd.Flags().String(flagRewardTo, "", "validator rewardTo address")
	cmd.Flags().String(flagType, "", "validator function type, including create, edit, retire")
	cmd.Flags().String(flagIntroduction, "genesis validator", "introduction")
	cmd.Flags().Bool(flagVerbose, false, "display verbose information")
	cmd.Flags().Uint64(flagGasPrice, 1, "specify gas price")
	cmd.Flags().String(flagChainId, "", "specify gas price")
	cmd.Flags().Uint64(flagNonce, 0, "specify tx nonce")
	cmd.Flags().String(flagValKey, "", "specify from address private key")

	_ = cmd.MarkFlagRequired(flagType)
	_ = cmd.MarkFlagRequired(flagValKey)
	_ = cmd.MarkFlagRequired(flagChainId)
	_ = cmd.MarkFlagRequired(flagNonce)
	return cmd
}

func printSignedTx(value *big.Int, data []byte, nonce uint64, priKey *ecdsa.PrivateKey, chainID *big.Int) error {
	to := common.Address(staking.StakingContractAddress)

	txData := &gethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(viper.GetInt64(flagGasPrice)),
		Gas:      staking.GasOfStakingExternalOp,
		To:       &to,
		Value:    value,
		Data:     data,
	}
	tx := gethtypes.NewTx(txData)
	tx, e := ethutils.SignTx(tx, chainID, priKey)
	if e != nil {
		return fmt.Errorf("sign tx errpr: %s", e.Error())
	}
	txBytes, e := ethutils.EncodeTx(tx)
	if e != nil {
		return fmt.Errorf("encode tx errpr: %s", e.Error())
	}
	fmt.Println("0x" + hex.EncodeToString(txBytes))
	if viper.GetBool(flagVerbose) {
		out, _ := tx.MarshalJSON()
		fmt.Println(string(out))
	}
	return nil
}
