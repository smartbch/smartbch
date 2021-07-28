package freegas

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingads/indextree"
	mevmtypes "github.com/smartbch/moeingevm/types"
)

type FreeGasKeeper struct {
	dirName string
	dbName  string
	rdb     *indextree.RocksDB
	logger  log.Logger
}

func NewKeeper(dirName string, currTime int64, logger log.Logger) *FreeGasKeeper {
	keeper := &FreeGasKeeper{dirName: dirName, logger: logger}
	keeper.tryOpenDB(dateStrFromTimestamp(currTime))
	return keeper
}

func (keeper *FreeGasKeeper) ProcessCommittedTxs(timestamp int64, committedTxs []*mevmtypes.Transaction) {
	if len(committedTxs) == 0 {
		return
	}
	keeper.beginBlock(timestamp)
	for _, tx := range committedTxs {
		keeper.deductGas(tx.From, tx.GasUsed)
	}
	keeper.endBlock()
}

func (keeper *FreeGasKeeper) GetRemainedGas(addr [20]byte) uint64 {
	if keeper.rdb == nil {
		return 0
	}
	return keeper.getRemainedGas(addr)
}

func (keeper *FreeGasKeeper) getRemainedGas(addr [20]byte) uint64 {
	gasBz := keeper.rdb.Get(addr[:])
	if len(gasBz) == 0 {
		return 0
	}
	return binary.LittleEndian.Uint64(gasBz[:])
}

func (keeper *FreeGasKeeper) tryOpenDB(dbName string) {
	dbPath := path.Join(keeper.dirName, dbName)
	keeper.logger.Info("trying to open GasDB", "dbPath", dbPath)
	if !exists(dbPath) { //today's db is not ready yet
		return
	}
	rocksdb, err := indextree.NewRocksDB(dbName, keeper.dirName)
	if err == nil {
		if keeper.rdb != nil {
			keeper.rdb.Close()
		}
		os.RemoveAll(path.Join(keeper.dirName, keeper.dbName)) //remove yesterday's db
		keeper.dbName = dbName
		keeper.rdb = rocksdb
	} else {
		keeper.logger.Error("Error in open rocksdb")
	}
}

func (keeper *FreeGasKeeper) beginBlock(timestamp int64) {
	dbName := dateStrFromTimestamp(timestamp)
	if keeper.dbName != dbName {
		keeper.tryOpenDB(dbName)
	}
	if keeper.rdb != nil {
		keeper.rdb.OpenNewBatch()
	}
}

func (keeper *FreeGasKeeper) endBlock() {
	if keeper.rdb != nil {
		keeper.rdb.CloseOldBatch()
	}
}

func (keeper *FreeGasKeeper) deductGas(addr [20]byte, gas uint64) {
	if keeper.rdb == nil {
		return
	}
	remainedGas := keeper.getRemainedGas(addr)
	if remainedGas > gas {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], remainedGas-gas)
		keeper.rdb.CurrBatch().Set(addr[:], buf[:])
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dateStrFromTimestamp(timestamp int64) string {
	t := time.Unix(timestamp, 0)
	return fmt.Sprintf("%04d%02d%02d", t.Year(), t.Month(), t.Day())
}
