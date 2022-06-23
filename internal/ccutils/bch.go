package ccutils

import (
	"encoding/hex"
	"fmt"

	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchutil"
)

func GetMultiSigP2SHAddr(redeemScriptWithoutConstructorArgs string,
	operatorPks []string, monitorPks []string) (string, error) {

	builder := txscript.NewScriptBuilder()

	for i := len(monitorPks) - 1; i >= 0; i-- {
		pk, err := hex.DecodeString(monitorPks[i])
		if err != nil {
			return "", fmt.Errorf("failed to decode monitorPk#%d", i)
		}
		if len(pk) != 33 {
			return "", fmt.Errorf("len of monitorPk#%d is not 33", i)
		}

		builder.AddData(pk)
	}

	for i := len(operatorPks) - 1; i >= 0; i-- {
		pk, err := hex.DecodeString(operatorPks[i])
		if err != nil {
			return "", fmt.Errorf("failed to decode operatorPk#%d", i)
		}
		if len(pk) != 33 {
			return "", fmt.Errorf("len of operatorPk#%d is not 33", i)
		}

		builder.AddData(pk)
	}

	ops, err := hex.DecodeString(redeemScriptWithoutConstructorArgs)
	if err != nil {
		return "", fmt.Errorf("failed to decode redeemScriptWithoutConstructorArgs")
	}
	builder.AddOps(ops)

	redeemScript, err := builder.Script()
	if err != nil {
		return "", err
	}
	//println("redeemScript:", hex.EncodeToString(redeemScript))

	// calculate the hash160 of the redeem script
	redeemHash := bchutil.Hash160(redeemScript)
	//println("redeemScriptHash160:", hex.EncodeToString(redeemHash))

	// if using Bitcoin main net then pass &chaincfg.MainNetParams as second argument
	addr, err := bchutil.NewAddressScriptHashFromHash(redeemHash, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}

	return addr.EncodeAddress(), nil
}
