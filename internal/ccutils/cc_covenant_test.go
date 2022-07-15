package ccutils

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/txscript"

	"github.com/smartbch/smartbch/internal/testutils"
)

func Test_GetP2SHAddr(t *testing.T) {
	redeemScriptWithoutConstructorArgs := testutils.HexToBytes("5279009c63557957797e58797ea98800537a547a52567a577a587a53afc3519dc4519d537a6300cd0376a91454797e0288ac7e886700cd02a91454797e01877e88686d7551677b519d547956797e57797ea97b8800727c52557a567a577a53af00c600cc9d00cd02a914537a7e01877e885601189502b40095b26d5168")
	operatorPks := [][]byte{
		testutils.HexToBytes("02d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6994"),
		testutils.HexToBytes("035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd8"),
		testutils.HexToBytes("03fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e4"),
	}
	monitorPks := [][]byte{
		testutils.HexToBytes("024a899d685daf6b1999a5c8f2fd3c9ed640d58e92fd0e00cf87cacee8ff1504b8"),
		testutils.HexToBytes("0374ac9ab3415253dbb7e29f46a69a3e51b5d2d66f125b0c9f2dc990b1d2e87e17"),
		testutils.HexToBytes("024cc911ba9d2c7806a217774618b7ba4848ccd33fe664414fc3144d144cdebf7b"),
	}

	c, err := NewCcCovenant(redeemScriptWithoutConstructorArgs, operatorPks, monitorPks, &chaincfg.TestNet3Params)
	require.NoError(t, err)
	addr, err := c.GetP2SHAddress()
	require.Equal(t, "prx037ejft4me86n5lajsyu284crh8nq6qlqjscazv", addr)
}

func Test_RedeemTx(t *testing.T) {
	redeemScriptWithoutConstructorArgs := testutils.HexToBytes("5279009c63557957797e58797ea98800537a547a52567a577a587a53afc3519dc4519d537a6300cd0376a91454797e0288ac7e886700cd02a91454797e01877e88686d7551677b519d547956797e57797ea97b8800727c52557a567a577a53af00c600cc9d00cd02a914537a7e01877e885601189502b40095b26d5168")
	operatorPks := [][]byte{
		testutils.HexToBytes("02d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6994"),
		testutils.HexToBytes("035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd8"),
		testutils.HexToBytes("03fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e4"),
	}
	monitorPks := [][]byte{
		testutils.HexToBytes("024a899d685daf6b1999a5c8f2fd3c9ed640d58e92fd0e00cf87cacee8ff1504b8"),
		testutils.HexToBytes("0374ac9ab3415253dbb7e29f46a69a3e51b5d2d66f125b0c9f2dc990b1d2e87e17"),
		testutils.HexToBytes("024cc911ba9d2c7806a217774618b7ba4848ccd33fe664414fc3144d144cdebf7b"),
	}

	c, err := NewCcCovenant(redeemScriptWithoutConstructorArgs, operatorPks, monitorPks, &chaincfg.TestNet3Params)
	require.NoError(t, err)

	txid := "7ff88192c5a5ee27237880230b4a9fc0c7e97d7dfe979831b23cd104d46160ee"
	vout := uint32(0)
	prevOutAmt := int64(10000)
	toAddr := "bchtest:qzvu0c953gzu6cpykgkdfy8uacc255wcvgmp7ekj7y"
	outAmt := int64(5000)

	unsignedTx, err := c.BuildUnsignedRedeemTx(txid, vout, toAddr, outAmt)
	require.NoError(t, err)

	sigHash, err := c.GetRedeemTxSigHash(txid, vout, prevOutAmt, toAddr, outAmt)
	require.NoError(t, err)

	hashType := txscript.SigHashAll | txscript.SigHashForkID
	sig0, err := SignCcCovenantTxSigHashECDSA("L482yD31EhZopxRD3V19QEANQaYkcUZfgNKYY2TV4RTCXa6izAKo", sigHash, hashType)
	require.NoError(t, err)
	//println(hex.EncodeToString(sig0))
	sig1, err := SignCcCovenantTxSigHashECDSA("L4JzvBMUmkQCTdz2zbVgTyW8dDMvMU8HFwe413qfnBxW3vKSw6sm", sigHash, hashType)
	require.NoError(t, err)
	//println(hex.EncodeToString(sig1))

	pkh := testutils.HexToBytes("99c7e0b48a05cd6024b22cd490fcee30aa51d862")
	rawTx, err := c.FinishRedeemTx(unsignedTx, [][]byte{sig0, sig1}, pkh)
	require.NoError(t, err)
	//println("rawTx:", rawTx)

	expectedTxHex := "0200000001ee6061d404d13cb2319897fe7d7de9c7c09f4a0b2380782327eea5c59281f87f00000000fdb701511499c7e0b48a05cd6024b22cd490fcee30aa51d8622103fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e421035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd82102d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd699447304402202498436ae5bcf58a5558b5f8af27e5e1ef1382e30a678a767117bdfdb8ae8de002205a598e02ebe7d77fecff8c4775e527b7c9d12cf2b3d34408d87ac6751a18b66a414830450221008ed564033f39ebee5630deab96743eb30d910575aae8166b5528e30d33ef58b4022053a0bc2e6770ece16c64d31c77b9b281af034f096269486a57241db86d8abfe341004ca71427c4ca4766591e6bb8cd71b83143946c53eaf9a3143165b76d51545182cb1e605c288f804da4852f6d5279009c63557957797e58797ea98800537a547a52567a577a587a53afc3519dc4519d537a6300cd0376a91454797e0288ac7e886700cd02a91454797e01877e88686d7551677b519d547956797e57797ea97b8800727c52557a567a577a53af00c600cc9d00cd02a914537a7e01877e885601189502b40095b26d5168ffffffff0188130000000000001976a91499c7e0b48a05cd6024b22cd490fcee30aa51d86288ac00000000"
	require.Equal(t, expectedTxHex, rawTx)
}
