package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/libs/cli"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/txscript"

	"github.com/smartbch/smartbch/crosschain/covenant"
	"github.com/smartbch/smartbch/param"
)

const (
	flagBytecodes       = "bytecodes"
	flagOperatorPubkeys = "operator-pubkeys"
	flagMonitorPubkeys  = "monitor-pubkeys"
	flagMinerFee        = "miner-fee"
	flagNet             = "net"
	flagTxid            = "txid"
	flagVout            = "vout"
	flagInAmt           = "in-amt"
	flagToAddr          = "to-addr"
	flagWifs            = "wifs"
	flagSigHash         = "sig-hash"
	flagSigs            = "sigs"
)

func main() {
	rootCmd := createCccCmd()
	executor := cli.Executor{Command: rootCmd, Exit: os.Exit}
	err := executor.Execute()
	if err != nil {
		panic(err)
	}
}

func createCccCmd() *cobra.Command {
	//cobra.EnableCommandSorting = false
	rootCmd := &cobra.Command{
		Use:   "cccovenant",
		Short: "SmartBCH cc-covenants CLI",
	}
	rootCmd.AddCommand(printAddrCmd())
	rootCmd.AddCommand(redeemByUserCmd())
	rootCmd.AddCommand(signTxByOperatorCmd())

	return rootCmd
}

func printAddrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "print-p2sh-addr",
		Short: "print cc-covenant P2SH address",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := viper.BindPFlags(cmd.Flags())
			if err != nil {
				return err
			}

			ccc, err := createCcCovenant()
			if err != nil {
				return err
			}

			addr, err := ccc.GetP2SHAddress()
			if err != nil {
				return err
			}

			addr20, err := ccc.GetP2SHAddress20()
			if err != nil {
				return err
			}

			fmt.Println("operator pubkeys hash:", ccc.GetOperatorPubkeysHash())
			fmt.Println("monitor pubkeys hash :", ccc.GetMonitorPubkeysHash())
			fmt.Println("redeem script hash   :", "0x"+hex.EncodeToString(addr20[:]))
			fmt.Println("P2SH cash address:", addr)
			return nil
		},
	}
	cmd.Flags().SortFlags = false
	addCcBasicFlags(cmd)
	return cmd
}

func redeemByUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redeem-by-user",
		Short: "make (unsigned|signed) redeem-by-user tx",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := viper.BindPFlags(cmd.Flags())
			if err != nil {
				return err
			}

			ccc, err := createCcCovenant()
			if err != nil {
				return err
			}

			txid := gethcmn.FromHex(viper.GetString(flagTxid))
			vout := viper.GetUint32(flagVout)
			inAmt := int64(viper.GetUint64(flagInAmt))
			toAddr := viper.GetString(flagToAddr)

			tx, sigHash, err := ccc.GetRedeemByUserTxSigHash(txid, vout, inAmt, toAddr)
			if err != nil {
				return err
			}

			argSigs := viper.GetStringSlice(flagSigs)
			if nSigs := len(argSigs); nSigs == 0 {
				fmt.Println("unsigned tx:", "0x"+hex.EncodeToString(covenant.MsgTxToBytes(tx)))
				fmt.Println("tx sig hash:", "0x"+hex.EncodeToString(sigHash))
			} else if nSigs < param.MinOperatorSigCount {
				return fmt.Errorf("not enough sigs: %d < %d", nSigs, param.MinOperatorSigCount)
			} else {
				var sigs [][]byte
				for _, argSig := range argSigs {
					sigs = append(sigs, gethcmn.FromHex(argSig))
				}
				signedTx, _, err := ccc.FinishRedeemByUserTx(tx, sigs)
				if err != nil {
					return err
				}
				fmt.Println("signed tx:", "0x"+hex.EncodeToString(covenant.MsgTxToBytes(signedTx)))
			}

			return nil
		},
	}
	cmd.Flags().SortFlags = false
	addCcBasicFlags(cmd)
	cmd.Flags().String(flagTxid, "", "txid of UTXO")
	cmd.Flags().Uint32(flagVout, 0, "output index of UTXO")
	cmd.Flags().Uint64(flagInAmt, 0, "amount of UTXO")
	cmd.Flags().String(flagToAddr, "", "receipt address")
	cmd.Flags().StringSlice(flagSigs, nil, "signatures to make signed tx")
	_ = cmd.MarkFlagRequired(flagTxid)
	_ = cmd.MarkFlagRequired(flagVout)
	_ = cmd.MarkFlagRequired(flagInAmt)
	_ = cmd.MarkFlagRequired(flagToAddr)
	return cmd
}

func signTxByOperatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign-by-operators",
		Short: "sign redeem-by-user or convert tx",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := viper.BindPFlags(cmd.Flags())
			if err != nil {
				return err
			}

			wifs := viper.GetStringSlice(flagWifs)
			sigHash := gethcmn.FromHex(viper.GetString(flagSigHash))
			hashType := txscript.SigHashAll | txscript.SigHashForkID

			for _, wif := range wifs {
				sig, err := covenant.SignCcCovenantTxSigHashECDSA(wif, sigHash, hashType)
				if err != nil {
					return err
				}
				fmt.Println("--sigs=0x" + hex.EncodeToString(sig))
			}

			return nil
		},
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().String(flagSigHash, "", "tx-sig-hash to be signed")
	cmd.Flags().StringSlice(flagWifs, nil, "key of signer in WIF format")
	_ = cmd.MarkFlagRequired(flagWifs)
	_ = cmd.MarkFlagRequired(flagSigHash)

	return cmd
}

func addCcBasicFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagBytecodes, "", "cc-covenant redeem script without constructor args")
	cmd.Flags().StringSlice(flagOperatorPubkeys, nil, "operator pubkeys")
	cmd.Flags().StringSlice(flagMonitorPubkeys, nil, "monitor pubkeys")
	cmd.Flags().Uint64(flagMinerFee, 2000, "miner fee in Satoshi")
	cmd.Flags().String(flagNet, "testnet3", "BCH network, mainnet|testnet3")
	_ = cmd.MarkFlagRequired(flagBytecodes)
	_ = cmd.MarkFlagRequired(flagOperatorPubkeys)
	_ = cmd.MarkFlagRequired(flagMonitorPubkeys)
}

func createCcCovenant() (*covenant.CcCovenant, error) {
	bytecodes := gethcmn.FromHex(viper.GetString(flagBytecodes))

	var operatorPubkeys [][]byte
	for _, s := range viper.GetStringSlice(flagOperatorPubkeys) {
		operatorPubkeys = append(operatorPubkeys, gethcmn.FromHex(s))
	}
	if len(operatorPubkeys) != param.OperatorsCount {
		return nil, fmt.Errorf("length of operator pubkeys must be %d", param.OperatorsCount)
	}

	var monitorPubkeys [][]byte
	for _, s := range viper.GetStringSlice(flagMonitorPubkeys) {
		monitorPubkeys = append(monitorPubkeys, gethcmn.FromHex(s))
	}
	if len(monitorPubkeys) != param.MonitorsCount {
		return nil, fmt.Errorf("length of monitor pubkeys must be %d", param.MonitorsCount)
	}

	minerFee := int64(viper.GetUint64(flagMinerFee))

	var net *chaincfg.Params
	switch s := viper.GetString(flagNet); s {
	case "mainnet":
		net = &chaincfg.MainNetParams
	case "testnet3":
		net = &chaincfg.TestNet3Params
	default:
		return nil, fmt.Errorf("unknown BCH network: %s", s)
	}

	return covenant.NewCcCovenant(bytecodes, operatorPubkeys, monitorPubkeys, minerFee, net)
}
