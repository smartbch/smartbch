package types

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	gethcmn "github.com/ethereum/go-ethereum/common"
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

func TestOpReturnReceiverParse(t *testing.T) {
	address := "c370743331b37d3c6d0ee798b3918f6561af2c92"
	bz := gethcmn.HexToAddress(address)
	receiver := hex.EncodeToString([]byte(address))
	r, ok := findReceiverInOPReturn("OP_RETURN " + receiver)
	require.True(t, ok)
	require.Equal(t, bz.Bytes(), r)

	receiver = hex.EncodeToString([]byte("0x" + address))
	r, ok = findReceiverInOPReturn("OP_RETURN " + receiver)
	require.True(t, ok)
	require.Equal(t, bz.Bytes(), r)

	receiver = hex.EncodeToString([]byte("01" + address))
	_, ok = findReceiverInOPReturn("OP_RETURN " + receiver)
	require.False(t, ok)
}

func TestGetAddrFromOpReturn(t *testing.T) {
	// https://www.blockchain.com/bch-testnet/block/1517179
	blockJson := `{
    "hash": "00000000567d66f18743e1a1607b1f557ea3b5a85eef17c9386c49a471ef4311",
    "confirmations": 4,
    "size": 470,
    "height": 1517179,
    "version": 536870912,
    "versionHex": "20000000",
    "merkleroot": "25b4a0e09b1e42f8e4fe887718d7d5ccfe70c2e870e1a213660688e2819eed66",
    "tx": [
      {
        "txid": "d239cdd0c66d5c132d51f64b5c743b1d0ed6c3cc9af4b2935df87c6bab24170c",
        "hash": "d239cdd0c66d5c132d51f64b5c743b1d0ed6c3cc9af4b2935df87c6bab24170c",
        "version": 1,
        "size": 112,
        "locktime": 0,
        "vin": [
          {
            "coinbase": "037b2617184e4954534f504f554c4f53204b4f4e5354414e54494e4f53",
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 0.390635,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_HASH160 00000009e62dc2a3eafd18b870ed16fff020a69d OP_EQUAL",
              "hex": "a91400000009e62dc2a3eafd18b870ed16fff020a69d87",
              "reqSigs": 1,
              "type": "scripthash",
              "addresses": [
                "bchtest:pqqqqqqfucku9gl2l5vtsu8dzmllqg9xn5xr336tqn"
              ]
            }
          }
        ],
        "hex": "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff1d037b2617184e4954534f504f554c4f53204b4f4e5354414e54494e4f53ffffffff01cc0f54020000000017a91400000009e62dc2a3eafd18b870ed16fff020a69d8700000000"
      },
      {
        "txid": "c01ab2bfa4a7f64cf781e886844de836e7b45f2c6150de380cb891045e8353c9",
        "hash": "c01ab2bfa4a7f64cf781e886844de836e7b45f2c6150de380cb891045e8353c9",
        "version": 2,
        "size": 277,
        "locktime": 0,
        "vin": [
          {
            "txid": "3a9eb8d0a8bb046f4b0e4423a83612803c6a0c8a0731e1e5895be6ad42144781",
            "vout": 1,
            "scriptSig": {
              "asm": "3045022100fcf716f6b6cb75be60c1b4f399facc7bc596fdcb521008cb0ccb6d8045a20f6a0220236859c32ee5f7868e6c97657dc35cfd2143b6ec5c5b62eba7006b0f63cc9b00[ALL|FORKID] 02d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3e",
              "hex": "483045022100fcf716f6b6cb75be60c1b4f399facc7bc596fdcb521008cb0ccb6d8045a20f6a0220236859c32ee5f7868e6c97657dc35cfd2143b6ec5c5b62eba7006b0f63cc9b00412102d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3e"
            },
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 5.001e-05,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_HASH160 6ad3f81523c87aa17f1dfa08271cf57b6277c98e OP_EQUAL",
              "hex": "a9146ad3f81523c87aa17f1dfa08271cf57b6277c98e87",
              "reqSigs": 1,
              "type": "scripthash",
              "addresses": [
                "bchtest:pp4d87q4y0y84gtlrhaqsfcu74akya7f3c54m3nhzk"
              ]
            }
          },
          {
            "value": 0.00048999,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 68ccb0e4918444bddb05dccb313d8c979e8e25f2 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a91468ccb0e4918444bddb05dccb313d8c979e8e25f288ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "bchtest:qp5vev8yjxzyf0wmqhwvkvfa3jtear397gwsfxg7sa"
              ]
            }
          },
          {
            "value": 0,
            "n": 2,
            "scriptPubKey": {
              "asm": "OP_RETURN 307863333730373433333331423337643343364430456537393842333931386636353631416632433932",
              "hex": "6a2a307863333730373433333331423337643343364430456537393842333931386636353631416632433932",
              "type": "nulldata"
            }
          }
        ],
        "hex": "020000000181471442ade65b89e5e131078a0c6a3c801236a823440e4b6f04bba8d0b89e3a010000006b483045022100fcf716f6b6cb75be60c1b4f399facc7bc596fdcb521008cb0ccb6d8045a20f6a0220236859c32ee5f7868e6c97657dc35cfd2143b6ec5c5b62eba7006b0f63cc9b00412102d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3effffffff03891300000000000017a9146ad3f81523c87aa17f1dfa08271cf57b6277c98e8767bf0000000000001976a91468ccb0e4918444bddb05dccb313d8c979e8e25f288ac00000000000000002c6a2a30786333373037343333333142333764334336443045653739384233393138663635363141663243393200000000"
      }
    ],
    "time": 1662779022,
    "mediantime": 1662775280,
    "nonce": 819914115,
    "bits": "1d00ffff",
    "difficulty": 1,
    "chainwork": "000000000000000000000000000000000000000000000092ebf98c2b37bb78b6",
    "nTx": 2,
    "previousblockhash": "000000000000006331122ef4a6def1c696b236d45ab2b214278161badc72a488",
    "nextblockhash": "00000000000000fd76dbde24e02a374d66d25c3c6afba6ecef37bc02eb42a6ea"
  }`

	var bi BlockInfo
	err := json.Unmarshal([]byte(blockJson), &bi)
	require.NoError(t, err)

	covenantAddr := "6ad3f81523c87aa17f1dfa08271cf57b6277c98e"
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
    "TxID": "c01ab2bfa4a7f64cf781e886844de836e7b45f2c6150de380cb891045e8353c9",
    "Index": 0,
    "Amount": "0x2d7bdc490400"
  },
  "Receiver": "c370743331b37d3c6d0ee798b3918f6561af2c92",
  "CovenantAddress": "6ad3f81523c87aa17f1dfa08271cf57b6277c98e"
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

func TestFindRedeemableTx(t *testing.T) {
	// https://www.blockchain.com/bch-testnet/block/1511443 #tx6
	blockJson := `
{
    "hash": "00000000e48695891ef4a9d46bd3e808d9f89fc5e6da13d157f7635f70175b37",
    "confirmations": 4608,
    "strippedsize": 0,
    "size": 2610,
    "height": 1511443,
    "version": 536870912,
    "versionHex": "20000000",
    "merkleroot": "c4af7fa4753e660b01336a7f73a549c5ca09b7ed0b463b34013fa42dd12d8bd9",
    "tx": [
      {
        "hex": "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff48031310170c0b2f454233322f414431322f04589ae7620421ab08120c89a1286200000000000000000a626368706f6f6c172f20626974636f696e636173682e6e6574776f726b202fffffffff01e31d5402000000001976a914158b5d181552c9f4f267c0de68aae4963043993988ac00000000",
        "txid": "e940672576b56e4659fcb823e2d07487ef1b4587e950b9457b7f24ae4bba5ce1",
        "hash": "e940672576b56e4659fcb823e2d07487ef1b4587e950b9457b7f24ae4bba5ce1",
        "size": 157,
        "version": 1,
        "locktime": 0,
        "vin": [
          {
            "coinbase": "031310170c0b2f454233322f414431322f04589ae7620421ab08120c89a1286200000000000000000a626368706f6f6c172f20626974636f696e636173682e6e6574776f726b202f",
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 0.39067107,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 158b5d181552c9f4f267c0de68aae49630439939 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a914158b5d181552c9f4f267c0de68aae4963043993988ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qq2ckhgcz4fvna8jvlqdu692ujtrqsue8yarpm648v"
              ]
            }
          }
        ],
        "blockhash": "00000000e48695891ef4a9d46bd3e808d9f89fc5e6da13d157f7635f70175b37",
        "confirmations": 4608,
        "time": 1659345496,
        "blocktime": 1659345496
      },
      {
        "hex": "010000000114eb97bbaf64e6c55f18c6d729e3cfdc3843de8c50770e28e19f53b71941f93b010000006a47304402201385d9806c679ba5dbf4fcf7590ec58bfe0aa609d73d517e28d0754012c7b69e0220605d111fde4ebe21687108fc8246a2be06befc7b9011c73a79adb9c413c45abe412103380aba6bc38912ec49e8d5902b429857dc67f222d3203347a9e03be5196c6c81feffffff02201d9a00000000001976a9148253469f3c5eceeae25eb3150fddf82e9805146588ace4808ed4000000001976a914075fc9a3780417d47d78fc26466495d9cb25534c88ac12101700",
        "txid": "02d4cbfd7cbe3f75f4a7a4a4449df4206fb66fa9c5fb7b8048b8e53a4b88fce8",
        "hash": "02d4cbfd7cbe3f75f4a7a4a4449df4206fb66fa9c5fb7b8048b8e53a4b88fce8",
        "size": 225,
        "version": 1,
        "locktime": 1511442,
        "vin": [
          {
            "txid": "3bf94119b7539fe1280e77508cde4338dccfe329d7c6185fc5e664afbb97eb14",
            "vout": 1,
            "scriptSig": {
              "asm": "304402201385d9806c679ba5dbf4fcf7590ec58bfe0aa609d73d517e28d0754012c7b69e0220605d111fde4ebe21687108fc8246a2be06befc7b9011c73a79adb9c413c45abe41 03380aba6bc38912ec49e8d5902b429857dc67f222d3203347a9e03be5196c6c81",
              "hex": "47304402201385d9806c679ba5dbf4fcf7590ec58bfe0aa609d73d517e28d0754012c7b69e0220605d111fde4ebe21687108fc8246a2be06befc7b9011c73a79adb9c413c45abe412103380aba6bc38912ec49e8d5902b429857dc67f222d3203347a9e03be5196c6c81"
            },
            "sequence": 4294967294
          }
        ],
        "vout": [
          {
            "value": 0.101,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 8253469f3c5eceeae25eb3150fddf82e98051465 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a9148253469f3c5eceeae25eb3150fddf82e9805146588ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qzp9x35l830va6hzt6e32r7alqhfspg5v56hfpdltq"
              ]
            }
          },
          {
            "value": 35.661089,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 075fc9a3780417d47d78fc26466495d9cb25534c OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a914075fc9a3780417d47d78fc26466495d9cb25534c88ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qqr4ljdr0qzp04ra0r7zv3nyjhvukf2nfs2ytvx3tp"
              ]
            }
          }
        ],
        "blockhash": "00000000e48695891ef4a9d46bd3e808d9f89fc5e6da13d157f7635f70175b37",
        "confirmations": 4608,
        "time": 1659345496,
        "blocktime": 1659345496
      },
      {
        "hex": "02000000017ac1d82c7241d5489f07427a4036aa3b5fbef7a6a9b29f18cc78e46fd8a96497010000006a4730440220185bfd689e9b664802386559309f085bd4c9e28427b59b0e71115efd64b9112c022020aa703a7efe4c99e496ff363923a9ad9eb38a35a76d45e8358760b2bd263535412102d7dcdefcb690b9766a4b1bdb5430be814cdce3027275888313753c54ea86e598ffffffff02003e4900000000001976a9140a2ad7b8685378ba97a11fd626fdc217171e16b588ac98cf2700000000001976a914251119599ddb6554e8e5f426875f9d85d93776a188ac00000000",
        "txid": "1d97557e82d4be3c1df75a027925bdb3bc4599f7e68c2109bf3f93682b3ee3ef",
        "hash": "1d97557e82d4be3c1df75a027925bdb3bc4599f7e68c2109bf3f93682b3ee3ef",
        "size": 225,
        "version": 2,
        "locktime": 0,
        "vin": [
          {
            "txid": "9764a9d86fe478cc189fb2a9a6f7be5f3baa36407a42079f48d541722cd8c17a",
            "vout": 1,
            "scriptSig": {
              "asm": "30440220185bfd689e9b664802386559309f085bd4c9e28427b59b0e71115efd64b9112c022020aa703a7efe4c99e496ff363923a9ad9eb38a35a76d45e8358760b2bd26353541 02d7dcdefcb690b9766a4b1bdb5430be814cdce3027275888313753c54ea86e598",
              "hex": "4730440220185bfd689e9b664802386559309f085bd4c9e28427b59b0e71115efd64b9112c022020aa703a7efe4c99e496ff363923a9ad9eb38a35a76d45e8358760b2bd263535412102d7dcdefcb690b9766a4b1bdb5430be814cdce3027275888313753c54ea86e598"
            },
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 0.048,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 0a2ad7b8685378ba97a11fd626fdc217171e16b5 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a9140a2ad7b8685378ba97a11fd626fdc217171e16b588ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qq9z44acdpfh3w5h5y0avfhacgt3w8skk5dzhptx23"
              ]
            }
          },
          {
            "value": 0.02609048,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 251119599ddb6554e8e5f426875f9d85d93776a1 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a914251119599ddb6554e8e5f426875f9d85d93776a188ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qqj3zx2enhdk248guh6zdp6lnkzajdmk5y5rtjsrf9"
              ]
            }
          }
        ],
        "blockhash": "00000000e48695891ef4a9d46bd3e808d9f89fc5e6da13d157f7635f70175b37",
        "confirmations": 4608,
        "time": 1659345496,
        "blocktime": 1659345496
      },
      {
        "hex": "010000000176c898502ab1dcf67bce872a9ca0bb9ecba84c289c8749d36020f847133adeeb010000006a47304402203213600f4c23bb7d6b96c39a7a1ce90da7284d143160080700e1df9a418c58360220354bbe4e2401e0b5c77be2e76d4b8d273a1a45bd804b18fbc28483ca33b378eb4121032eaa9e90e72138388edb9038d9e94bb699f50139bb362087831008e554f64ed5feffffff02201d9a00000000001976a914381a31295ce7e60db4183051a1c359f08627bd6a88ac309f28d5000000001976a9144b8b4246cee6eb6c1f46382a2bba9a288f86f51288ac12101700",
        "txid": "3bf94119b7539fe1280e77508cde4338dccfe329d7c6185fc5e664afbb97eb14",
        "hash": "3bf94119b7539fe1280e77508cde4338dccfe329d7c6185fc5e664afbb97eb14",
        "size": 225,
        "version": 1,
        "locktime": 1511442,
        "vin": [
          {
            "txid": "ebde3a1347f82060d349879c284ca8cb9ebba09c2a87ce7bf6dcb12a5098c876",
            "vout": 1,
            "scriptSig": {
              "asm": "304402203213600f4c23bb7d6b96c39a7a1ce90da7284d143160080700e1df9a418c58360220354bbe4e2401e0b5c77be2e76d4b8d273a1a45bd804b18fbc28483ca33b378eb41 032eaa9e90e72138388edb9038d9e94bb699f50139bb362087831008e554f64ed5",
              "hex": "47304402203213600f4c23bb7d6b96c39a7a1ce90da7284d143160080700e1df9a418c58360220354bbe4e2401e0b5c77be2e76d4b8d273a1a45bd804b18fbc28483ca33b378eb4121032eaa9e90e72138388edb9038d9e94bb699f50139bb362087831008e554f64ed5"
            },
            "sequence": 4294967294
          }
        ],
        "vout": [
          {
            "value": 0.101,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 381a31295ce7e60db4183051a1c359f08627bd6a OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a914381a31295ce7e60db4183051a1c359f08627bd6a88ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qqup5vfftnn7vrd5rqc9rgwrt8cgvfaadgymzse0ad"
              ]
            }
          },
          {
            "value": 35.762092,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 4b8b4246cee6eb6c1f46382a2bba9a288f86f512 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a9144b8b4246cee6eb6c1f46382a2bba9a288f86f51288ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qp9cksjxemnwkmqlgcuz52a6ng5glph4zg6azzuczl"
              ]
            }
          }
        ],
        "blockhash": "00000000e48695891ef4a9d46bd3e808d9f89fc5e6da13d157f7635f70175b37",
        "confirmations": 4608,
        "time": 1659345496,
        "blocktime": 1659345496
      },
      {
        "hex": "0100000001bfc98494ea6a1ab8c5c32f2f98878bab427dc26087f77fbdf1e42a3f440df091010000006441a9347f3f889f0429e14d3dd7745cfc852773d8b81fc7c0ceb047512534e2ac87b0aa275c20a74b7f9f12a2cdbe3bf442ebbc7ae6d10382e4ead45f79ec0d00fa412102e1fe5e348e585439d3bb059f80fa613ba4d6250e73ebe1f6f8eab7e33b4f39acffffffff02a0860100000000001976a9149f07b5e907c623e23bcbe084e305a3eb8f87179688acb3850100000000001976a9147142e0d74f19f013ef174faaa8f201cb5aa5efcf88ac00000000",
        "txid": "7aeb6f3cf2258adc82948a1794a451559bb6a1a396c66f880723ee37dade97df",
        "hash": "7aeb6f3cf2258adc82948a1794a451559bb6a1a396c66f880723ee37dade97df",
        "size": 219,
        "version": 1,
        "locktime": 0,
        "vin": [
          {
            "txid": "91f00d443f2ae4f1bd7ff78760c27d42ab8b87982f2fc3c5b81a6aea9484c9bf",
            "vout": 1,
            "scriptSig": {
              "asm": "a9347f3f889f0429e14d3dd7745cfc852773d8b81fc7c0ceb047512534e2ac87b0aa275c20a74b7f9f12a2cdbe3bf442ebbc7ae6d10382e4ead45f79ec0d00fa41 02e1fe5e348e585439d3bb059f80fa613ba4d6250e73ebe1f6f8eab7e33b4f39ac",
              "hex": "41a9347f3f889f0429e14d3dd7745cfc852773d8b81fc7c0ceb047512534e2ac87b0aa275c20a74b7f9f12a2cdbe3bf442ebbc7ae6d10382e4ead45f79ec0d00fa412102e1fe5e348e585439d3bb059f80fa613ba4d6250e73ebe1f6f8eab7e33b4f39ac"
            },
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 0.001,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 9f07b5e907c623e23bcbe084e305a3eb8f871796 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a9149f07b5e907c623e23bcbe084e305a3eb8f87179688ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qz0s0d0fqlrz8c3me0sgfcc9504clpchjc9d8del2t"
              ]
            }
          },
          {
            "value": 0.00099763,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 7142e0d74f19f013ef174faaa8f201cb5aa5efcf OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a9147142e0d74f19f013ef174faaa8f201cb5aa5efcf88ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qpc59cxhfuvlqyl0za86428jq8944f00eu4mw422l6"
              ]
            }
          }
        ],
        "blockhash": "00000000e48695891ef4a9d46bd3e808d9f89fc5e6da13d157f7635f70175b37",
        "confirmations": 4608,
        "time": 1659345496,
        "blocktime": 1659345496
      },
      {
        "hex": "02000000017c02cdae7601e10c89246741ecfecd92787fe8350a3fb7a0878dc86c9dc4944700000000fd730414553fac4027a7a3c4e8a3eaea75aab173d3c8144b1427c4ca4766591e6bb8cd71b83143946c53eaf9a32103883b732620e238e74041e5fab900234dc80f7a48d56a1bf41e8523c4661f82432103bfe6f6ecb5e10662481aeb6f6408db2a32b9b86a660acbb8c5374dbb976e53ca210386f450b1bee3b220c6a9a25515f15f05bd80a23e5f707873dfbac52db933b27d2102fbbc3870035c2ee30cfa3102aff15e58bdfc0d0f95998cd7e1eeebc09cdb6873210271ea0c254ebbb7ed78668ba8653abe222b9f7177642d3a75709d95912a8d9d2c210394ec324d59305638ead14b4f4da9a50c793f1e328e180f92c04a4990bb573af121038fd3d33474e1bd453614f85d8fb1edecae92255867d18a9048669119fb710af52103fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e421035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd82102d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd699447304402207bd5dd3fce07a56d28a1e456c538901fbc9820f52855f5502ecb7c5a35b05d41022027611a420ba0617764672d11bf908c7f31b6f05f66ca2268fa3135f689515214414730440220148300596475fd0b80c2e9704caa4a77485ab50e6666b80dec73d067ed38a30402205385a562dde1c06e15d33280b32908ce2ca42f6ef9760feb646679fafe1aeaae4147304402205b5d26db1d7cd4b9ddebef0c3a74124d5b8ce549946d2ddcd6ba9121d8adb886022038d0572915960a1e1ca4e921c4bf6432e336f83e2c2a5f7e48697783d8ef92fd414730440220541a6035542abd33b318584b01a62ea4678e4d407e28ec5c4a1d98be8da20a7802203e05ae652cb0a8f615fca5a0531df7111da80a7985598f446761ccf7ccbea8b241483045022100a2e36c1b73bfe1baa2eb4a44c5ee675eaee67e85a26eecf5d54cbddf655ba3e802207fb19769f4e9ba97093af84a1bc96465c76c9a4d10b23d76d84049d6e9517c91414730440220081a832b6f0beef5a2311bc24eb818bb34e598d88fa9cc217b41e4d88cc517de02203f158e5746f481a4676c3cbe7edd6a2e648a2294552458d64fea0dda27ad72ed4147304402203b289d27eed12a2abb874c41a36bee14c5a4340f34e1306c21b32fad2040613b022061db8baa44af6adffc0f420bd7ff40b23501fb0d3a32e86e718730e2729d965841004cf914553fac4027a7a3c4e8a3eaea75aab173d3c8144b1427c4ca4766591e6bb8cd71b83143946c53eaf9a35279009c635a795c797e5d797e5e797e5f797e60797e0111797e0112797e0113797e0114797ea952798800717c567a577a587a597a5a7a575c7a5d7a5e7a5f7a607a01117a01127a01137a01147a01157a5aafc3519dc4519d00cc00c602d007949d5379879154797b87919b63011453797e01147e52797ec1012a7f777e02a91478a97e01877e00cd78886d686d7551677b519d547956797e57797ea98800727c52557a567a577a53afc0009d00cc00c69d024065b27501147b7ec101157f777e02a9147ca97e01877e00cd877768feffffff01401f0000000000001976a91499c7e0b48a05cd6024b22cd490fcee30aa51d86288ac12101700",
        "txid": "bd1731ce1a008e0b778fe73f20b67e49df2acd178ff81b09bd8a91b8acdd9538",
        "hash": "bd1731ce1a008e0b778fe73f20b67e49df2acd178ff81b09bd8a91b8acdd9538",
        "size": 1226,
        "version": 2,
        "locktime": 1511442,
        "vin": [
          {
            "txid": "4794c49d6cc88d87a0b73f0a35e87f7892cdfeec416724890ce10176aecd027c",
            "vout": 0,
            "scriptSig": {
              "asm": "553fac4027a7a3c4e8a3eaea75aab173d3c8144b 27c4ca4766591e6bb8cd71b83143946c53eaf9a3 03883b732620e238e74041e5fab900234dc80f7a48d56a1bf41e8523c4661f8243 03bfe6f6ecb5e10662481aeb6f6408db2a32b9b86a660acbb8c5374dbb976e53ca 0386f450b1bee3b220c6a9a25515f15f05bd80a23e5f707873dfbac52db933b27d 02fbbc3870035c2ee30cfa3102aff15e58bdfc0d0f95998cd7e1eeebc09cdb6873 0271ea0c254ebbb7ed78668ba8653abe222b9f7177642d3a75709d95912a8d9d2c 0394ec324d59305638ead14b4f4da9a50c793f1e328e180f92c04a4990bb573af1 038fd3d33474e1bd453614f85d8fb1edecae92255867d18a9048669119fb710af5 03fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e4 035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd8 02d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6994 304402207bd5dd3fce07a56d28a1e456c538901fbc9820f52855f5502ecb7c5a35b05d41022027611a420ba0617764672d11bf908c7f31b6f05f66ca2268fa3135f68951521441 30440220148300596475fd0b80c2e9704caa4a77485ab50e6666b80dec73d067ed38a30402205385a562dde1c06e15d33280b32908ce2ca42f6ef9760feb646679fafe1aeaae41 304402205b5d26db1d7cd4b9ddebef0c3a74124d5b8ce549946d2ddcd6ba9121d8adb886022038d0572915960a1e1ca4e921c4bf6432e336f83e2c2a5f7e48697783d8ef92fd41 30440220541a6035542abd33b318584b01a62ea4678e4d407e28ec5c4a1d98be8da20a7802203e05ae652cb0a8f615fca5a0531df7111da80a7985598f446761ccf7ccbea8b241 3045022100a2e36c1b73bfe1baa2eb4a44c5ee675eaee67e85a26eecf5d54cbddf655ba3e802207fb19769f4e9ba97093af84a1bc96465c76c9a4d10b23d76d84049d6e9517c9141 30440220081a832b6f0beef5a2311bc24eb818bb34e598d88fa9cc217b41e4d88cc517de02203f158e5746f481a4676c3cbe7edd6a2e648a2294552458d64fea0dda27ad72ed41 304402203b289d27eed12a2abb874c41a36bee14c5a4340f34e1306c21b32fad2040613b022061db8baa44af6adffc0f420bd7ff40b23501fb0d3a32e86e718730e2729d965841 0 14553fac4027a7a3c4e8a3eaea75aab173d3c8144b1427c4ca4766591e6bb8cd71b83143946c53eaf9a35279009c635a795c797e5d797e5e797e5f797e60797e0111797e0112797e0113797e0114797ea952798800717c567a577a587a597a5a7a575c7a5d7a5e7a5f7a607a01117a01127a01137a01147a01157a5aafc3519dc4519d00cc00c602d007949d5379879154797b87919b63011453797e01147e52797ec1012a7f777e02a91478a97e01877e00cd78886d686d7551677b519d547956797e57797ea98800727c52557a567a577a53afc0009d00cc00c69d024065b27501147b7ec101157f777e02a9147ca97e01877e00cd877768",
              "hex": "14553fac4027a7a3c4e8a3eaea75aab173d3c8144b1427c4ca4766591e6bb8cd71b83143946c53eaf9a32103883b732620e238e74041e5fab900234dc80f7a48d56a1bf41e8523c4661f82432103bfe6f6ecb5e10662481aeb6f6408db2a32b9b86a660acbb8c5374dbb976e53ca210386f450b1bee3b220c6a9a25515f15f05bd80a23e5f707873dfbac52db933b27d2102fbbc3870035c2ee30cfa3102aff15e58bdfc0d0f95998cd7e1eeebc09cdb6873210271ea0c254ebbb7ed78668ba8653abe222b9f7177642d3a75709d95912a8d9d2c210394ec324d59305638ead14b4f4da9a50c793f1e328e180f92c04a4990bb573af121038fd3d33474e1bd453614f85d8fb1edecae92255867d18a9048669119fb710af52103fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e421035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd82102d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd699447304402207bd5dd3fce07a56d28a1e456c538901fbc9820f52855f5502ecb7c5a35b05d41022027611a420ba0617764672d11bf908c7f31b6f05f66ca2268fa3135f689515214414730440220148300596475fd0b80c2e9704caa4a77485ab50e6666b80dec73d067ed38a30402205385a562dde1c06e15d33280b32908ce2ca42f6ef9760feb646679fafe1aeaae4147304402205b5d26db1d7cd4b9ddebef0c3a74124d5b8ce549946d2ddcd6ba9121d8adb886022038d0572915960a1e1ca4e921c4bf6432e336f83e2c2a5f7e48697783d8ef92fd414730440220541a6035542abd33b318584b01a62ea4678e4d407e28ec5c4a1d98be8da20a7802203e05ae652cb0a8f615fca5a0531df7111da80a7985598f446761ccf7ccbea8b241483045022100a2e36c1b73bfe1baa2eb4a44c5ee675eaee67e85a26eecf5d54cbddf655ba3e802207fb19769f4e9ba97093af84a1bc96465c76c9a4d10b23d76d84049d6e9517c91414730440220081a832b6f0beef5a2311bc24eb818bb34e598d88fa9cc217b41e4d88cc517de02203f158e5746f481a4676c3cbe7edd6a2e648a2294552458d64fea0dda27ad72ed4147304402203b289d27eed12a2abb874c41a36bee14c5a4340f34e1306c21b32fad2040613b022061db8baa44af6adffc0f420bd7ff40b23501fb0d3a32e86e718730e2729d965841004cf914553fac4027a7a3c4e8a3eaea75aab173d3c8144b1427c4ca4766591e6bb8cd71b83143946c53eaf9a35279009c635a795c797e5d797e5e797e5f797e60797e0111797e0112797e0113797e0114797ea952798800717c567a577a587a597a5a7a575c7a5d7a5e7a5f7a607a01117a01127a01137a01147a01157a5aafc3519dc4519d00cc00c602d007949d5379879154797b87919b63011453797e01147e52797ec1012a7f777e02a91478a97e01877e00cd78886d686d7551677b519d547956797e57797ea98800727c52557a567a577a53afc0009d00cc00c69d024065b27501147b7ec101157f777e02a9147ca97e01877e00cd877768"
            },
            "sequence": 4294967294
          }
        ],
        "vout": [
          {
            "value": 8e-05,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 99c7e0b48a05cd6024b22cd490fcee30aa51d862 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a91499c7e0b48a05cd6024b22cd490fcee30aa51d86288ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qzvu0c953gzu6cpykgkdfy8uacc255wcvgmp7ekj7y"
              ]
            }
          }
        ],
        "blockhash": "00000000e48695891ef4a9d46bd3e808d9f89fc5e6da13d157f7635f70175b37",
        "confirmations": 4608,
        "time": 1659345496,
        "blocktime": 1659345496
      },
      {
        "hex": "010000000184fa30667dfefa5dc1fb64060b632d98c3dba6088c8a82fd31a00fb8ea73ccd3010000006b48304502210082092edf199efb2f810f1c688a048461f7b848a4b7cf3be382e8fb29109bf2b2022060d1e05173bbe4d52d71ed666da2354e13ad6b56c11ef769ddf5788acd2d23f1412102aac58283a65ec6e3b2c25de2d6d2cc19fe9ab219c3c83c213acd480c2bdc597dffffffff03e130ba06000000001976a9149f8f290a203c8913156d4086059675c9d609a89d88ac87247c04000000001976a9147161e182651081441414789bfd3b96034bbd4d7688ac0000000000000000116a0f4d4947524154453a3338313439353900000000",
        "txid": "d4bb120e210cb130e37144a2e0327b88703aa79f3c1c86557896ad3f6765099d",
        "hash": "d4bb120e210cb130e37144a2e0327b88703aa79f3c1c86557896ad3f6765099d",
        "size": 252,
        "version": 1,
        "locktime": 0,
        "vin": [
          {
            "txid": "d3cc73eab80fa031fd828a8c08a6dbc3982d630b0664fbc15dfafe7d6630fa84",
            "vout": 1,
            "scriptSig": {
              "asm": "304502210082092edf199efb2f810f1c688a048461f7b848a4b7cf3be382e8fb29109bf2b2022060d1e05173bbe4d52d71ed666da2354e13ad6b56c11ef769ddf5788acd2d23f141 02aac58283a65ec6e3b2c25de2d6d2cc19fe9ab219c3c83c213acd480c2bdc597d",
              "hex": "48304502210082092edf199efb2f810f1c688a048461f7b848a4b7cf3be382e8fb29109bf2b2022060d1e05173bbe4d52d71ed666da2354e13ad6b56c11ef769ddf5788acd2d23f1412102aac58283a65ec6e3b2c25de2d6d2cc19fe9ab219c3c83c213acd480c2bdc597d"
            },
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 1.12865505,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 9f8f290a203c8913156d4086059675c9d609a89d OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a9149f8f290a203c8913156d4086059675c9d609a89d88ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qz0c72g2yq7gjyc4d4qgvpvkwhyavzdgn5j339jcxl"
              ]
            }
          },
          {
            "value": 0.75244679,
            "n": 1,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 7161e182651081441414789bfd3b96034bbd4d76 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a9147161e182651081441414789bfd3b96034bbd4d7688ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qpckrcvzv5ggz3q5z3ufhlfmjcp5h02dwcs9a06098"
              ]
            }
          },
          {
            "value": 0,
            "n": 2,
            "scriptPubKey": {
              "asm": "OP_RETURN 4d4947524154453a33383134393539",
              "hex": "6a0f4d4947524154453a33383134393539",
              "type": "nulldata"
            }
          }
        ],
        "blockhash": "00000000e48695891ef4a9d46bd3e808d9f89fc5e6da13d157f7635f70175b37",
        "confirmations": 4608,
        "time": 1659345496,
        "blocktime": 1659345496
      }
    ],
    "time": 1659345496,
    "nonce": 2066317056,
    "bits": "1d00ffff",
    "difficulty": 1,
    "previousblockhash": "00000000000003801ac15c6f8f8e686d913ee7bb46670db2dca05258080d3f5e",
    "nextblockhash": "00000000000002645dece9c0c773af4d310dd8c0db5a39454826a6b5e6671d17"
  }
`

	var bi BlockInfo
	err := json.Unmarshal([]byte(blockJson), &bi)
	require.NoError(t, err)

	covenantAddr := "ccf8fb324aebbc9f53a7fb28138a3d703b9e60d0"
	parser := &CcTxParser{
		CurrentCovenantAddress: covenantAddr,
		UtxoSet: map[[32]byte]uint32{
			gethcmn.HexToHash("4794c49d6cc88d87a0b73f0a35e87f7892cdfeec416724890ce10176aecd027c"): 0,
		},
	}
	infos := parser.GetCCUTXOTransferInfo(&bi)
	require.Len(t, infos, 1)
	require.Equal(t, `{
  "Type": 2,
  "PrevUTXO": {
    "TxID": "4794c49d6cc88d87a0b73f0a35e87f7892cdfeec416724890ce10176aecd027c",
    "Index": 0,
    "Amount": "0x0"
  },
  "UTXO": {
    "TxID": "0000000000000000000000000000000000000000000000000000000000000000",
    "Index": 0,
    "Amount": "0x0"
  },
  "Receiver": "0000000000000000000000000000000000000000",
  "CovenantAddress": "0000000000000000000000000000000000000000"
}`, ccTransferInfoToJSON(infos[0]))
}

func TestFindConvertTx(t *testing.T) {
	// https://www.blockchain.com/bch-testnet/block/1511588 #tx2
	blockJson := `
{
    "hash": "00000000000000c1e4447ded2dbd15fe29030810bc64c09c04868959c8f5cf3b",
    "confirmations": 4469,
    "strippedsize": 0,
    "size": 1417,
    "height": 1511588,
    "version": 710221824,
    "versionHex": "2a552000",
    "merkleroot": "1f8b7d802955f1584972887206db745a1e92a8ecd342b50a3e7cd4892f38036c",
    "tx": [
      {
        "hex": "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff1903a410170120120909092009092009043ab4002d5600000000ffffffff01b4135402000000001976a914f13d7a29aa9bfa88ff898387a074b04c8f67b4d888ac00000000",
        "txid": "69b48a6ff6e902412e4366884f0b90f1b0b001a9692e2a0a69fe16c6f7f40919",
        "hash": "69b48a6ff6e902412e4366884f0b90f1b0b001a9692e2a0a69fe16c6f7f40919",
        "size": 110,
        "version": 1,
        "locktime": 0,
        "vin": [
          {
            "coinbase": "03a410170120120909092009092009043ab4002d5600000000",
            "sequence": 4294967295
          }
        ],
        "vout": [
          {
            "value": 0.390645,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_DUP OP_HASH160 f13d7a29aa9bfa88ff898387a074b04c8f67b4d8 OP_EQUALVERIFY OP_CHECKSIG",
              "hex": "76a914f13d7a29aa9bfa88ff898387a074b04c8f67b4d888ac",
              "reqSigs": 1,
              "type": "pubkeyhash",
              "addresses": [
                "qrcn673f42dl4z8l3xpc0gr5kpxg7ea5mqhj3atxd3"
              ]
            }
          }
        ],
        "blockhash": "00000000000000c1e4447ded2dbd15fe29030810bc64c09c04868959c8f5cf3b",
        "confirmations": 4469,
        "time": 1659425576,
        "blocktime": 1659425576
      },
      {
        "hex": "02000000012b6554caa41f47fa2c376e304d3686850762350df982544fe58a8a9ba03c9fba00000000fd75041457a158339cd184037a5b27d3033cb721713f27c31427c4ca4766591e6bb8cd71b83143946c53eaf9a32103883b732620e238e74041e5fab900234dc80f7a48d56a1bf41e8523c4661f82432103bfe6f6ecb5e10662481aeb6f6408db2a32b9b86a660acbb8c5374dbb976e53ca210386f450b1bee3b220c6a9a25515f15f05bd80a23e5f707873dfbac52db933b27d2102fbbc3870035c2ee30cfa3102aff15e58bdfc0d0f95998cd7e1eeebc09cdb6873210271ea0c254ebbb7ed78668ba8653abe222b9f7177642d3a75709d95912a8d9d2c210394ec324d59305638ead14b4f4da9a50c793f1e328e180f92c04a4990bb573af121038fd3d33474e1bd453614f85d8fb1edecae92255867d18a9048669119fb710af52103fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e421035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd82102d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6994483045022100da71035a60844b1cfc94e5c5893b5c05fa485dd8f2aa0c3ddf4adf6bdc79a5a502200b6e5c6ce8cbe8dac0ec967a73eda15370d2475890d858cd400781441363d5d54147304402201609927a7a4fb610ce175daddad164e0d91987c74f2a66ce142e881a7bc0efe802200c9e0870a182550690c50d8cb70d3119b2dc1dbf932eb0ebbe6d37d773963cab41473044022072b855faac8e14860a7fcfbe421435934346be684a5ac855efbceeeaff0b2998022037c72565850d6f8974191327a5c33f87c944648f592e7b3e1291f3ba657b02d9414730440220117535312284088e5872c723ddbc66530ffab5548f32228810f5bc53b76c53110220012bf6bf8b4bd4c9f5c5a83347af81aa297a0c597e6f99ee3e23abb96e5e86ab41483045022100c702deb1c32df3a75571356b6b98d588fda16d6835a5a95402ac1d3c2bffdc4d0220325231de03b0bac3863fd02efe2da294212e389ce412e6e4356f0de142234d70414830450221009cd14d49cbddcb7de5268c51685d42c8451d15fd56151671d5bea1266177c817022001fc372c8b7a29a923f8ee3f44443df5990ecc614c23a1e3b675fc0ac4870aa741473044022008fd154a74fb144f282b4e1ea51e7a0740b2a793d27afe5b8ad3353448775acc022014cc728b00091c5ba75370fdc09e846fcf8d2b4ec1827b4bb98dedc0468be18741004cf914553fac4027a7a3c4e8a3eaea75aab173d3c8144b1427c4ca4766591e6bb8cd71b83143946c53eaf9a35279009c635a795c797e5d797e5e797e5f797e60797e0111797e0112797e0113797e0114797ea952798800717c567a577a587a597a5a7a575c7a5d7a5e7a5f7a607a01117a01127a01137a01147a01157a5aafc3519dc4519d00cc00c602d007949d5379879154797b87919b63011453797e01147e52797ec1012a7f777e02a91478a97e01877e00cd78886d686d7551677b519d547956797e57797ea98800727c52557a567a577a53afc0009d00cc00c69d024065b27501147b7ec101157f777e02a9147ca97e01877e00cd877768feffffff01401f00000000000017a914ae2c75b69475fe48a15f1a838b5238f4cc54bd5887a3101700",
        "txid": "7106656d2013e63345441b2a3bcb24c0b72dd90ba04b829bbf3be772c9329f29",
        "hash": "7106656d2013e63345441b2a3bcb24c0b72dd90ba04b829bbf3be772c9329f29",
        "size": 1226,
        "version": 2,
        "locktime": 1511587,
        "vin": [
          {
            "txid": "ba9f3ca09b8a8ae54f5482f90d3562078586364d306e372cfa471fa4ca54652b",
            "vout": 0,
            "scriptSig": {
              "asm": "57a158339cd184037a5b27d3033cb721713f27c3 27c4ca4766591e6bb8cd71b83143946c53eaf9a3 03883b732620e238e74041e5fab900234dc80f7a48d56a1bf41e8523c4661f8243 03bfe6f6ecb5e10662481aeb6f6408db2a32b9b86a660acbb8c5374dbb976e53ca 0386f450b1bee3b220c6a9a25515f15f05bd80a23e5f707873dfbac52db933b27d 02fbbc3870035c2ee30cfa3102aff15e58bdfc0d0f95998cd7e1eeebc09cdb6873 0271ea0c254ebbb7ed78668ba8653abe222b9f7177642d3a75709d95912a8d9d2c 0394ec324d59305638ead14b4f4da9a50c793f1e328e180f92c04a4990bb573af1 038fd3d33474e1bd453614f85d8fb1edecae92255867d18a9048669119fb710af5 03fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e4 035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd8 02d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6994 3045022100da71035a60844b1cfc94e5c5893b5c05fa485dd8f2aa0c3ddf4adf6bdc79a5a502200b6e5c6ce8cbe8dac0ec967a73eda15370d2475890d858cd400781441363d5d541 304402201609927a7a4fb610ce175daddad164e0d91987c74f2a66ce142e881a7bc0efe802200c9e0870a182550690c50d8cb70d3119b2dc1dbf932eb0ebbe6d37d773963cab41 3044022072b855faac8e14860a7fcfbe421435934346be684a5ac855efbceeeaff0b2998022037c72565850d6f8974191327a5c33f87c944648f592e7b3e1291f3ba657b02d941 30440220117535312284088e5872c723ddbc66530ffab5548f32228810f5bc53b76c53110220012bf6bf8b4bd4c9f5c5a83347af81aa297a0c597e6f99ee3e23abb96e5e86ab41 3045022100c702deb1c32df3a75571356b6b98d588fda16d6835a5a95402ac1d3c2bffdc4d0220325231de03b0bac3863fd02efe2da294212e389ce412e6e4356f0de142234d7041 30450221009cd14d49cbddcb7de5268c51685d42c8451d15fd56151671d5bea1266177c817022001fc372c8b7a29a923f8ee3f44443df5990ecc614c23a1e3b675fc0ac4870aa741 3044022008fd154a74fb144f282b4e1ea51e7a0740b2a793d27afe5b8ad3353448775acc022014cc728b00091c5ba75370fdc09e846fcf8d2b4ec1827b4bb98dedc0468be18741 0 14553fac4027a7a3c4e8a3eaea75aab173d3c8144b1427c4ca4766591e6bb8cd71b83143946c53eaf9a35279009c635a795c797e5d797e5e797e5f797e60797e0111797e0112797e0113797e0114797ea952798800717c567a577a587a597a5a7a575c7a5d7a5e7a5f7a607a01117a01127a01137a01147a01157a5aafc3519dc4519d00cc00c602d007949d5379879154797b87919b63011453797e01147e52797ec1012a7f777e02a91478a97e01877e00cd78886d686d7551677b519d547956797e57797ea98800727c52557a567a577a53afc0009d00cc00c69d024065b27501147b7ec101157f777e02a9147ca97e01877e00cd877768",
              "hex": "1457a158339cd184037a5b27d3033cb721713f27c31427c4ca4766591e6bb8cd71b83143946c53eaf9a32103883b732620e238e74041e5fab900234dc80f7a48d56a1bf41e8523c4661f82432103bfe6f6ecb5e10662481aeb6f6408db2a32b9b86a660acbb8c5374dbb976e53ca210386f450b1bee3b220c6a9a25515f15f05bd80a23e5f707873dfbac52db933b27d2102fbbc3870035c2ee30cfa3102aff15e58bdfc0d0f95998cd7e1eeebc09cdb6873210271ea0c254ebbb7ed78668ba8653abe222b9f7177642d3a75709d95912a8d9d2c210394ec324d59305638ead14b4f4da9a50c793f1e328e180f92c04a4990bb573af121038fd3d33474e1bd453614f85d8fb1edecae92255867d18a9048669119fb710af52103fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e421035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd82102d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6994483045022100da71035a60844b1cfc94e5c5893b5c05fa485dd8f2aa0c3ddf4adf6bdc79a5a502200b6e5c6ce8cbe8dac0ec967a73eda15370d2475890d858cd400781441363d5d54147304402201609927a7a4fb610ce175daddad164e0d91987c74f2a66ce142e881a7bc0efe802200c9e0870a182550690c50d8cb70d3119b2dc1dbf932eb0ebbe6d37d773963cab41473044022072b855faac8e14860a7fcfbe421435934346be684a5ac855efbceeeaff0b2998022037c72565850d6f8974191327a5c33f87c944648f592e7b3e1291f3ba657b02d9414730440220117535312284088e5872c723ddbc66530ffab5548f32228810f5bc53b76c53110220012bf6bf8b4bd4c9f5c5a83347af81aa297a0c597e6f99ee3e23abb96e5e86ab41483045022100c702deb1c32df3a75571356b6b98d588fda16d6835a5a95402ac1d3c2bffdc4d0220325231de03b0bac3863fd02efe2da294212e389ce412e6e4356f0de142234d70414830450221009cd14d49cbddcb7de5268c51685d42c8451d15fd56151671d5bea1266177c817022001fc372c8b7a29a923f8ee3f44443df5990ecc614c23a1e3b675fc0ac4870aa741473044022008fd154a74fb144f282b4e1ea51e7a0740b2a793d27afe5b8ad3353448775acc022014cc728b00091c5ba75370fdc09e846fcf8d2b4ec1827b4bb98dedc0468be18741004cf914553fac4027a7a3c4e8a3eaea75aab173d3c8144b1427c4ca4766591e6bb8cd71b83143946c53eaf9a35279009c635a795c797e5d797e5e797e5f797e60797e0111797e0112797e0113797e0114797ea952798800717c567a577a587a597a5a7a575c7a5d7a5e7a5f7a607a01117a01127a01137a01147a01157a5aafc3519dc4519d00cc00c602d007949d5379879154797b87919b63011453797e01147e52797ec1012a7f777e02a91478a97e01877e00cd78886d686d7551677b519d547956797e57797ea98800727c52557a567a577a53afc0009d00cc00c69d024065b27501147b7ec101157f777e02a9147ca97e01877e00cd877768"
            },
            "sequence": 4294967294
          }
        ],
        "vout": [
          {
            "value": 8e-05,
            "n": 0,
            "scriptPubKey": {
              "asm": "OP_HASH160 ae2c75b69475fe48a15f1a838b5238f4cc54bd58 OP_EQUAL",
              "hex": "a914ae2c75b69475fe48a15f1a838b5238f4cc54bd5887",
              "reqSigs": 1,
              "type": "scripthash",
              "addresses": [
                "pzhzcadkj36luj9ptudg8z6j8r6vc49atqqhxjyaf3"
              ]
            }
          }
        ],
        "blockhash": "00000000000000c1e4447ded2dbd15fe29030810bc64c09c04868959c8f5cf3b",
        "confirmations": 4469,
        "time": 1659425576,
        "blocktime": 1659425576
      }
    ],
    "time": 1659425576,
    "nonce": 360652084,
    "bits": "1a014f2d",
    "difficulty": 12813878.5683818,
    "previousblockhash": "00000000000000520d71331a0e231fb1545ab8d90feb2efbe3ccec8b13766df6",
    "nextblockhash": "0000000000000119f94208f935478d69d134d099dd8376cabde0d6aca41db594"
  }
`

	var bi BlockInfo
	err := json.Unmarshal([]byte(blockJson), &bi)
	require.NoError(t, err)

	covenantAddr := "ae2c75b69475fe48a15f1a838b5238f4cc54bd58"
	parser := &CcTxParser{
		CurrentCovenantAddress: covenantAddr,
		UtxoSet: map[[32]byte]uint32{
			gethcmn.HexToHash("ba9f3ca09b8a8ae54f5482f90d3562078586364d306e372cfa471fa4ca54652b"): 0,
		},
	}
	infos := parser.GetCCUTXOTransferInfo(&bi)
	require.Len(t, infos, 1)
	require.Equal(t, `{
  "Type": 1,
  "PrevUTXO": {
    "TxID": "ba9f3ca09b8a8ae54f5482f90d3562078586364d306e372cfa471fa4ca54652b",
    "Index": 0,
    "Amount": "0x0"
  },
  "UTXO": {
    "TxID": "7106656d2013e63345441b2a3bcb24c0b72dd90ba04b829bbf3be772c9329f29",
    "Index": 0,
    "Amount": "0x48c273950000"
  },
  "Receiver": "0000000000000000000000000000000000000000",
  "CovenantAddress": "ae2c75b69475fe48a15f1a838b5238f4cc54bd58"
}`, ccTransferInfoToJSON(infos[0]))
}

func TestFindRedeemableTx2(t *testing.T) {
	//parser := CcTxParser{CurrentCovenantAddress: "0000000000000000000000000000000000001234"}
	parser := CcTxParser{CurrentCovenantAddress: gethcmn.HexToAddress("0000000000000000000000000000000000001234").String()}
	var txs []TxInfo
	_ = json.Unmarshal([]byte(`[{
  "txid":"5a06790b3f566fe43e67c6f57252d86d263d1b4bac521ccdb91d97bf48d02dfb",
  "hash":"5a06790b3f566fe43e67c6f57252d86d263d1b4bac521ccdb91d97bf48d02dfb",
  "version":2,
  "size":243,
  "locktime":0,
  "vin":[
    {
      "scriptSig":{
        "asm":"3045022100f5c60f0d71a884af901423dff2c5e67f6ee450c1017c7c6330782696085fca4f022045143186bf5e281ee3865d20ff87fce3eb0e31f0b6545996f6faf9c8705b2aab[ALL|FORKID] 02d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3e",
        "hex":"483045022100f5c60f0d71a884af901423dff2c5e67f6ee450c1017c7c6330782696085fca4f022045143186bf5e281ee3865d20ff87fce3eb0e31f0b6545996f6faf9c8705b2aab412102d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3e"
      },
      "sequence":4294967295,
      "txid":"3557528b10a44babd22f59b09c327de7f2e81e2c638543b55ed2b685f7759e5c",
      "vout":0
    }
  ],
  "vout":[
    {
      "value":0.00002,
      "n":0,
      "scriptPubKey":{
        "addresses":[
          "bchtest:pqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqjxsku0hxsh9"
        ],
        "asm":"OP_HASH160 0000000000000000000000000000000000001234 OP_EQUAL",
        "hex":"a914000000000000000000000000000000000000123487",
        "reqSigs":1,
        "type":"scripthash"
      }
    },
    {
      "value":0,
      "n":1,
      "scriptPubKey":{
        "asm":"OP_RETURN 307863333730373433333331423337643343364430456537393842333931386636353631416632433932",
        "hex":"6a2a307863333730373433333331423337643343364430456537393842333931386636353631416632433932",
        "type":"nulldata"
      }
    }
  ],
  "hex":"02000000015c9e75f785b6d25eb54385632c1ee8f2e77d329cb0592fd2ab4ba4108b525735000000006b483045022100f5c60f0d71a884af901423dff2c5e67f6ee450c1017c7c6330782696085fca4f022045143186bf5e281ee3865d20ff87fce3eb0e31f0b6545996f6faf9c8705b2aab412102d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3effffffff02d00700000000000017a91400000000000000000000000000000000000012348700000000000000002c6a2a30786333373037343333333142333764334336443045653739384233393138663635363141663243393200000000",
  "blockhash":"",
  "confirmations":0,
  "time":0,
  "blocktime":0
}]`), &txs)

	infos := parser.findRedeemableTx(txs)
	require.Len(t, infos, 1)
}
