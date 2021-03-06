package staking

type ValidatorInfo struct {
	Pubkey         [32]byte
	NominatedCount int
}

const (
	NumBlocksInEpoch int64 = 2016
)

type BCHBlock struct {
	Height    int64
	Timestamp int64
	HashId    [32]byte
	ParentBlk [32]byte
	Validator *ValidatorInfo
}

type Epoch struct {
	StartHeight    int64
	EndTime        int64
	Duration       int64
	ValMapByPubkey map[[32]byte]*ValidatorInfo
}

type StakingStatus struct {
	LastEpochEndHeight     int64
	LatestFinalizedHeight  int64
	HashToBlock            map[[32]byte]*BCHBlock
	HeightToFinalizedBlock map[int64]*BCHBlock
	EpochList              []*Epoch
}

func (ss *StakingStatus) AddBlock(blk *BCHBlock) (missingBlockHash *[32]byte) {
	parent, ok := ss.HashToBlock[blk.ParentBlk]
	if !ok {
		return &blk.ParentBlk
	}
	for confirmCount := 1; confirmCount < 10; confirmCount++ {
		parent, ok = ss.HashToBlock[parent.ParentBlk]
		if !ok {
			panic("Blocken Chain")
		}
	}
	finalizedBlk, ok := ss.HeightToFinalizedBlock[parent.Height]
	if ok {
		if finalizedBlk == parent {
			return nil //nothing to do
		} else {
			panic("Deep Reorganization")
		}
	}
	ss.HeightToFinalizedBlock[parent.Height] = parent
	if ss.LatestFinalizedHeight+1 != parent.Height {
		panic("Height Skipped")
	}
	ss.LatestFinalizedHeight = parent.Height
	if ss.LatestFinalizedHeight-ss.LastEpochEndHeight == NumBlocksInEpoch {
		ss.AnalyzeNewEpoch()
	}
	return nil
}

func (ss *StakingStatus) AnalyzeNewEpoch() {
	epoch := &Epoch{
		StartHeight:    ss.LastEpochEndHeight + 1,
		ValMapByPubkey: make(map[[32]byte]*ValidatorInfo),
	}
	startTime := int64(1 << 62)
	for i := epoch.StartHeight; i <= ss.LatestFinalizedHeight; i++ {
		blk, ok := ss.HeightToFinalizedBlock[i]
		if !ok {
			panic("Missing Block")
		}
		if epoch.EndTime < blk.Timestamp {
			epoch.EndTime = blk.Timestamp
		}
		if startTime > blk.Timestamp {
			startTime = blk.Timestamp
		}
		if blk.Validator == nil {
			continue
		}
		if _, ok := epoch.ValMapByPubkey[blk.Validator.Pubkey]; !ok {
			epoch.ValMapByPubkey[blk.Validator.Pubkey] = blk.Validator
		}
		epoch.ValMapByPubkey[blk.Validator.Pubkey].NominatedCount++
	}
	epoch.Duration = epoch.EndTime - startTime
	if len(ss.EpochList) != 0 {
		lastEpoch := ss.EpochList[len(ss.EpochList)-1]
		epoch.Duration = epoch.EndTime - lastEpoch.EndTime
	}
	ss.EpochList = append(ss.EpochList, epoch)
	ss.LastEpochEndHeight = ss.LatestFinalizedHeight
}

func (ss *StakingStatus) ClearOldData() {
	elLen := len(ss.EpochList)
	if elLen == 0 {
		return
	}
	height := ss.EpochList[elLen-1].StartHeight
	height -= 5 * NumBlocksInEpoch
	for {
		blk, ok := ss.HeightToFinalizedBlock[height]
		if !ok {
			break
		}
		delete(ss.HeightToFinalizedBlock, height)
		delete(ss.HashToBlock, blk.HashId)
		height--
	}
}
