package crosschain

import (
	"math/big"

	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/smartbch/smartbch/internal/ethutils"
)

/*
// SPDX-License-Identifier: Apache-2.0
pragma solidity >=0.8.0;


contract CC {
    event NewRedeemable(uint256 indexed txid, uint32 indexed vout, address indexed covenantAddr);
    event NewLostAndFound(uint256 indexed txid, uint32 indexed vout, address indexed covenantAddr);
    event Redeem(uint256 indexed txid, uint32 indexed vout, address indexed covenantAddr, uint8 sourceType);
    event ChangeAddr(address indexed oldCovenantAddr, address indexed newCovenantAddr);
    event Convert(uint256 indexed prevTxid, uint32 indexed prevVout, address indexed oldCovenantAddr, uint256 txid, uint32 vout, address newCovenantAddr);
    event Deleted(uint256 indexed txid, uint32 indexed vout, address indexed covenantAddr, uint8 sourceType);

    function redeem(uint256 txid, uint256 index, address targetAddress) external {}
    function startRescan(uint256 mainFinalizedBlockHeight) external {}
    function pause() external {}
    function resume() external {}
    function handleUTXOs() external {}
}
*/

var ABI = ethutils.MustParseABI(`
[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "address",
				"name": "oldCovenantAddr",
				"type": "address"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "newCovenantAddr",
				"type": "address"
			}
		],
		"name": "ChangeAddr",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "uint256",
				"name": "prevTxid",
				"type": "uint256"
			},
			{
				"indexed": true,
				"internalType": "uint32",
				"name": "prevVout",
				"type": "uint32"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "oldCovenantAddr",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"indexed": false,
				"internalType": "address",
				"name": "newCovenantAddr",
				"type": "address"
			}
		],
		"name": "Convert",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"indexed": true,
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "covenantAddr",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint8",
				"name": "sourceType",
				"type": "uint8"
			}
		],
		"name": "Deleted",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"indexed": true,
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"indexed": true,
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
				"indexed": true,
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"indexed": true,
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"indexed": true,
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
				"indexed": true,
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"indexed": true,
				"internalType": "uint32",
				"name": "vout",
				"type": "uint32"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "covenantAddr",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint8",
				"name": "sourceType",
				"type": "uint8"
			}
		],
		"name": "Redeem",
		"type": "event"
	},
	{
		"inputs": [],
		"name": "handleUTXOs",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "pause",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "txid",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "index",
				"type": "uint256"
			},
			{
				"internalType": "address",
				"name": "targetAddress",
				"type": "address"
			}
		],
		"name": "redeem",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "resume",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "mainFinalizedBlockHeight",
				"type": "uint256"
			}
		],
		"name": "startRescan",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]
`)

func PackRedeemFunc(txid, index *big.Int, targetAddress gethcmn.Address) []byte {
	return ABI.MustPack("redeem", txid, index, targetAddress)
}

func PackStartRescanFunc(mainFinalizedBlockHeight *big.Int) []byte {
	return ABI.MustPack("startRescan", mainFinalizedBlockHeight)
}

func PackPauseFunc() []byte {
	return ABI.MustPack("pause")
}

func PackHandleUTXOsFunc() []byte {
	return ABI.MustPack("handleUTXOs")
}
