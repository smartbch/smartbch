package types

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	cctypes "github.com/smartbch/smartbch/crosschain/types"
)

type UtxoForTest struct {
	TxID   string
	Index  uint32
	Amount string
}
type CCTransferInfoForTest struct {
	Type            cctypes.UTXOType
	PrevUTXO        UtxoForTest
	UTXO            UtxoForTest
	Receiver        string
	CovenantAddress string
}

func ccTransferInfoToJSON(info *cctypes.CCTransferInfo) string {
	bz, _ := json.MarshalIndent(newCCTransferInfoForTest(info), "", "  ")
	return string(bz)
}
func newCCTransferInfoForTest(info *cctypes.CCTransferInfo) CCTransferInfoForTest {
	return CCTransferInfoForTest{
		Type: info.Type,
		PrevUTXO: UtxoForTest{
			TxID:   hex.EncodeToString(info.PrevUTXO.TxID[:]),
			Index:  info.PrevUTXO.Index,
			Amount: uint256.NewInt(0).SetBytes32(info.PrevUTXO.Amount[:]).String(),
		},
		UTXO: UtxoForTest{
			TxID:   hex.EncodeToString(info.UTXO.TxID[:]),
			Index:  info.UTXO.Index,
			Amount: uint256.NewInt(0).SetBytes32(info.UTXO.Amount[:]).String(),
		},
		Receiver:        hex.EncodeToString(info.Receiver[:]),
		CovenantAddress: hex.EncodeToString(info.CovenantAddress[:]),
	}
}

func TestGetAddrFromOpReturn(t *testing.T) {
	// https://www.blockchain.com/bch-testnet/block/1508978
	blockJson := `
 {
    "hash": "00000000000000c8d02f76b19ee228ff14eefc1fd00ff85d9837c023da232503",
    "confirmations": 6911,
    "strippedsize": 0,
    "size": 484,
    "height": 1508978,
    "version": 549453824,
    "versionHex": "20c00000",
    "merkleroot": "90ccadacbfd7d90107e31acb21d43c7ec4e2d5fd80472a527698dc79901a9e96",
    "tx": [
      {
        "hex": "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff0403720617ffffffff020d513602000000001976a914f60e91e018a0f963a21129aa7427357b1653d17288ac00000000000000002a6a288ab89e331cb2e163133de3c1c4a016f8655cac4ca1fb363a9a823cb741e243f89c0b00002500000000000000",
        "hash": "80de78e76bc26b901d9d1156b3f0369f350170117ea005421dd8723a2dd46333",
        "size": 140,
        "version": 1,
        "locktime": 0,
        "vin": [
          {
            "coinbase": "03720617",
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 0.37114125,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 f60e91e018a0f963a21129aa7427357b1653d172 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a914f60e91e018a0f963a21129aa7427357b1653d17288ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qrmqay0qrzs0jcazzy565ap8x4a3v573wgru39vu0e"
              ]
            }
          },
          {
            "value": 0,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_RETURN 8ab89e331cb2e163133de3c1c4a016f8655cac4ca1fb363a9a823cb741e243f89c0b000025000000",
              "hex": "6a288ab89e331cb2e163133de3c1c4a016f8655cac4ca1fb363a9a823cb741e243f89c0b000025000000",
              "type": "nulldata"
            }
          }
        ],
        "blockhash": "00000000000000c8d02f76b19ee228ff14eefc1fd00ff85d9837c023da232503",
        "confirmations": 6911,
        "time": 1657866426,
        "blocktime": 1657866426
      },
      {
        "hex": "0200000001c3b1d3755fee3d4370833f0cec46e86ab2763a30b93f28d6ca94707a66b6af84000000006b483045022100ac5165cccc65fc104523bee1979c498116f5becdd06614808b41a2f4222ad13b022016d107fe4784a772d7d293281592af59d72fe3c5fe7c6a349b736c107c3b5203412102d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3effffffff03102700000000000017a914ccf8fb324aebbc9f53a7fb28138a3d703b9e60d087084c0100000000001976a91468ccb0e4918444bddb05dccb313d8c979e8e25f288ac00000000000000001e6a1c7342434841646472c370743331b37d3c6d0ee798b3918f6561af2c9200000000",
        "hash": "7ff88192c5a5ee27237880230b4a9fc0c7e97d7dfe979831b23cd104d46160ee",
        "size": 263,
        "version": 2,
        "locktime": 0,
        "vin": [
          {
            "txid": "84afb6667a7094cad6283fb9303a76b26ae846ec0c3f8370433dee5f75d3b1c3",
            "vout": 0,
            "scriptSig": {
              "asm": "3045022100ac5165cccc65fc104523bee1979c498116f5becdd06614808b41a2f4222ad13b022016d107fe4784a772d7d293281592af59d72fe3c5fe7c6a349b736c107c3b520341 02d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3e",
              "hex": "483045022100ac5165cccc65fc104523bee1979c498116f5becdd06614808b41a2f4222ad13b022016d107fe4784a772d7d293281592af59d72fe3c5fe7c6a349b736c107c3b5203412102d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3e"
            },
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 0.0001,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_HASH160 ccf8fb324aebbc9f53a7fb28138a3d703b9e60d0 OP_EQUAL",
              "hex": "a914ccf8fb324aebbc9f53a7fb28138a3d703b9e60d087",
              "reqSigs": 1,
              "type": "scripthash",
              "addresses": [
                "prx037ejft4me86n5lajsyu284crh8nq6qlqjscazv"
              ]
            }
          },
          {
            "value": 0.00085,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 68ccb0e4918444bddb05dccb313d8c979e8e25f2 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a91468ccb0e4918444bddb05dccb313d8c979e8e25f288ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qp5vev8yjxzyf0wmqhwvkvfa3jtear397gwsfxg7sa"
              ]
            }
          },
          {
            "value": 0,
            "n": 2,
            "scriptPubKey": {
              "asm": "OP_RETURN 7342434841646472c370743331b37d3c6d0ee798b3918f6561af2c92",
              "hex": "6a1c7342434841646472c370743331b37d3c6d0ee798b3918f6561af2c92",
              "type": "nulldata"
            }
          }
        ],
        "blockhash": "00000000000000c8d02f76b19ee228ff14eefc1fd00ff85d9837c023da232503",
        "confirmations": 6911,
        "time": 1657866426,
        "blocktime": 1657866426
      }
    ],
    "time": 1657866426,
    "nonce": 1741380489,
    "bits": "1a05c74e",
    "difficulty": 2903324.64724242,
    "previousblockhash": "0000000000000123229171002dc6d67dd34fc6241166624334e343201e480251",
    "nextblockhash": "00000000022b966f1eb246455c7ade9faf1384df39f807096b930a595f6498b5"
  }
`

	var bi BlockInfo
	err := json.Unmarshal([]byte(blockJson), &bi)
	require.NoError(t, err)

	covenantAddr := "ccf8fb324aebbc9f53a7fb28138a3d703b9e60d0"
	parser := &CcTxParser{
		CurrentCovenantAddress: covenantAddr,
	}
	infos := parser.GetCCUTXOTransferInfo(&bi)
	require.Len(t, infos, 1)
	require.Equal(t, `{
  "Type": 0,
  "PrevUTXO": {
    "TxID": "0000000000000000000000000000000000000000000000000000000000000000",
    "Index": 0,
    "Amount": "0x0"
  },
  "UTXO": {
    "TxID": "7ff88192c5a5ee27237880230b4a9fc0c7e97d7dfe979831b23cd104d46160ee",
    "Index": 0,
    "Amount": "0x5af3107a4000"
  },
  "Receiver": "c370743331b37d3c6d0ee798b3918f6561af2c92",
  "CovenantAddress": "ccf8fb324aebbc9f53a7fb28138a3d703b9e60d0"
}`, ccTransferInfoToJSON(infos[0]))
}

func TestGetAddrFromInput(t *testing.T) {
	// https://www.blockchain.com/bch-testnet/block/1508979
	blockJson := `
 {
    "hash": "00000000022b966f1eb246455c7ade9faf1384df39f807096b930a595f6498b5",
    "confirmations": 6912,
    "strippedsize": 0,
    "size": 1167,
    "height": 1508979,
    "version": 549453824,
    "versionHex": "20c00000",
    "merkleroot": "48c2275d5b203d68ec2be17223db6afe61c70f343a3559a2db7fffee19b2bf75",
    "tx": [
      {
        "hex": "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff0403730617ffffffff02a2663602000000001976a914f60e91e018a0f963a21129aa7427357b1653d17288ac00000000000000002a6a28c52e790efe7d634161622626442ba1b1bb868018bb08bfffd23f51b21346b2bb000000000400000000000000",
        "txid": "bc7e77583185222e69cf7df1d7003e66e15257c620bd740c9e05b1fc82bb1754",
        "hash": "bc7e77583185222e69cf7df1d7003e66e15257c620bd740c9e05b1fc82bb1754",
        "size": 140,
        "version": 1,
        "locktime": 0,
        "vin": [
          {
            "coinbase": "03730617",
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 0.3711965,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 f60e91e018a0f963a21129aa7427357b1653d172 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a914f60e91e018a0f963a21129aa7427357b1653d17288ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qrmqay0qrzs0jcazzy565ap8x4a3v573wgru39vu0e"
              ]
            }
          },
          {
            "value": 0,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_RETURN c52e790efe7d634161622626442ba1b1bb868018bb08bfffd23f51b21346b2bb0000000004000000",
              "hex": "6a28c52e790efe7d634161622626442ba1b1bb868018bb08bfffd23f51b21346b2bb0000000004000000",
              "type": "nulldata"
            }
          }
        ],
        "blockhash": "00000000022b966f1eb246455c7ade9faf1384df39f807096b930a595f6498b5",
        "confirmations": 6912,
        "time": 1657867628,
        "blocktime": 1657867628
      },
      {
        "hex": "0100000001e6efb798f66a6b9dce4da30aef52da27ba9035ea9fe1ef2f627a48aea21432b5010000006b483045022100b24b94488d6fb046d3042dd0cc24ee43431d1838736e91146528df25e142dcab0220095b9c14cb5211372a8a37d2c2caff529dddcd0ae655d74c0d93e0c3f3cb140041210210fd6791f811af0013ca17a23294621c311a5b09b546af6fd262b8da8aece2fefeffffff02201d9a000000000017a914665f02b56bca7b4a40772b370e8bc2caece72e358788799553010000001976a91448594b7f204881ef6dd396e3f043fbddd97fbf9b88ac71061700",
        "txid": "09fd6d6595d54441285b31f8d27fc8e22a0c476d0889fd6776fe74f856ec4196",
        "hash": "09fd6d6595d54441285b31f8d27fc8e22a0c476d0889fd6776fe74f856ec4196",
        "size": 224,
        "version": 1,
        "locktime": 1508977,
        "vin": [
          {
            "txid": "b53214a2ae487a622fefe19fea3590ba27da52ef0aa34dce9d6b6af698b7efe6",
            "vout": 1,
            "scriptSig": {
              "asm": "3045022100b24b94488d6fb046d3042dd0cc24ee43431d1838736e91146528df25e142dcab0220095b9c14cb5211372a8a37d2c2caff529dddcd0ae655d74c0d93e0c3f3cb140041 0210fd6791f811af0013ca17a23294621c311a5b09b546af6fd262b8da8aece2fe",
              "hex": "483045022100b24b94488d6fb046d3042dd0cc24ee43431d1838736e91146528df25e142dcab0220095b9c14cb5211372a8a37d2c2caff529dddcd0ae655d74c0d93e0c3f3cb140041210210fd6791f811af0013ca17a23294621c311a5b09b546af6fd262b8da8aece2fe"
            },
            "sequence": 4294967294
          }
        ],
        "vout": [
          {
            "value": 0.101,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_HASH160 665f02b56bca7b4a40772b370e8bc2caece72e35 OP_EQUAL",
              "hex": "a914665f02b56bca7b4a40772b370e8bc2caece72e3587",
              "reqSigs": 1,
              "type": "scripthash",
              "addresses": [
                "ppn97q44d098kjjqwu4nwr5tct9weeewx5s7xjspzu"
              ]
            }
          },
          {
            "value": 56.972722,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 48594b7f204881ef6dd396e3f043fbddd97fbf9b OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a91448594b7f204881ef6dd396e3f043fbddd97fbf9b88ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qpy9jjmlypygrmmd6wtw8uzrl0wajlalnvdusau6u3"
              ]
            }
          }
        ],
        "blockhash": "00000000022b966f1eb246455c7ade9faf1384df39f807096b930a595f6498b5",
        "confirmations": 6912,
        "time": 1657867628,
        "blocktime": 1657867628
      },
      {
        "hex": "0200000001984cf597e8fee35e6ec12dac7646d7e2212a1aed0bb1f7430e15a64d27ffbdb8010000006b4830450221009a80442a8b29bc6015100b3ccc98c4b37fb4a5d1b83f55d41859d7d26cac096d0220084840757a66dca75161d4092b72523725ff22ad9141d179b5346570fc38209d412102d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3effffffff02102700000000000017a914ccf8fb324aebbc9f53a7fb28138a3d703b9e60d087d8d60000000000001976a91468ccb0e4918444bddb05dccb313d8c979e8e25f288ac00000000",
        "txid": "3a9eb8d0a8bb046f4b0e4423a83612803c6a0c8a0731e1e5895be6ad42144781",
        "hash": "3a9eb8d0a8bb046f4b0e4423a83612803c6a0c8a0731e1e5895be6ad42144781",
        "size": 224,
        "version": 2,
        "locktime": 0,
        "vin": [
          {
            "txid": "b8bdff274da6150e43f7b10bed1a2a21e2d74676ac2dc16e5ee3fee897f54c98",
            "vout": 1,
            "scriptSig": {
              "asm": "30450221009a80442a8b29bc6015100b3ccc98c4b37fb4a5d1b83f55d41859d7d26cac096d0220084840757a66dca75161d4092b72523725ff22ad9141d179b5346570fc38209d41 02d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3e",
              "hex": "4830450221009a80442a8b29bc6015100b3ccc98c4b37fb4a5d1b83f55d41859d7d26cac096d0220084840757a66dca75161d4092b72523725ff22ad9141d179b5346570fc38209d412102d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3e"
            },
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 0.0001,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_HASH160 ccf8fb324aebbc9f53a7fb28138a3d703b9e60d0 OP_EQUAL",
              "hex": "a914ccf8fb324aebbc9f53a7fb28138a3d703b9e60d087",
              "reqSigs": 1,
              "type": "scripthash",
              "addresses": [
                "prx037ejft4me86n5lajsyu284crh8nq6qlqjscazv"
              ]
            }
          },
          {
            "value": 0.00055,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 68ccb0e4918444bddb05dccb313d8c979e8e25f2 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a91468ccb0e4918444bddb05dccb313d8c979e8e25f288ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qp5vev8yjxzyf0wmqhwvkvfa3jtear397gwsfxg7sa"
              ]
            }
          }
        ],
        "blockhash": "00000000022b966f1eb246455c7ade9faf1384df39f807096b930a595f6498b5",
        "confirmations": 6912,
        "time": 1657867628,
        "blocktime": 1657867628
      },
      {
        "hex": "0200000001ee6061d404d13cb2319897fe7d7de9c7c09f4a0b2380782327eea5c59281f87f010000006b483045022100ec64c8944857a444617a93627ce12506b6c54cafd076ec9e04864622a6e3935902200440e7acf1ca62144bcad36cf9343f2cd7c8bef220e78f0b131c569a276e9b11412102d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3effffffff03102700000000000017a914ccf8fb324aebbc9f53a7fb28138a3d703b9e60d08770110100000000001976a91468ccb0e4918444bddb05dccb313d8c979e8e25f288ac0000000000000000076a05123456789000000000",
        "txid": "b8bdff274da6150e43f7b10bed1a2a21e2d74676ac2dc16e5ee3fee897f54c98",
        "hash": "b8bdff274da6150e43f7b10bed1a2a21e2d74676ac2dc16e5ee3fee897f54c98",
        "size": 240,
        "version": 2,
        "locktime": 0,
        "vin": [
          {
            "txid": "7ff88192c5a5ee27237880230b4a9fc0c7e97d7dfe979831b23cd104d46160ee",
            "vout": 1,
            "scriptSig": {
              "asm": "3045022100ec64c8944857a444617a93627ce12506b6c54cafd076ec9e04864622a6e3935902200440e7acf1ca62144bcad36cf9343f2cd7c8bef220e78f0b131c569a276e9b1141 02d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3e",
              "hex": "483045022100ec64c8944857a444617a93627ce12506b6c54cafd076ec9e04864622a6e3935902200440e7acf1ca62144bcad36cf9343f2cd7c8bef220e78f0b131c569a276e9b11412102d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3e"
            },
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 0.0001,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_HASH160 ccf8fb324aebbc9f53a7fb28138a3d703b9e60d0 OP_EQUAL",
              "hex": "a914ccf8fb324aebbc9f53a7fb28138a3d703b9e60d087",
              "reqSigs": 1,
              "type": "scripthash",
              "addresses": [
                "prx037ejft4me86n5lajsyu284crh8nq6qlqjscazv"
              ]
            }
          },
          {
            "value": 0.0007,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 68ccb0e4918444bddb05dccb313d8c979e8e25f2 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a91468ccb0e4918444bddb05dccb313d8c979e8e25f288ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qp5vev8yjxzyf0wmqhwvkvfa3jtear397gwsfxg7sa"
              ]
            }
          },
          {
            "value": 0,
            "n": 2,
            "scriptPubKey": {
              "asm": "OP_RETURN 1234567890",
              "hex": "6a051234567890",
              "type": "nulldata"
            }
          }
        ],
        "blockhash": "00000000022b966f1eb246455c7ade9faf1384df39f807096b930a595f6498b5",
        "confirmations": 6912,
        "time": 1657867628,
        "blocktime": 1657867628
      },
      {
        "hex": "01000000019dd229e3932233f4c9628f444697f3a285b833a2c22a8e5decaaef351ebe3377010000008b483045022100fb2d7808ed6b4aefc0931544054ae8bb74edc6993fac9bd5b157ff9a5c9a8a4002202cd2c96743bcdffaf0b46d842af65d86be829963aefdad5838f1e875fcded53c4141045d2cef9f20f0a86f01b80ee380b0da00a1d296f0437b224a0da921ff7bd74ad95c160c30a20115693c71c7500cea26c9ec860cea2adef8dc02ae13a2b5aeb11cffffffff02e8030000000000001976a914c6dd87e5dace4ab6f02bb344e118622b613af82c88acda2e9100000000001976a9148b67633cfd3842d9fdbf51c3fa739a2a3bba5c1f88ac00000000",
        "txid": "f1f43b32faa4b3e89393cd4008238f862219608565e924d010d52c1bf9c0e7ee",
        "hash": "f1f43b32faa4b3e89393cd4008238f862219608565e924d010d52c1bf9c0e7ee",
        "size": 258,
        "version": 1,
        "locktime": 0,
        "vin": [
          {
            "txid": "7733be1e35efaaec5d8e2ac2a233b885a2f39746448f62c9f4332293e329d29d",
            "vout": 1,
            "scriptSig": {
              "asm": "3045022100fb2d7808ed6b4aefc0931544054ae8bb74edc6993fac9bd5b157ff9a5c9a8a4002202cd2c96743bcdffaf0b46d842af65d86be829963aefdad5838f1e875fcded53c41 045d2cef9f20f0a86f01b80ee380b0da00a1d296f0437b224a0da921ff7bd74ad95c160c30a20115693c71c7500cea26c9ec860cea2adef8dc02ae13a2b5aeb11c",
              "hex": "483045022100fb2d7808ed6b4aefc0931544054ae8bb74edc6993fac9bd5b157ff9a5c9a8a4002202cd2c96743bcdffaf0b46d842af65d86be829963aefdad5838f1e875fcded53c4141045d2cef9f20f0a86f01b80ee380b0da00a1d296f0437b224a0da921ff7bd74ad95c160c30a20115693c71c7500cea26c9ec860cea2adef8dc02ae13a2b5aeb11c"
            },
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 1e-05,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 c6dd87e5dace4ab6f02bb344e118622b613af82c OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a914c6dd87e5dace4ab6f02bb344e118622b613af82c88ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qrrdmpl9mt8y4dhs9we5fcgcvg4kzwhc9sqgnznehp"
              ]
            }
          },
          {
            "value": 0.09514714,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 8b67633cfd3842d9fdbf51c3fa739a2a3bba5c1f OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a9148b67633cfd3842d9fdbf51c3fa739a2a3bba5c1f88ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qz9kwceul5uy9k0ahagu87nnng4rhwjuru6tvn9ajc"
              ]
            }
          }
        ],
        "blockhash": "00000000022b966f1eb246455c7ade9faf1384df39f807096b930a595f6498b5",
        "confirmations": 6912,
        "time": 1657867628,
        "blocktime": 1657867628
      }
    ],
    "time": 1657867628,
    "nonce": 3235622197,
    "bits": "1d00ffff",
    "difficulty": 1,
    "previousblockhash": "00000000000000c8d02f76b19ee228ff14eefc1fd00ff85d9837c023da232503",
    "nextblockhash": "00000000000002c6c063ac619f94197b51dc42762169d8efc4f8d0b50e1b7262"
  }
`

	var bi BlockInfo
	err := json.Unmarshal([]byte(blockJson), &bi)
	require.NoError(t, err)

	covenantAddr := "ccf8fb324aebbc9f53a7fb28138a3d703b9e60d0"
	parser := &CcTxParser{
		CurrentCovenantAddress: covenantAddr,
	}
	infos := parser.GetCCUTXOTransferInfo(&bi)
	require.Len(t, infos, 2)
	require.Equal(t, `{
  "Type": 0,
  "PrevUTXO": {
    "TxID": "0000000000000000000000000000000000000000000000000000000000000000",
    "Index": 0,
    "Amount": "0x0"
  },
  "UTXO": {
    "TxID": "3a9eb8d0a8bb046f4b0e4423a83612803c6a0c8a0731e1e5895be6ad42144781",
    "Index": 0,
    "Amount": "0x5af3107a4000"
  },
  "Receiver": "e4dfc77a15490d868f492959fd56a6649bf818f7",
  "CovenantAddress": "ccf8fb324aebbc9f53a7fb28138a3d703b9e60d0"
}`, ccTransferInfoToJSON(infos[0]))
	require.Equal(t, `{
  "Type": 0,
  "PrevUTXO": {
    "TxID": "0000000000000000000000000000000000000000000000000000000000000000",
    "Index": 0,
    "Amount": "0x0"
  },
  "UTXO": {
    "TxID": "b8bdff274da6150e43f7b10bed1a2a21e2d74676ac2dc16e5ee3fee897f54c98",
    "Index": 0,
    "Amount": "0x5af3107a4000"
  },
  "Receiver": "e4dfc77a15490d868f492959fd56a6649bf818f7",
  "CovenantAddress": "ccf8fb324aebbc9f53a7fb28138a3d703b9e60d0"
}`, ccTransferInfoToJSON(infos[1]))
}
