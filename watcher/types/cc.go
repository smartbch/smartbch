package types

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/holiman/uint256"
	"strings"

	cctypes "github.com/smartbch/smartbch/crosschain/types"
)

// p2pkh lock script: 76 + a9(OP_HASH160) + 14 + 20-byte-length-pubkey-hash + 88(OP_EQUALVERIFY) + ac(OP_CHECKSIG)
// p2sh lock script:  a9(OP_HASH160) + 14 + 20-byte-redeem-script-hash + 87(OP_EQUAL)
// cc related op return: 6a(OP_RETURN) + 1c(8 + 20) + 7342434841646472(sBCHAddr) + 20-byte-side-address

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

// GetCCUTXOTransferInfo todo: refactor
func (bi *BlockInfo) GetCCUTXOTransferInfo(ids [][36]byte) []*cctypes.CCTransferInfo {
	var outpointSet map[[32]byte]uint32 /*txid => vout*/
	for _, id := range ids {
		var txid [32]byte
		copy(txid[:], id[:32])
		index := binary.BigEndian.Uint32(id[32:])
		outpointSet[txid] = index
	}
	var infos []*cctypes.CCTransferInfo
	hasReceiver := false
	hasCurrentRedeemAddressInOutput := false
	hasCCUTXOInInput := false
	for _, ti := range bi.Tx {
		var info cctypes.CCTransferInfo
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
					info.UTXO.Amount = uint256.NewInt(0).Mul(uint256.NewInt(uint64(vOut.Value)), uint256.NewInt(10e8)).Bytes32()
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
			infos = append(infos, &info)
			continue
		}
		if hasCCUTXOInInput {
			info.Type = cctypes.RedeemOrLostAndFoundType
			//todo: find the receiver
			infos = append(infos, &info)
			continue
		}
		if hasCurrentRedeemAddressInOutput && hasReceiver {
			info.Type = cctypes.TransferType
		}
	}
	return infos
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
