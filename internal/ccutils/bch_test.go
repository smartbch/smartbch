package ccutils

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gcash/bchd/txscript"

	"github.com/smartbch/smartbch/internal/testutils"
)

func Test_CcCovenant_GetP2SHAddr(t *testing.T) {
	redeemScriptWithoutConstructorArgs := "5279009c63557957797e58797ea98800537a547a52567a577a587a53afc3519dc4519d537a6300cd0376a91454797e0288ac7e886700cd02a91454797e01877e88686d7551677b519d547956797e57797ea97b8800727c52557a567a577a53af00c600cc9d00cd02a914537a7e01877e885601189502b40095b26d5168"
	operatorPks := []string{
		"02d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6994",
		"035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd8",
		"03fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e4",
	}
	monitorPks := []string{
		"024a899d685daf6b1999a5c8f2fd3c9ed640d58e92fd0e00cf87cacee8ff1504b8",
		"0374ac9ab3415253dbb7e29f46a69a3e51b5d2d66f125b0c9f2dc990b1d2e87e17",
		"024cc911ba9d2c7806a217774618b7ba4848ccd33fe664414fc3144d144cdebf7b",
	}

	addr, err := GetCcCovenantP2SHAddr(redeemScriptWithoutConstructorArgs, operatorPks, monitorPks)
	require.NoError(t, err)
	require.Equal(t, "prx037ejft4me86n5lajsyu284crh8nq6qlqjscazv", addr)
}

func Test_CcCovenant_RedeemTx(t *testing.T) {
	txid := "7ff88192c5a5ee27237880230b4a9fc0c7e97d7dfe979831b23cd104d46160ee"
	vout := uint32(0)
	toAddr := "bchtest:qzvu0c953gzu6cpykgkdfy8uacc255wcvgmp7ekj7y"
	outAmt := int64(5000)

	tx, err := MakeCcCovenantUnsignedRedeemTx(txid, vout, toAddr, outAmt)
	require.NoError(t, err)
	println(testutils.ToJSON(tx))

	inputIdx := 0
	redeemScript := testutils.HexToBytes("1427c4ca4766591e6bb8cd71b83143946c53eaf9a3143165b76d51545182cb1e605c288f804da4852f6d5279009c63557957797e58797ea98800537a547a52567a577a587a53afc3519dc4519d537a6300cd0376a91454797e0288ac7e886700cd02a91454797e01877e88686d7551677b519d547956797e57797ea97b8800727c52557a567a577a53af00c600cc9d00cd02a914537a7e01877e885601189502b40095b26d5168")
	hashType := txscript.SigHashAll | txscript.SigHashForkID
	prevOutAmt := int64(10000)
	sigHash, err := GetSigHash(tx, inputIdx, redeemScript, hashType, prevOutAmt)
	require.NoError(t, err)
	println(hex.EncodeToString(sigHash))

	pk0 := testutils.HexToBytes("02d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6994")
	pk1 := testutils.HexToBytes("035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd8")
	pk2 := testutils.HexToBytes("03fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e4")

	sig0, err := SignCcCovenantTxSigHashECDSA("L482yD31EhZopxRD3V19QEANQaYkcUZfgNKYY2TV4RTCXa6izAKo", sigHash, hashType)
	require.NoError(t, err)
	println(hex.EncodeToString(sig0))
	sig1, err := SignCcCovenantTxSigHashECDSA("L4JzvBMUmkQCTdz2zbVgTyW8dDMvMU8HFwe413qfnBxW3vKSw6sm", sigHash, hashType)
	require.NoError(t, err)
	println(hex.EncodeToString(sig1))

	pkh := testutils.HexToBytes("99c7e0b48a05cd6024b22cd490fcee30aa51d862")
	rawTx, err := FixCcCovenantUnsignedRedeemTx(tx, redeemScript, [][]byte{pk0, pk1, pk2}, [][]byte{sig0, sig1}, pkh)
	require.NoError(t, err)
	println("rawTx:", rawTx)
}
