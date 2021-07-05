package param

import "github.com/holiman/uint256"

//FILE: consensus configurable params collected here!
var (
	/**app consensus params**/
	BlockMaxBytes int64 = 24 * 1024 * 1024 // 24MB
	BlockMaxGas   int64 = 900_000_000_000

	/**ebp consensus params**/
	EbpExeRoundCount = 200
	EbpRunnerNumber  = 256
	EbpParallelNum   = 32

	/**staking consensus params**/
	StakingEpochSwitchDelay int64 = 3*80 + 40
	// reward params
	StakingEpochCountBeforeRewardMature int64 = 1
	StakingBaseProposerPercentage             = uint256.NewInt().SetUint64(15)
	StakingExtraProposerPercentage            = uint256.NewInt().SetUint64(15)
	// epoch params
	StakingMinVotingPercentPerEpoch              = 10 //10 percent in NumBlocksInEpoch, like 2016 / 10 = 201
	StakingMinVotingPubKeysPercentPerEpoch       = 34 //34 percent in active validators,
	WatcherNumBlocksInEpoch                int64 = 200
	WatcherNumBlocksToClearMemory          int64 = 1000
	WatcherWaitingBlockDelayTime           int64 = 2
)
