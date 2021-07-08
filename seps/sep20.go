package seps

import (
	"math/big"

	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/smartbch/smartbch/internal/ethutils"
)

var (
	SEP206Addr = gethcmn.HexToAddress("0x0000000000000000000000000000000000002711")
)

var SEP20ABI = ethutils.MustParseABI(`
[
{
  "anonymous": false,
  "inputs": [
	{
	  "indexed": true,
	  "internalType": "address",
	  "name": "_owner",
	  "type": "address"
	},
	{
	  "indexed": true,
	  "internalType": "address",
	  "name": "_spender",
	  "type": "address"
	},
	{
	  "indexed": false,
	  "internalType": "uint256",
	  "name": "_value",
	  "type": "uint256"
	}
  ],
  "name": "Approval",
  "type": "event"
},
{
  "anonymous": false,
  "inputs": [
	{
	  "indexed": true,
	  "internalType": "address",
	  "name": "_from",
	  "type": "address"
	},
	{
	  "indexed": true,
	  "internalType": "address",
	  "name": "_to",
	  "type": "address"
	},
	{
	  "indexed": false,
	  "internalType": "uint256",
	  "name": "_value",
	  "type": "uint256"
	}
  ],
  "name": "Transfer",
  "type": "event"
},
{
  "inputs": [],
  "name": "name",
  "outputs": [
	{
	  "internalType": "string",
	  "name": "",
	  "type": "string"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [],
  "name": "symbol",
  "outputs": [
	{
	  "internalType": "string",
	  "name": "",
	  "type": "string"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [],
  "name": "decimals",
  "outputs": [
	{
	  "internalType": "uint8",
	  "name": "",
	  "type": "uint8"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [],
  "name": "totalSupply",
  "outputs": [
	{
	  "internalType": "uint256",
	  "name": "",
	  "type": "uint256"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [],
  "name": "owner",
  "outputs": [
	{
	  "internalType": "address",
	  "name": "",
	  "type": "address"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_owner",
	  "type": "address"
	}
  ],
  "name": "balanceOf",
  "outputs": [
	{
	  "internalType": "uint256",
	  "name": "balance",
	  "type": "uint256"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_to",
	  "type": "address"
	},
	{
	  "internalType": "uint256",
	  "name": "_value",
	  "type": "uint256"
	}
  ],
  "name": "transfer",
  "outputs": [
	{
	  "internalType": "bool",
	  "name": "success",
	  "type": "bool"
	}
  ],
  "stateMutability": "nonpayable",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_from",
	  "type": "address"
	},
	{
	  "internalType": "address",
	  "name": "_to",
	  "type": "address"
	},
	{
	  "internalType": "uint256",
	  "name": "_value",
	  "type": "uint256"
	}
  ],
  "name": "transferFrom",
  "outputs": [
	{
	  "internalType": "bool",
	  "name": "success",
	  "type": "bool"
	}
  ],
  "stateMutability": "nonpayable",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_spender",
	  "type": "address"
	},
	{
	  "internalType": "uint256",
	  "name": "_value",
	  "type": "uint256"
	}
  ],
  "name": "approve",
  "outputs": [
	{
	  "internalType": "bool",
	  "name": "success",
	  "type": "bool"
	}
  ],
  "stateMutability": "nonpayable",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_owner",
	  "type": "address"
	},
	{
	  "internalType": "address",
	  "name": "_spender",
	  "type": "address"
	}
  ],
  "name": "allowance",
  "outputs": [
	{
	  "internalType": "uint256",
	  "name": "remaining",
	  "type": "uint256"
	}
  ],
  "stateMutability": "view",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_spender",
	  "type": "address"
	},
	{
	  "internalType": "uint256",
	  "name": "_delta",
	  "type": "uint256"
	}
  ],
  "name": "increaseAllowance",
  "outputs": [
	{
	  "internalType": "bool",
	  "name": "success",
	  "type": "bool"
	}
  ],
  "stateMutability": "nonpayable",
  "type": "function"
},
{
  "inputs": [
	{
	  "internalType": "address",
	  "name": "_spender",
	  "type": "address"
	},
	{
	  "internalType": "uint256",
	  "name": "_delta",
	  "type": "uint256"
	}
  ],
  "name": "decreaseAllowance",
  "outputs": [
	{
	  "internalType": "bool",
	  "name": "success",
	  "type": "bool"
	}
  ],
  "stateMutability": "nonpayable",
  "type": "function"
}
]
`)

func PackSEP20Transfer(addr gethcmn.Address, val *big.Int) []byte {
	return SEP20ABI.MustPack("transfer", addr, val)
}
