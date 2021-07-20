// +build params_testnet

package param

//FILE: consensus configurable params collected here!
const (
	/**app consensus params**/
	BlockMaxBytes int64 = 4 * 1024 * 1024 // 4MB
	BlockMaxGas   int64 = 1_000_000_000   //1Billion

	/**ebp consensus params**/
	EbpExeRoundCount int = 200
	EbpRunnerNumber  int = 256
	EbpParallelNum   int = 32

	// gas limit for each transaction
	MaxTxGasLimit uint64 = 1000_0000

	/**staking consensus params**/
	// reward params
	StakingEpochCountBeforeRewardMature int64  = 1
	StakingBaseProposerPercentage       uint64 = 15
	StakingExtraProposerPercentage      uint64 = 15

	// epoch params
	StakingMinVotingPercentPerEpoch        int   = 10 //10 percent in StakingNumBlocksInEpoch, like 2016 / 10 = 201
	StakingMinVotingPubKeysPercentPerEpoch int   = 34 //34 percent in active validators,
	StakingNumBlocksInEpoch                int64 = 30
	StakingEpochSwitchDelay                int64 = 3*10 + 10
	StakingMaxValidatorCount               int   = 50
)
