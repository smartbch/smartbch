package crosschain

import (
	"github.com/smartbch/smartbch/internal/ethutils"
)

var ABI = ethutils.MustParseABI(`
[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "bytes32",
				"name": "mainnetTxId",
				"type": "bytes32"
			},
			{
				"indexed": true,
				"internalType": "bytes4",
				"name": "vOutIndex",
				"type": "bytes4"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "Burn",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "bytes32",
				"name": "mainnetTxId",
				"type": "bytes32"
			},
			{
				"indexed": true,
				"internalType": "bytes4",
				"name": "vOutIndex",
				"type": "bytes4"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "from",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "TransferToMainnet",
		"type": "event"
	},
	{
		"inputs": [
			{
				"internalType": "bytes",
				"name": "utxo",
				"type": "bytes"
			}
		],
		"name": "burnBCH",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "bytes",
				"name": "utxo",
				"type": "bytes"
			}
		],
		"name": "transferBCHToMainnet",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]
`)

func PackTransferBCHToMainnet(utxo [36]byte) []byte {
	return ABI.MustPack("transferBCHToMainnet", utxo[:])
}
func PackBurnBCH(utxo [36]byte) []byte {
	return ABI.MustPack("burnBCH", utxo[:])
}
