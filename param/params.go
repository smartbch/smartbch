//go:build !params_testnet && !params_amber
// +build !params_testnet,!params_amber

package param

//FILE: consensus configurable params collected here!
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

	// cc params
	StartMainnetHeightForCC     = 10000000 // mainnet height which cc tx collected from
	StartEpochNumberForCC       = 300      // epoch number which cc enabled from
	AlreadyBurntOnMainChain     = 100      // BCH already burnt on main chain when cc enabled
	GenesisCovenantAddress      = "0x1234"
	MonitorElectionEpochs       = 12 //must be divisible by OperatorElectionEpochs
	OperatorElectionEpochs      = 4
	MaxMonitorNumber        int = 3
	OperatorsGovSequence        = 0 // TODO
	MonitorsGovSequence         = 0 // TODO
	OperatorMinStakedBCH        = 10000
	MonitorMinStakedBCH         = 100000
	OperatorsCount              = 10
	OperatorsMaxChangeCount     = 3
	MonitorsCount               = 3
	MonitorsMaxChangeCount      = 1

	// cc covenant params
	RedeemScriptWithoutConstructorArgs = `0x` // TODO
	MinOperatorSigCount                = 7
	MinMonitorSigCount                 = 2
	RedeemOrCovertMinerFee             = 2000
	MonitorTransferWaitBlocks          = 34560 // 6 * 24 * 30 * 8 ~= 8 months
	CcBchNetwork                       = "mainnet"

	// epoch params
	StakingMinVotingPercentPerEpoch        int   = 10 //10 percent in StakingNumBlocksInEpoch, like 2016 / 10 = 201
	StakingMinVotingPubKeysPercentPerEpoch int   = 34 //34 percent in active validators,
	StakingNumBlocksInEpoch                int64 = 2016
	StakingEpochSwitchDelay                int64 = 600 * 2016 / 20 // 5% time of an epoch
	MaxActiveValidatorCount                int   = 50

	// network params
	IsAmber                           bool  = false
	AmberBlocksInEpochAfterXHedgeFork int64 = 2016 * 10 * 60 / 6

	// fork params
	XHedgeContractSequence uint64 = 0x13311
	XHedgeForkBlock        int64  = 4106000
	ShaGateForkBlock       int64  = 80000000
	ShaGateSwitch          bool   = false
)
