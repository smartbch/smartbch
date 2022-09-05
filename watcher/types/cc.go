package types

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gcash/bchd/txscript"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchutil"
	"github.com/holiman/uint256"
	"github.com/smartbch/moeingdb/types"

	cctypes "github.com/smartbch/smartbch/crosschain/types"
)

// p2pkh lock script: 76 + a9(OP_HASH160) + 14 + 20-byte-length-pubkey-hash + 88(OP_EQUALVERIFY) + ac(OP_CHECKSIG)
// p2sh lock script:  a9(OP_HASH160) + 14 + 20-byte-redeem-script-hash + 87(OP_EQUAL)
// cc related op return: 6a(OP_RETURN) + 1c(8 + 20) + 7342434841646472(sBCHAddr) + 20-byte-side-address

const ccIdentifier = "7342434841646472" // hex("sBCHAddr")

//type ScriptSig struct {
//	Asm string `json:"asm"`
//	Hex string `json:"hex"`
//}

/*
	type Vin struct {
		Coinbase  string     `json:"coinbase"`
		Txid      string     `json:"txid"`
		Vout      uint32     `json:"vout"`
		ScriptSig *ScriptSig `json:"scriptSig"`
		Sequence  uint32     `json:"sequence"`
	}
*/

type CcTxParser struct {
	DB                     types.DB
	CurrentCovenantAddress string
	PrevCovenantAddress    string
	UtxoSet                map[[32]byte]uint32
}

func (cc *CcTxParser) GetCCUTXOTransferInfo(bi *BlockInfo) (infos []*cctypes.CCTransferInfo) {
	infos = append(infos, cc.findRedeemableTx(bi.Tx)...)
	infos = append(infos, cc.findConvertTx(bi.Tx)...)
	infos = append(infos, cc.findRedeemOrLostAndFoundTx(bi.Tx)...)
	return
}

func (cc *CcTxParser) Refresh(prevCovenantAddr, currCovenantAddr common.Address) {
	var outpointSet = make(map[[32]byte]uint32 /*txid => vout*/)
	for _, id := range cc.DB.GetAllUtxoIds() {
		var txid [32]byte
		copy(txid[:], id[:32])
		index := binary.BigEndian.Uint32(id[32:])
		outpointSet[txid] = index
	}
	cc.UtxoSet = outpointSet
	cc.PrevCovenantAddress = prevCovenantAddr.String()
	cc.CurrentCovenantAddress = currCovenantAddr.String()
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
				info.UTXO.Amount = uint256.NewInt(0).Mul(uint256.NewInt(uint64(vOut.Value*1e8)), uint256.NewInt(1e10)).Bytes32()
				info.UTXO.TxID = common.HexToHash(ti.Hash)
				info.UTXO.Index = uint32(n)
				info.CovenantAddress = common.HexToAddress(cc.CurrentCovenantAddress)
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
		if len(ti.VoutList) != 1 || len(ti.VinList) != 1 {
			continue
		}
		var maybeConvertTx bool
		var info = cctypes.CCTransferInfo{
			Type: cctypes.ConvertType,
		}
		vOut := ti.VoutList[0]
		script, ok := getPubkeyScript(vOut)
		if !ok {
			continue
		}
		if script == "OP_HASH160 "+cc.CurrentCovenantAddress+" OP_EQUAL" {
			info.UTXO.Amount = uint256.NewInt(0).Mul(uint256.NewInt(uint64(vOut.Value*1e8)), uint256.NewInt(1e10)).Bytes32()
			info.UTXO.TxID = common.HexToHash(ti.Hash)
			info.UTXO.Index = uint32(0)
			info.CovenantAddress = common.HexToAddress(cc.CurrentCovenantAddress)
			maybeConvertTx = true
			//break
		}
		if maybeConvertTx {
			txid, vout, err := getSpentTxInfo(ti.VinList[0])
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
	return
}

func (cc *CcTxParser) findRedeemOrLostAndFoundTx(txs []TxInfo) (infos []*cctypes.CCTransferInfo) {
	for _, ti := range txs {
		if len(ti.VoutList) != 1 || len(ti.VinList) != 1 {
			continue
		}
		var maybeTargetTx bool
		var info = cctypes.CCTransferInfo{
			Type: cctypes.RedeemOrLostAndFoundType,
		}
		txid, vout, err := getSpentTxInfo(ti.VinList[0])
		if err != nil {
			fmt.Println(err)
			continue
		}
		if cc.isCcUXTOSpent(txid, vout) {
			copy(info.PrevUTXO.TxID[:], txid[:])
			info.PrevUTXO.Index = vout
			maybeTargetTx = true
			//break
		}
		if maybeTargetTx {
			script, ok := getPubkeyScript(ti.VoutList[0])
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
	nVout, ok := vout.(float64)
	if !ok {
		return [32]byte{}, 0, errors.New("not float64")
	}
	index = uint32(nVout)
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

// todo: not support Schnorr Signature now
func getP2PKHAddress(vIn map[string]interface{}) ([]byte, bool) {
	script, exist := vIn["scriptSig"]
	if !exist || script == nil {
		return nil, false
	}
	scriptSig, ok := script.(map[string]interface{})
	if !ok {
		return nil, false
	}
	scriptSigHex, ok := scriptSig["hex"].(string)
	if !ok {
		return nil, false
	}
	bs, err := hex.DecodeString(scriptSigHex)
	if err != nil {
		return nil, false
	}
	//https://github.com/gcash/bchd/blob/master/txscript/engine.go#L580
	minSigLen := 8
	//https://github.com/gcash/bchd/blob/master/txscript/engine.go#L588
	maxSigLen := 72
	minP2PKHSigScriptLen := 1 + minSigLen + 1 + 33
	maxP2PKHSigScriptLen := 1 + maxSigLen + 1 + 65
	if len(bs) < minP2PKHSigScriptLen || len(bs) > maxP2PKHSigScriptLen {
		return nil, false
	}
	sigLen := bs[0]
	if int(sigLen) < minSigLen || int(sigLen) > maxSigLen {
		return nil, false
	}
	leftLength := len(bs) - 1 - int(sigLen)
	if leftLength != 33+1 && leftLength != 65+1 {
		return nil, false
	}
	pubkeyLengthPos := sigLen + 1
	if bs[pubkeyLengthPos] != 33 && bs[pubkeyLengthPos] != 65 {
		return nil, false
	}
	// remove push op
	bs = bs[1:]
	_, err = bchec.ParseDERSignature(bs[:sigLen], bchec.S256())
	if err != nil {
		return nil, false
	}
	// remove sig
	bs = bs[sigLen:]
	// remove push op code
	bs = bs[1:]
	pubkey, err := bchutil.NewAddressPubKey(bs, &chaincfg.MainNetParams)
	if err != nil {
		return nil, false
	}
	return crypto.PubkeyToAddress(*pubkey.PubKey().ToECDSA()).Bytes(), true
}

func getP2PKHAddressNew(vIn map[string]interface{}) ([]byte, bool) {
	script, exist := vIn["scriptSig"]
	if !exist || script == nil {
		return nil, false
	}
	scriptSig, ok := script.(map[string]interface{})
	if !ok {
		return nil, false
	}
	scriptSigHex, ok := scriptSig["hex"].(string)
	if !ok {
		return nil, false
	}
	bs, err := hex.DecodeString(scriptSigHex)
	if err != nil {
		return nil, false
	}
	//https://github.com/gcash/bchd/blob/master/txscript/engine.go#L580
	minSigLen := 8
	//https://github.com/gcash/bchd/blob/master/txscript/engine.go#L588
	maxSigLen := 72
	minP2PKHSigScriptLen := 1 + minSigLen + 1 + 33
	maxP2PKHSigScriptLen := 1 + maxSigLen + 1 + 65
	// Schnorr Signature length is 64byte, in the [minSigLen, maxSigLen]
	if len(bs) < minP2PKHSigScriptLen || len(bs) > maxP2PKHSigScriptLen {
		return nil, false
	}
	elements, err := txscript.ExtractDataElements(bs)
	if err != nil {
		return nil, false
	}
	if len(elements) != 2 {
		return nil, false
	}
	pubkeyBytes := elements[1]
	if len(pubkeyBytes) != 65 && len(pubkeyBytes) != 33 {
		return nil, false
	}
	pubkey, err := bchutil.NewAddressPubKey(pubkeyBytes, &chaincfg.MainNetParams)
	if err != nil {
		return nil, false
	}
	return crypto.PubkeyToAddress(*pubkey.PubKey().ToECDSA()).Bytes(), true
}
