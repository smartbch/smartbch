package types

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	cctypes "github.com/smartbch/smartbch/crosschain/types"
)

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
	require.Equal(t, cctypes.TransferType, infos[0].Type)
	require.Equal(t, cctypes.UTXO{}, infos[0].PrevUTXO)
	require.Equal(t, "7ff88192c5a5ee27237880230b4a9fc0c7e97d7dfe979831b23cd104d46160ee",
		hex.EncodeToString(infos[0].UTXO.TxID[:]))
	require.Equal(t, uint32(0), infos[0].UTXO.Index)
	require.Equal(t, uint256.NewInt(1e14).Bytes32(), infos[0].UTXO.Amount)
	require.Equal(t, "c370743331b37d3c6d0ee798b3918f6561af2c92",
		hex.EncodeToString(infos[0].Receiver[:]))
	require.Equal(t, "ccf8fb324aebbc9f53a7fb28138a3d703b9e60d0",
		hex.EncodeToString(infos[0].CovenantAddress[:]))

}
