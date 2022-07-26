package crosschain

import (
	"math/big"

	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/smartbch/smartbch/internal/ethutils"
)

//event NewRedeemable(uint256 txid, uint32 vout, address covenantAddr);
//event NewLostAndFound(uint256 txid, uint32 vout, address covenantAddr);
//event Redeem(uint256 txid, uint32 vout, address covenantAddr, uint8 sourceType);
//event Convert(uint256 txid, uint32 vout, address newCovenantAddr);
//event ChangeAddr(uint256 prevTxid, uint32 prevVout, address newCovenantAddr, uint256 txid, uint32 vout);
//event Deleted(uint256 txid, uint32 vout, address covenantAddr, uint8 sourceType);

var ABI = ethutils.MustParseABI(`
[
	{
		"anonymous": false,
		"inputs": [
			{
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"internalType": "address",
				"name": "covenantAddr",
				"type": "address"
			}
		],
		"name": "NewRedeemable",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"internalType": "address",
				"name": "covenantAddr",
				"type": "address"
			}
		],
		"name": "NewLostAndFound",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"internalType": "address",
				"name": "covenantAddr",
				"type": "address"
			},
			{
				"internalType": "uint8",
				"name": "sourceType",
				"type": "uint8"
			}
		],
		"name": "Redeem",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"internalType": "address",
				"name": "covenantAddr",
				"type": "address"
			},
			{
				"internalType": "uint8",
				"name": "sourceType",
				"type": "uint8"
			}
		],
		"name": "Deleted",
		"type": "event"
	},
]
`)

func PackNewRedeemable(txid *big.Int, vout uint32, covenantAddress gethcmn.Address) []byte {
	return ABI.MustPack("NewRedeemable", txid, vout, covenantAddress)
}

func PackNewLostAndFound(txid *big.Int, vout uint32, covenantAddress gethcmn.Address) []byte {
	return ABI.MustPack("NewLostAndFound", txid, vout, covenantAddress)
}

func PackRedeem(txid *big.Int, vout uint32, covenantAddress gethcmn.Address, sourceType uint8) []byte {
	return ABI.MustPack("Redeem", txid, vout, covenantAddress, sourceType)
}
