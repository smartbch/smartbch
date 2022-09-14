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
	flagBytecodes          = "bytecodes"
	flagOperatorPubkeys    = "operator-pubkeys"
	flagNewOperatorPubkeys = "new-operator-pubkeys"
	flagMonitorPubkeys     = "monitor-pubkeys"
	flagNewMonitorPubkeys  = "new-monitor-pubkeys"
	flagMinerFee           = "miner-fee"
	flagNet                = "net"
	flagTxid               = "txid"
	flagVout               = "vout"
	flagInAmt              = "in-amt"
	flagToAddr             = "to-addr"
	flagWifs               = "wifs"
	flagSigHash            = "sig-hash"
	flagSigs               = "sigs"
	flagSignedTx           = "signed-tx"
	flagSignerWif          = "signer-wif"
	flagSignerPubkey       = "signer-pubkey"
	flagSignerAddr         = "signer-addr"
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
	rootCmd.AddCommand(convertByOperatorsCmd())
	rootCmd.AddCommand(signTxByOperatorsCmd())

	rootCmd.AddCommand(convertByMonitorsCmd())
	rootCmd.AddCommand(signTxByMonitorsCmd())
	rootCmd.AddCommand(addConvertByMonitorsTxMinerFeeCmd())

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

			addr, err := ccc.GetP2SHAddress()
			if err != nil {
				return err
			}
			fmt.Println("address:", addr)

			txid := gethcmn.FromHex(viper.GetString(flagTxid))
			vout := viper.GetUint32(flagVout)
			inAmt := int64(viper.GetUint64(flagInAmt))
			toAddr := viper.GetString(flagToAddr)
			sigs := getBytesSliceArg(flagSigs)

			if nSigs := len(sigs); nSigs > 0 && nSigs < param.MinOperatorSigCount {
				return fmt.Errorf("not enough sigs: %d < %d", nSigs, param.MinOperatorSigCount)
			}

			tx, sigHash, err := ccc.GetRedeemByUserTxSigHash(txid, vout, inAmt, toAddr)
			if err != nil {
				return err
			}

			if len(sigs) == 0 {
				fmt.Println("unsigned tx:", "0x"+hex.EncodeToString(covenant.MsgTxToBytes(tx)))
				fmt.Println("tx sig hash:", "0x"+hex.EncodeToString(sigHash))
			} else {
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

func convertByOperatorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert-by-operators",
		Short: "make (unsigned|signed) convert-by-operators tx",
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
			fmt.Println("address:", addr)

			txid := gethcmn.FromHex(viper.GetString(flagTxid))
			vout := viper.GetUint32(flagVout)
			inAmt := int64(viper.GetUint64(flagInAmt))
			newOperatorPubkeys := getBytesSliceArg(flagNewOperatorPubkeys)
			newMonitorPubkeys := getBytesSliceArg(flagNewMonitorPubkeys)
			sigs := getBytesSliceArg(flagSigs)

			if len(newOperatorPubkeys) != param.OperatorsCount {
				return fmt.Errorf("length of new operator pubkeys must be %d", param.OperatorsCount)
			}
			if len(newMonitorPubkeys) != param.MonitorsCount {
				return fmt.Errorf("length of new monitor pubkeys must be %d", param.MonitorsCount)
			}
			if nSigs := len(sigs); nSigs > 0 && nSigs < param.MinOperatorSigCount {
				return fmt.Errorf("not enough sigs: %d < %d", nSigs, param.MinOperatorSigCount)
			}

			newAddr, err := ccc.GetP2SHAddressNew(newOperatorPubkeys, newMonitorPubkeys)
			if err != nil {
				return err
			}
			fmt.Println("new address:", newAddr)

			tx, sigHash, err := ccc.GetConvertByOperatorsTxSigHash(txid, vout, inAmt, newOperatorPubkeys, newMonitorPubkeys)
			if err != nil {
				return err
			}

			if len(sigs) == 0 {
				fmt.Println("unsigned tx:", "0x"+hex.EncodeToString(covenant.MsgTxToBytes(tx)))
				fmt.Println("tx sig hash:", "0x"+hex.EncodeToString(sigHash))
			} else {
				signedTx, _, err := ccc.FinishConvertByOperatorsTx(tx, newOperatorPubkeys, newMonitorPubkeys, sigs)
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
	cmd.Flags().StringSlice(flagNewOperatorPubkeys, nil, "new operator pubkeys")
	cmd.Flags().StringSlice(flagNewMonitorPubkeys, nil, "new monitor pubkeys")
	cmd.Flags().StringSlice(flagSigs, nil, "signatures to make signed tx")
	_ = cmd.MarkFlagRequired(flagTxid)
	_ = cmd.MarkFlagRequired(flagVout)
	_ = cmd.MarkFlagRequired(flagInAmt)
	_ = cmd.MarkFlagRequired(flagToAddr)
	_ = cmd.MarkFlagRequired(flagNewOperatorPubkeys)
	_ = cmd.MarkFlagRequired(flagNewMonitorPubkeys)
	return cmd
}

func signTxByOperatorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign-by-operators",
		Short: "sign redeem-by-user or convert-by-operators tx",
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

func convertByMonitorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert-by-monitors",
		Short: "make (unsigned|signed) convert-by-monitors tx",
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
			fmt.Println("address:", addr)

			txid := gethcmn.FromHex(viper.GetString(flagTxid))
			vout := viper.GetUint32(flagVout)
			inAmt := int64(viper.GetUint64(flagInAmt))
			newOperatorPubkeys := getBytesSliceArg(flagNewOperatorPubkeys)
			sigs := getBytesSliceArg(flagSigs)
			if nSigs := len(sigs); nSigs > 0 && nSigs < param.MinMonitorSigCount {
				return fmt.Errorf("not enough sigs: %d < %d", nSigs, param.MinMonitorSigCount)
			}

			tx, sigHash, err := ccc.GetConvertByMonitorsTxSigHash(txid, vout, inAmt, newOperatorPubkeys)
			if err != nil {
				return err
			}

			if len(sigs) == 0 {
				fmt.Println("unsigned tx:", "0x"+hex.EncodeToString(covenant.MsgTxToBytes(tx)))
				fmt.Println("tx sig hash:", "0x"+hex.EncodeToString(sigHash))
			} else {
				tx, err = ccc.AddConvertByMonitorsTxMonitorSigs(tx, newOperatorPubkeys, sigs)
				if err != nil {
					return err
				}
				fmt.Println("monitors signed tx:", "0x"+hex.EncodeToString(covenant.MsgTxToBytes(tx)))
			}

			return nil
		},
	}

	cmd.Flags().SortFlags = false
	addCcBasicFlags(cmd)
	cmd.Flags().String(flagTxid, "", "txid of UTXO")
	cmd.Flags().Uint32(flagVout, 0, "output index of UTXO")
	cmd.Flags().Uint64(flagInAmt, 0, "amount of UTXO")
	cmd.Flags().StringSlice(flagNewOperatorPubkeys, nil, "new operator pubkeys")
	cmd.Flags().StringSlice(flagSigs, nil, "signatures to make signed tx")
	_ = cmd.MarkFlagRequired(flagTxid)
	_ = cmd.MarkFlagRequired(flagVout)
	_ = cmd.MarkFlagRequired(flagInAmt)
	_ = cmd.MarkFlagRequired(flagToAddr)
	_ = cmd.MarkFlagRequired(flagNewOperatorPubkeys)
	return cmd
}

func signTxByMonitorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign-by-monitors",
		Short: "sign convert-by-monitors tx",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := viper.BindPFlags(cmd.Flags())
			if err != nil {
				return err
			}

			wifs := viper.GetStringSlice(flagWifs)
			sigHash := gethcmn.FromHex(viper.GetString(flagSigHash))
			hashType := txscript.SigHashSingle | txscript.SigHashAnyOneCanPay | txscript.SigHashForkID

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

func addConvertByMonitorsTxMinerFeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-miner-fee",
		Short: "add miner fee to signed convert-by-monitors tx",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := viper.BindPFlags(cmd.Flags())
			if err != nil {
				return err
			}

			signedTxData := gethcmn.FromHex(viper.GetString(flagSignedTx))
			signerWif := viper.GetString(flagSignerWif)
			signerPubkey := gethcmn.FromHex(viper.GetString(flagSignerPubkey))
			signerAddr := viper.GetString(flagSignerAddr)
			txid := gethcmn.FromHex(viper.GetString(flagTxid))
			vout := viper.GetUint32(flagVout)
			inAmt := int64(viper.GetUint64(flagInAmt))
			minerFee := int64(viper.GetUint64(flagMinerFee))
			net, err := getBchNetArg()
			if err != nil {
				return err
			}

			tx, err := covenant.MsgTxFromBytes(signedTxData)
			if err != nil {
				return err
			}

			txWithMinerFee, err := covenant.AddConvertByMonitorsTxMinerFee(tx, txid, vout, inAmt, minerFee, signerAddr, net)
			if err != nil {
				return err
			}

			sigHash, err := covenant.GetConvertByMonitorsTxSigHash2(txWithMinerFee, inAmt, signerAddr, net)
			if err != nil {
				return err
			}

			hashType := txscript.SigHashAll | txscript.SigHashForkID
			sig, err := covenant.SignCcCovenantTxSigHashECDSA(signerWif, sigHash, hashType)
			if err != nil {
				return err
			}

			txWithMinerFeeSig, err := covenant.AddConvertByMonitorsTxMinerFeeSig(txWithMinerFee, sig, signerPubkey)
			txData := covenant.MsgTxToBytes(txWithMinerFeeSig)
			fmt.Println("txWithMinerFeeSig:", hex.EncodeToString(txData))

			return nil
		},
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().String(flagSignedTx, "", "signed convert-by-monitors tx")
	cmd.Flags().String(flagSignerWif, "", "key of miner fee provider in WIF format")
	cmd.Flags().String(flagSignerPubkey, "", "pubkey of miner fee provider")
	cmd.Flags().String(flagSignerAddr, "", "address of miner fee provider")
	cmd.Flags().String(flagTxid, "", "txid of UTXO")
	cmd.Flags().Uint32(flagVout, 0, "output index of UTXO")
	cmd.Flags().Uint64(flagInAmt, 0, "amount of UTXO")
	cmd.Flags().Uint64(flagMinerFee, 2000, "miner fee in Satoshi")
	cmd.Flags().String(flagNet, "testnet3", "BCH network, mainnet|testnet3")
	_ = cmd.MarkFlagRequired(flagSignedTx)
	_ = cmd.MarkFlagRequired(flagSignerWif)
	_ = cmd.MarkFlagRequired(flagSignerPubkey)
	_ = cmd.MarkFlagRequired(flagSignerAddr)
	_ = cmd.MarkFlagRequired(flagTxid)
	_ = cmd.MarkFlagRequired(flagVout)
	_ = cmd.MarkFlagRequired(flagInAmt)

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

	operatorPubkeys := getBytesSliceArg(flagOperatorPubkeys)
	if len(operatorPubkeys) != param.OperatorsCount {
		return nil, fmt.Errorf("length of operator pubkeys must be %d", param.OperatorsCount)
	}

	monitorPubkeys := getBytesSliceArg(flagMonitorPubkeys)
	if len(monitorPubkeys) != param.MonitorsCount {
		return nil, fmt.Errorf("length of monitor pubkeys must be %d", param.MonitorsCount)
	}

	minerFee := int64(viper.GetUint64(flagMinerFee))
	net, err := getBchNetArg()
	if err != nil {
		return nil, err
	}

	return covenant.NewCcCovenant(bytecodes, operatorPubkeys, monitorPubkeys, minerFee, net)
}

func getBytesSliceArg(flagName string) [][]byte {
	var result [][]byte
	for _, s := range viper.GetStringSlice(flagName) {
		result = append(result, gethcmn.FromHex(s))
	}
	return result
}

func getBchNetArg() (*chaincfg.Params, error) {
	switch s := viper.GetString(flagNet); s {
	case "mainnet":
		return &chaincfg.MainNetParams, nil
	case "testnet3":
		return &chaincfg.TestNet3Params, nil
	default:
		return nil, fmt.Errorf("unknown BCH network: %s", s)
	}
}
