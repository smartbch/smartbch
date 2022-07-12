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

/*
	type Vin struct {
		Coinbase  string     `json:"coinbase"`
		Txid      string     `json:"txid"`
		Vout      uint32     `json:"vout"`
		ScriptSig *ScriptSig `json:"scriptSig"`
		Sequence  uint32     `json:"sequence"`
	}
*/

func (ti TxInfo) GetCCUTXOTransferInfo(outpointSet map[[32]byte]uint32 /*txid => vout*/) *cctypes.CCTransferInfo {
	var info cctypes.CCTransferInfo
	hasReceiver := false
	hasCurrentRedeemAddressInOutput := false
	hasCCUTXOInInput := false

	for n, vOut := range ti.VoutList {
		if hasCurrentRedeemAddressInOutput && hasReceiver {
			break
		}
		asm, ok := vOut.ScriptPubKey["asm"]
		if !ok || asm == nil {
			continue
		}
		script, ok := asm.(string)
		if !ok {
			continue
		}
		if !hasCurrentRedeemAddressInOutput {
			if script == "OP_HASH160 "+currentRedeemScriptAddr+" OP_EQUAL" {
				info.UTXO.Amount = int64(vOut.Value * (10e8))
				copy(info.UTXO.TxID[:], ti.Hash)
				info.UTXO.Index = uint32(n)
				hasCurrentRedeemAddressInOutput = true
				continue
			}
		}
		if !hasReceiver {
			var receiver []byte
			receiver, hasReceiver = findReceiverInOPReturn(script)
			if hasReceiver {
				copy(info.Receiver[:], receiver)
				continue
			}
		}
	}

	for _, vIn := range ti.VinList {
		if !hasReceiver {
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
				hasReceiver = true
				continue
			}
		}
		if !hasCCUTXOInInput {
			txidV, exist := vIn["txid"]
			if !exist || txidV == nil {
				continue
			}
			txidString, ok := txidV.(string)
			if !ok {
				continue
			}
			id, err := hex.DecodeString(txidString)
			if err != nil {
				continue
			}
			if len(id) != 32 {
				continue
			}
			var txid [32]byte
			copy(txid[:], id)
			if index, ok := outpointSet[txid]; ok {
				vout, exist := vIn["vout"]
				if !exist || vout == nil {
					continue
				}
				v, ok := vout.(uint32)
				if !ok {
					continue
				}
				if index == v {
					info.PrevUTXO.TxID = txid
					info.PrevUTXO.Index = v
					hasCCUTXOInInput = true
					break
				}
			}
		}
	}

	// todo: not allow redeem target address is current redeem address
	if hasCCUTXOInInput && hasCurrentRedeemAddressInOutput {
		info.Type = cctypes.ConvertType
		return &info
	}
	if hasCCUTXOInInput {
		info.Type = cctypes.RedeemOrLostAndFoundType
		//todo: find the receiver
		return &info
	}
	if hasCurrentRedeemAddressInOutput && hasReceiver {
		info.Type = cctypes.TransferType
	}
	return nil
}

func findReceiverInOPReturn(script string) ([]byte, bool) {
	prefix := "OP_RETURN " + ccIdentifier
	if !strings.HasPrefix(script, prefix) {
		return nil, false
	}
	script = script[len(prefix):]
	if len(script) != 40 {
		return nil, false
	}
	bz, err := hex.DecodeString(script)
	if err != nil {
		return nil, false
	}
	return bz, true
}
