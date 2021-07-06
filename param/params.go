package param

//FILE: consensus configurable params collected here!
const (
	/**app consensus params**/
	BlockMaxBytes int64 = 24 * 1024 * 1024 // 24MB
	BlockMaxGas   int64 = 900_000_000_000

	/**ebp consensus params**/
	EbpExeRoundCount = 200
	EbpRunnerNumber  = 256
	EbpParallelNum   = 32

	// gas limit for each transaction
	MaxTxGasLimit = 1000_0000

	/**staking consensus params**/
	StakingEpochSwitchDelay int64 = 3*80 + 40
	// reward params
	StakingEpochCountBeforeRewardMature int64 = 1
	StakingBaseProposerPercentage             = 15
	StakingExtraProposerPercentage            = 15

	// epoch params
	StakingMinVotingPercentPerEpoch              = 10 //10 percent in NumBlocksInEpoch, like 2016 / 10 = 201
	StakingMinVotingPubKeysPercentPerEpoch       = 34 //34 percent in active validators,
	NumBlocksInEpoch                       int64 = 200
)
