package types

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/holiman/uint256"
	"github.com/smartbch/moeingdb/types"
	"strings"

	cctypes "github.com/smartbch/smartbch/crosschain/types"
)

// p2pkh lock script: 76 + a9(OP_HASH160) + 14 + 20-byte-length-pubkey-hash + 88(OP_EQUALVERIFY) + ac(OP_CHECKSIG)
// p2sh lock script:  a9(OP_HASH160) + 14 + 20-byte-redeem-script-hash + 87(OP_EQUAL)
// cc related op return: 6a(OP_RETURN) + 1c(8 + 20) + 7342434841646472(sBCHAddr) + 20-byte-side-address

var ccIdentifier = "sBCHAddr"

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

func (cc *CcTxParser) GetCCUTXOTransferInfo(bi *BlockInfo) (infos []*cctypes.CCTransferInfo) {
	cc.refresh(bi.Height)
	infos = append(infos, cc.findRedeemableTx(bi.Tx)...)
	infos = append(infos, cc.findConvertTx(bi.Tx)...)
	infos = append(infos, cc.findRedeemOrLostAndFoundTx(bi.Tx)...)
	return
}

type CcTxParser struct {
	DB                     types.DB
	CurrentCovenantAddress string
	PrevCovenantAddress    string
	UtxoSet                map[[32]byte]uint32
}

func (cc *CcTxParser) refresh(height int64) {
	var outpointSet map[[32]byte]uint32 /*txid => vout*/
	//todo: replace with other design
	for _, id := range cc.DB.GetAllUtxoIds() {
		var txid [32]byte
		copy(txid[:], id[:32])
		index := binary.BigEndian.Uint32(id[32:])
		outpointSet[txid] = index
	}
	cc.getParams(height)
}

func (cc *CcTxParser) getParams(height int64) {
	//todo: need modb support
	// db.getCurrentCovenantAddress(mainBlockHeight int64)
	// db.getPrevCovenantAddress(mainBlockHeight int64)
}

func (cc *CcTxParser) findRedeemableTx(txs []TxInfo) (infos []*cctypes.CCTransferInfo) {
	for _, ti := range txs {
		var isRedeemableTx bool
		var info = cctypes.CCTransferInfo{
			Type: cctypes.TransferType,
		}
		for n, vOut := range ti.VoutList {
			script, ok := getPubkeyScript(vOut)
			if !ok {
				continue
			}
			if script == "OP_HASH160 "+cc.CurrentCovenantAddress+" OP_EQUAL" {
				info.UTXO.Amount = uint256.NewInt(0).Mul(uint256.NewInt(uint64(vOut.Value)), uint256.NewInt(10e8)).Bytes32()
				copy(info.UTXO.TxID[:], ti.Hash)
				info.UTXO.Index = uint32(n)
				isRedeemableTx = true
				break
			}
		}
		if isRedeemableTx {
			receiver := findReceiver(ti)
			if receiver != nil {
				copy(info.Receiver[:], receiver)
				infos = append(infos, &info)
			}
		}
	}
	return
}

func (cc *CcTxParser) findConvertTx(txs []TxInfo) (infos []*cctypes.CCTransferInfo) {
	for _, ti := range txs {
		var maybeConvertTx bool
		var info = cctypes.CCTransferInfo{
			Type: cctypes.ConvertType,
		}
		for n, vOut := range ti.VoutList {
			script, ok := getPubkeyScript(vOut)
			if !ok {
				continue
			}
			if script == "OP_HASH160 "+cc.CurrentCovenantAddress+" OP_EQUAL" {
				info.UTXO.Amount = uint256.NewInt(0).Mul(uint256.NewInt(uint64(vOut.Value)), uint256.NewInt(10e8)).Bytes32()
				copy(info.UTXO.TxID[:], ti.Hash)
				info.UTXO.Index = uint32(n)
				maybeConvertTx = true
				break
			}
		}
		if maybeConvertTx {
			for _, vIn := range ti.VinList {
				txid, vout, err := getSpentTxInfo(vIn)
				if err != nil {
					fmt.Println(err)
					continue
				}
				if cc.isCcUXTOSpent(txid, vout) {
					copy(info.PrevUTXO.TxID[:], txid[:])
					info.PrevUTXO.Index = vout
					infos = append(infos, &info)
					break
				}
			}
		}
	}
	return
}

func (cc *CcTxParser) findRedeemOrLostAndFoundTx(txs []TxInfo) (infos []*cctypes.CCTransferInfo) {
	for _, ti := range txs {
		var maybeTargetTx bool
		var info = cctypes.CCTransferInfo{
			Type: cctypes.RedeemOrLostAndFoundType,
		}
		for _, vIn := range ti.VinList {
			txid, vout, err := getSpentTxInfo(vIn)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if cc.isCcUXTOSpent(txid, vout) {
				copy(info.PrevUTXO.TxID[:], txid[:])
				info.PrevUTXO.Index = vout
				maybeTargetTx = true
				break
			}
		}
		if maybeTargetTx {
			if len(ti.VoutList) != 1 {
				continue
			}
			vOut := ti.VoutList[0]
			script, ok := getPubkeyScript(vOut)
			if !ok {
				continue
			}
			// only check prefix
			if strings.HasPrefix(script, "OP_DUP OP_HASH160") {
				infos = append(infos, &info)
				continue
			}
		}
	}
	return
}

func (cc *CcTxParser) isCcUXTOSpent(txid [32]byte, vout uint32) bool {
	index, ok := cc.UtxoSet[txid]
	if ok {
		return index == vout
	}
	return false
}

//util functions
func getSpentTxInfo(vIn map[string]interface{}) (txid [32]byte, index uint32, err error) {
	txidV, exist := vIn["txid"]
	if !exist || txidV == nil {
		return [32]byte{}, 0, errors.New("no txid")
	}
	txidString, ok := txidV.(string)
	if !ok {
		return [32]byte{}, 0, errors.New("txid not string")
	}
	id, err := hex.DecodeString(txidString)
	if err != nil {
		return [32]byte{}, 0, errors.New("not hex string")
	}
	if len(id) != 32 {
		return [32]byte{}, 0, errors.New("txid bytes length incorrect")
	}
	copy(txid[:], id)
	vout, exist := vIn["vout"]
	if !exist || vout == nil {
		return [32]byte{}, 0, errors.New("no vout")
	}
	index, ok = vout.(uint32)
	if !ok {
		return [32]byte{}, 0, errors.New("not uint32")
	}
	return
}

func findReceiver(tx TxInfo) []byte {
	for _, vOut := range tx.VoutList {
		script, ok := getPubkeyScript(vOut)
		if !ok {
			continue
		}
		receiver, exist := findReceiverInOPReturn(script)
		if exist {
			return receiver
		}
	}
	for _, vIn := range tx.VinList {
		receiver, exist := getP2PKHAddress(vIn)
		if exist {
			return receiver
		}
	}
	return nil
}

func getPubkeyScript(v Vout) (script string, ok bool) {
	asm, done := v.ScriptPubKey["asm"]
	if !done || asm == nil {
		return "", false
	}
	script, ok = asm.(string)
	return
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

func getP2PKHAddress(vIn map[string]interface{}) ([]byte, bool) {
	script, exist := vIn["scriptSig"]
	if !exist || script == nil {
		return nil, false
	}
	scriptSig, ok := script.(ScriptSig)
	if !ok {
		return nil, false
	}
	if len(scriptSig.Hex) == 25 && strings.HasPrefix(scriptSig.Hex, "76a914") &&
		scriptSig.Hex[23] == 0x88 && scriptSig.Hex[24] == 0xac {
		return []byte(scriptSig.Hex[3:23]), true
	}
	return nil, false
}
