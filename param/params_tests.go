//go:build params_testnet
// +build params_testnet

package param

//FILE: consensus configurable params collected here!
const (
	/**app consensus params**/
	BlockMaxBytes      int64  = 4 * 1024 * 1024 // 4MB
	BlockMaxGas        int64  = 1_000_000_000   //1Billion
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
	StakingNumBlocksInEpoch                int64 = 30
	StakingEpochSwitchDelay                int64 = 3*10 + 10
	MaxActiveValidatorCount                int   = 50

	// ccEpoch param
	BlocksInCCEpoch    int64 = 3
	CCEpochSwitchDelay int64 = 3*10 + 10

	// staking params
	OnlineWindowSize    int64  = 500
	MinOnlineSignatures int32  = 400
	SlashAmountDivisor  uint64 = 10

	// network params
	IsAmber                           bool  = false
	AmberBlocksInEpochAfterXHedgeFork int64 = 2016 * 10 * 60 / 6

	//fork params
	XHedgeContractSequence uint64 = 0xc94
	XHedgeForkBlock        int64  = 3088100
	ShaGateForkBlock       int64  = 80000000
	ShaGateSwitch          bool   = false
	StakingForkHeight      int64  = 80000000
)
