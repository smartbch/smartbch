package staking

import (
	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/smartbch/smartbch/internal/ethutils"
)

var ABI = ethutils.MustParseABI(`
[
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "rewardTo",
				"type": "address"
			},
			{
				"internalType": "bytes32",
				"name": "introduction",
				"type": "bytes32"
			},
			{
				"internalType": "bytes32",
				"name": "pubkey",
				"type": "bytes32"
			}
		],
		"name": "createValidator",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "decreaseMinGasPrice",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "rewardTo",
				"type": "address"
			},
			{
				"internalType": "bytes32",
				"name": "introduction",
				"type": "bytes32"
			}
		],
		"name": "editValidator",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "increaseMinGasPrice",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "retire",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
    {
      "inputs": [
        {
          "internalType": "address[]",
          "name": "addrList",
          "type": "address[]"
        }
      ],
      "name": "sumVotingPower",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "summedPower",
          "type": "uint256"
        },
        {
          "internalType": "uint256",
          "name": "totalPower",
          "type": "uint256"
        }
      ],
      "stateMutability": "nonpayable",
      "type": "function"
    }
]
`)

func PackCreateValidator(rewardTo gethcmn.Address, intro [32]byte, pubKey [32]byte) []byte {
	return ABI.MustPack("createValidator", rewardTo, intro, pubKey)
}
func PackEditValidator(rewardTo gethcmn.Address, intro [32]byte) []byte {
	return ABI.MustPack("editValidator", rewardTo, intro)
}
func PackRetire() []byte {
	return ABI.MustPack("retire")
}
func PackIncreaseMinGasPrice() []byte {
	return ABI.MustPack("increaseMinGasPrice")
}
func PackDecreaseMinGasPrice() []byte {
	return ABI.MustPack("decreaseMinGasPrice")
}
func PackSumVotingPower(addrList []gethcmn.Address) []byte {
	return ABI.MustPack("sumVotingPower", addrList)
}
