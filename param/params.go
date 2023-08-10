//go:build !params_testnet && !params_amber
// +build !params_testnet,!params_amber

package param

import "math"

// FILE: consensus configurable params collected here!
const (
	/**app consensus params**/
	BlockMaxBytes      int64  = 4 * 1024 * 1024 // 4MB
	BlockMaxGas        int64  = 1_000_000_000   // 1Billion
	DefaultMinGasPrice uint64 = 10_000_000_000  // 10gwei

	/**ebp consensus params**/
	EbpExeRoundCount int = 200
	EbpRunnerNumber  int = 256
	EbpParallelNum   int = 32

	// gas limit for each transaction
	MaxTxGasLimit uint64 = 1000_0000

	/**staking consensus params**/
	// reward params
	EpochCountBeforeRewardMature  int64  = 1
	ProposerBaseMintFeePercentage uint64 = 15
	CollectorMintFeePercentage    uint64 = 15

	// epoch params
	StakingMinVotingPercentPerEpoch        int   = 10 //10 percent in StakingNumBlocksInEpoch, like 2016 / 10 = 201
	StakingMinVotingPubKeysPercentPerEpoch int   = 34 //34 percent in active validators,
	StakingNumBlocksInEpoch                int64 = 2016
	StakingEpochSwitchDelay                int64 = 600 * 2016 / 20 // 5% time of an epoch
	MaxActiveValidatorCount                int   = 50
	BlocksInEpochAfterStakingFork          int64 = 12 * 10 * 60 / 6 // 2h

	// ccEpoch params:
	BlocksInCCEpoch    int64 = 7
	CCEpochSwitchDelay int64 = 3 * 20 / 20

	// staking and slash params
	ValidatorWatchWindowSize       int64  = 50
	ValidatorWatchMinSignatures    int32  = 10
	VotingPowerDivider             int64  = 10
	OnlineWindowSize               int64  = 300 // 0.5h = 1 * 3600s / 6s
	MinOnlineSignatures            int32  = 180 // OnlineWindowSize * 0.6
	NotOnlineSlashAmountDivisor    uint64 = 16  // 1/16 MinimumStakingAmountAfterStakingFork
	DuplicateSigSlashAMountDivisor uint64 = 4   // 1/4 MinimumStakingAmountAfterStakingFork
	SlashReceiver                  string = "0xad114243D2D61b78F76D63C1Fef6709219b2cd22"

	// network params
	IsAmber                           bool  = false
	AmberBlocksInEpochAfterXHedgeFork int64 = 2016 * 10 * 60 / 6

	// fork params
	XHedgeContractSequence uint64 = 0x13311
	XHedgeForkBlock        int64  = 4106000
	ShaGateForkBlock       int64  = math.MaxInt64
	ShaGateSwitch          bool   = false
	StakingForkHeight      int64  = 10870248 // near 202308071610
)
