package crosschain

import (
	"github.com/tendermint/tendermint/libs/log"

	mevmtypes "github.com/smartbch/moeingevm/types"

	"github.com/smartbch/smartbch/crosschain/types"
)

func HandleMonitorVoteInfos(ctx *mevmtypes.Context, blockTime int64, infos []*types.MonitorVoteInfo, logger log.Logger) {
	var pubkeyVoteMap = make(map[[33]byte]int64)
	for _, info := range infos {
		for _, n := range info.Nominations {
			if _, ok := pubkeyVoteMap[n.Pubkey]; !ok {
				pubkeyVoteMap[n.Pubkey] = n.NominatedCount
				continue
			}
			pubkeyVoteMap[n.Pubkey] += n.NominatedCount
		}
	}
	handleMonitorInfos(ctx, pubkeyVoteMap, blockTime, logger)
}

func handleMonitorInfos(ctx *mevmtypes.Context, pubkeyVoteMap map[[33]byte]int64, blockTime int64, logger log.Logger) {
	ElectMonitors(ctx, pubkeyVoteMap, blockTime, logger)
}
