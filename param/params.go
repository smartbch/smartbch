package param

//FILE: consensus configurable params collected here!
const (
	/**app consensus params**/
	BlockMaxBytes int64 = 24 * 1024 * 1024 // 24MB
	BlockMaxGas   int64 = 900_000_000_000

	/**ebp consensus params**/
	EbpExeRoundCount int = 200
	EbpRunnerNumber  int = 256
	EbpParallelNum   int = 32

	// gas limit for each transaction
	MaxTxGasLimit uint64 = 1000_0000

	/**staking consensus params**/
	StakingEpochSwitchDelay int64 = 3*80 + 40
	// reward params
	StakingEpochCountBeforeRewardMature int64  = 1
	StakingBaseProposerPercentage       uint64 = 15
	StakingExtraProposerPercentage      uint64 = 15

	// epoch params
	StakingMinVotingPercentPerEpoch        int   = 10 //10 percent in NumBlocksInEpoch, like 2016 / 10 = 201
	StakingMinVotingPubKeysPercentPerEpoch int   = 34 //34 percent in active validators,
	NumBlocksInEpoch                       int64 = 200
)
