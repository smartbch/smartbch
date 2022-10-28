package crosschain

import (
	"bytes"
	"sort"

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
	// 1. sort pubkey vote map
	var infos = make([]*types.Nomination, 0, len(pubkeyVoteMap))
	for k, v := range pubkeyVoteMap {
		infos = append(infos, &types.Nomination{
			Pubkey:         k,
			NominatedCount: v,
		})
	}
	SortMonitorVoteNominations(infos)
	//if len(infos) > param.MaxMonitorNumber {
	//	infos = infos[:param.MaxMonitorNumber]
	//}
	// 2. set the monitor info to vote contract
	ElectMonitors(ctx, infos, blockTime, logger)
}

func SortMonitorVoteNominations(nominations []*types.Nomination) {
	sort.Slice(nominations, func(i, j int) bool {
		return bytes.Compare(nominations[i].Pubkey[:], nominations[j].Pubkey[:]) < 0
	})
	sort.SliceStable(nominations, func(i, j int) bool {
		return nominations[i].NominatedCount > nominations[j].NominatedCount
	})
}
