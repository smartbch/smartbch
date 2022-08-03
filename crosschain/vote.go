package crosschain

import (
	"bytes"
	"sort"

	mevmtypes "github.com/smartbch/moeingevm/types"

	"github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/param"
)

func HandleMonitorVoteInfo(ctx *mevmtypes.Context, info *types.MonitorVoteInfo, blockTime int64) {
	SaveMonitorVoteInfo(ctx, *info)
	if info.Number%param.EpochNumbersPerCCEpoch != 0 {
		return
	}
	var pubkeyVoteMap = make(map[[33]byte]int64)
	for i := info.Number - param.EpochStartHeightForCC; i < info.Number; i++ {
		voteInfo := LoadMonitorVoteInfo(ctx, i)
		if voteInfo == nil {
			panic("should have vote info here")
		}
		for _, n := range voteInfo.Nominations {
			if _, ok := pubkeyVoteMap[n.Pubkey]; !ok {
				pubkeyVoteMap[n.Pubkey] = n.NominatedCount
				continue
			}
			pubkeyVoteMap[n.Pubkey] += n.NominatedCount
		}
	}
	handleMonitorInfos(ctx, pubkeyVoteMap, blockTime)
}

func handleMonitorInfos(ctx *mevmtypes.Context, pubkeyVoteMap map[[33]byte]int64, blockTime int64) {
	// 1. sort pubkey vote map
	var infos = make([]*types.Nomination, 0, len(pubkeyVoteMap))
	for k, v := range pubkeyVoteMap {
		infos = append(infos, &types.Nomination{
			Pubkey:         k,
			NominatedCount: v,
		})
	}
	SortMonitorVoteNominations(infos)
	if len(infos) > param.MaxMonitorNumber {
		infos = infos[:param.MaxMonitorNumber]
	}
	// 2. set the monitor info to vote contract
	monitors := ReadMonitorArr(ctx, MonitorsGovSeq)
	for idx, monitor := range monitors {
		WriteMonitorElectedTime(ctx, MonitorsGovSeq, uint64(idx), 0)
		for _, info := range infos {
			if bytes.Equal(info.Pubkey[:], monitor.Pubkey) {
				WriteMonitorElectedTime(ctx, MonitorsGovSeq, uint64(idx), uint64(blockTime))
				break
			}
		}
	}
	WriteMonitorsLastElectionTime(ctx, MonitorsGovSeq, uint64(blockTime))
}

func SortMonitorVoteNominations(nominations []*types.Nomination) {
	sort.Slice(nominations, func(i, j int) bool {
		return bytes.Compare(nominations[i].Pubkey[:], nominations[j].Pubkey[:]) < 0
	})
	sort.SliceStable(nominations, func(i, j int) bool {
		return nominations[i].NominatedCount > nominations[j].NominatedCount
	})
}
