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
	StartMainnetHeightForCC = 1531570 // mainnet height which cc tx collected from
	StartEpochNumberForCC   = 0       // epoch number which cc enabled from
	AlreadyBurntOnMainChain = 0       // BCH already burnt on main chain when cc enabled
	GenesisCovenantAddress  = "0x6ad3f81523c87aa17f1dfa08271cf57b6277c98e"
	MonitorElectionEpochs   = 2 // For test
	OperatorElectionEpochs  = 2 // For test

	OperatorsGovSequence    = 0x130 // TODO: change this in production mode
	MonitorsGovSequence     = 0x1b5 // TODO: change this in production mode
	OperatorMinStakedBCH    = 1     // TODO: change this in production mode
	MonitorMinStakedBCH     = 1     // TODO: change this in production mode
	MonitorMinOpsNomination = 6     // TODO: change this in production mode
	MonitorMinPowNomination = 0     // TODO: change this in production mode
	OperatorsCount          = 10
	OperatorsMaxChangeCount = 3
	MonitorsCount           = 3
	MonitorsMaxChangeCount  = 1

	// cc covenant params
	RedeemScriptWithoutConstructorArgs = `0x5279009c635a795c797e5d797e5e797e5f797e60797e0111797e0112797e0113797e0114797ea97b8800537a717c567a577a587a597a575b7a5c7a5d7a5e7a5f7a607a01117a01127a01137a01147a5aafc3519dc4519d00cc00c602204e94a2695279827700a05479827700a09b635279827701149d5379827701149d011454797e01147e53797ec1012a7f777e02a91478a97e01877e00cd78886d686d6d51677b519d547956797e57797ea98800727c52557a567a577a53afc0009d00cc00c69d03008700b27501147b7ec101157f777e02a9147ca97e01877e00cd877768`
	MinOperatorSigCount                = 7
	MinMonitorSigCount                 = 2
	RedeemOrCovertMinerFee             = 2000
	MonitorTransferWaitBlocks          = 34560 // 6 * 24 * 30 * 8 ~= 8 months
	CcBchNetwork                       = "testnet3"

	// epoch params
	StakingMinVotingPercentPerEpoch        int   = 10 //10 percent in StakingNumBlocksInEpoch, like 2016 / 10 = 201
	StakingMinVotingPubKeysPercentPerEpoch int   = 34 //34 percent in active validators,
	StakingNumBlocksInEpoch                int64 = 60
	StakingEpochSwitchDelay                int64 = 10 * 60 * 3 // 5% time of an epoch
	MaxActiveValidatorCount                int   = 50

	// network params
	IsAmber                           bool  = false
	AmberBlocksInEpochAfterXHedgeFork int64 = 2016 * 10 * 60 / 6

	// fork params
	XHedgeContractSequence uint64 = 0x13311
	XHedgeForkBlock        int64  = 4106000
	ShaGateForkBlock       int64  = 2
	ShaGateSwitch          bool   = false
)
