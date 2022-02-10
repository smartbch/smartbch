//go:build params_amber
// +build params_amber

package param

//FILE: consensus configurable params collected here!
const (
	/**app consensus params**/
	BlockMaxBytes      int64  = 4 * 1024 * 1024 // 4MB
	BlockMaxGas        int64  = 1_000_000_000   //1Billion
	DefaultMinGasPrice uint64 = 1_000_000_000   // 1gwei

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
	StakingNumBlocksInEpoch                int64 = 2016
	StakingEpochSwitchDelay                int64 = 9 * 2016 / 20 // 5% time of an epoch
	StakingMaxValidatorCount               int   = 50

	// ccEpoch params
	BlocksInCCEpoch    int64 = 7
	CCEpochSwitchDelay int64 = 3 * 20 / 20

	// network params
	IsAmber                           bool  = true
	AmberBlocksInEpochAfterXHedgeFork int64 = 1000

	// fork params
	XHedgeContractSequence uint64 = 3774
	XHedgeForkBlock        int64  = 2766853
	ShaGateForkBlock       int64  = 80000000
	ShaGateSwitch          bool   = false
)
