package staking

import (
	"math/big"

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
    },
	{
		"inputs": [],
		"name": "executeProposal",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "validator",
				"type": "address"
			}
		],
		"name": "getVote",
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
		"inputs": [
			{
				"internalType": "uint256",
				"name": "target",
				"type": "uint256"
			}
		],
		"name": "proposal",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "target",
				"type": "uint256"
			}
		],
		"name": "vote",
		"outputs": [],
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
func PackProposal(target *big.Int) []byte {
	return ABI.MustPack("proposal", target)
}
func PackVote(target *big.Int) []byte {
	return ABI.MustPack("vote", target)
}
func PackExecuteProposal() []byte {
	return ABI.MustPack("executeProposal")
}
func PackGetVote(validator gethcmn.Address) []byte {
	return ABI.MustPack("getVote", validator)
}

func PackSumVotingPower(addrList []gethcmn.Address) []byte {
	return ABI.MustPack("sumVotingPower", addrList)
}
func UnpackSumVotingPowerReturnData(data []byte) (*big.Int, *big.Int) {
	ret := ABI.MustUnpack("sumVotingPower", data)
	return ret[0].(*big.Int), ret[1].(*big.Int)
}
