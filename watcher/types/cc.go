package types

import (
	"encoding/hex"
	"strings"

	cctypes "github.com/smartbch/smartbch/crosschain/types"
)

// p2pkh lock script: 76 + a9(OP_HASH160) + 14 + 20-byte-length-pubkey-hash + 88(OP_EQUALVERIFY) + ac(OP_CHECKSIG)
// p2sh lock script:  a9(OP_HASH160) + 14 + 20-byte-redeem-script-hash + 87(OP_EQUAL)
// cc related op return: 6a(OP_RETURN) + 1c(8 + 20) + 7342434841646472(sBCHAddr) + 20-byte-side-address

var prevRedeemScriptAddr = [20]byte{1}
var ccIdentifier = "sBCHAddr"
var currentRedeemScriptAddr = "current_redeem_address"
var burnAddressLockScript = "76a91404df9d9fede348a5f82337ce87a829be2200aed688ac" //burn address 04df9d9fede348a5f82337ce87a829be2200aed6

type ScriptSig struct {
	Asm string `json:"asm"`
	Hex string `json:"hex"`
}

func (ti TxInfo) GetNewCCUTXOTransferInfo() *cctypes.CCTransferInfo {
	var info cctypes.CCTransferInfo
	hasReceiver := false
	for n, vOut := range ti.VoutList {
		asm, ok := vOut.ScriptPubKey["asm"]
		if !ok || asm == nil {
			continue
		}
		script, ok := asm.(string)
		if !ok {
			continue
		}
		if script == "OP_HASH160 "+currentRedeemScriptAddr+" OP_EQUAL" {
			info.UTXO.Amount = int64(vOut.Value * (10e8))
			copy(info.UTXO.TxID[:], ti.Hash)
			info.UTXO.Index = uint32(n)
		} else {
			prefix := "OP_RETURN " + ccIdentifier
			if !strings.HasPrefix(script, prefix) {
				continue
			}
			script = script[len(prefix):]
			if len(script) != 40 {
				continue
			}
			bz, err := hex.DecodeString(script)
			if err != nil {
				continue
			}
			copy(info.Receiver[:], bz)
			hasReceiver = true
		}
	}
	if hasReceiver {
		return &info
	}
	if info.UTXO.Amount != 0 {
		for _, vIn := range ti.VinList {
			script, exist := vIn["scriptSig"]
			if !exist || script == nil {
				continue
			}
			scriptSig, ok := script.(ScriptSig)
			if !ok {
				continue
			}
			if len(scriptSig.Hex) == 25 && strings.HasPrefix(scriptSig.Hex, "76a914") &&
				scriptSig.Hex[23] == 0x88 && scriptSig.Hex[24] == 0xa9 {
				copy(info.Receiver[:], scriptSig.Hex[3:23])
				info.Type = cctypes.TransferType
				return &info
			}
		}
	}
	return nil
}

/*
	type Vin struct {
		Coinbase  string     `json:"coinbase"`
		Txid      string     `json:"txid"`
		Vout      uint32     `json:"vout"`
		ScriptSig *ScriptSig `json:"scriptSig"`
		Sequence  uint32     `json:"sequence"`
	}
*/
func (ti TxInfo) GetConvertUTXOTransferInfo(outpointSet map[string]uint32) *cctypes.CCTransferInfo {
	var info cctypes.CCTransferInfo
	outputHasCurrentRedeemScript := false
	for n, vOut := range ti.VoutList {
		asm, ok := vOut.ScriptPubKey["asm"]
		if !ok || asm == nil {
			continue
		}
		script, ok := asm.(string)
		if !ok {
			continue
		}
		if script == "OP_HASH160 "+currentRedeemScriptAddr+" OP_EQUAL" {
			info.UTXO.Amount = int64(vOut.Value * (10e8))
			copy(info.UTXO.TxID[:], ti.Hash)
			info.UTXO.Index = uint32(n)
			outputHasCurrentRedeemScript = true
			break
		}
	}
	if outputHasCurrentRedeemScript {
		for _, vIn := range ti.VinList {
			txid, exist := vIn["txid"]
			if !exist || txid == nil {
				continue
			}
			txidString, ok := txid.(string)
			if !ok {
				continue
			}
			if index, ok := outpointSet[txidString]; ok {
				vout, exist := vIn["vout"]
				if !exist || vout == nil {
					continue
				}
				v, ok := vout.(uint32)
				if !ok {
					continue
				}
				if index == v {
					copy(info.PrevUTXO.TxID[:], ti.Hash)
					info.PrevUTXO.Index = v
					info.Type = cctypes.ConvertType
					return &info
				}
			}
		}
	}
	return nil
}
