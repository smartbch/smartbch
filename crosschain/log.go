package crosschain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	mevmtypes "github.com/smartbch/moeingevm/types"

	"github.com/smartbch/smartbch/crosschain/types"
)

func buildEvmLogWithTxidVoutAndAddress(eventHash [32]byte, txid [32]byte, vout uint32, covenantAddress common.Address) mevmtypes.EvmLog {
	evmLog := mevmtypes.EvmLog{
		Address: CCContractAddress,
		Topics:  make([]common.Hash, 0, 4),
	}
	evmLog.Topics = append(evmLog.Topics, eventHash)
	evmLog.Topics = append(evmLog.Topics, txid)
	evmLog.Topics = append(evmLog.Topics, uint256.NewInt(uint64(vout)).Bytes32())
	evmLog.Topics = append(evmLog.Topics, covenantAddress.Hash())
	return evmLog
}

func AddDataToEvmLog(log *mevmtypes.EvmLog, data []byte) *mevmtypes.EvmLog {
	log.Data = data
	return log
}

func buildNewRedeemable(txid [32]byte, vout uint32, covenantAddress common.Address) mevmtypes.EvmLog {
	return buildEvmLogWithTxidVoutAndAddress(HashOfEventNewRedeemable, txid, vout, covenantAddress)
}

func buildNewLostAndFound(txid [32]byte, vout uint32, covenantAddress common.Address) mevmtypes.EvmLog {
	return buildEvmLogWithTxidVoutAndAddress(HashOfEventNewLostAndFound, txid, vout, covenantAddress)
}

func buildConvert(prevTxid [32]byte, prevVout uint32, newCovenantAddress common.Address) mevmtypes.EvmLog {
	return buildEvmLogWithTxidVoutAndAddress(HashOfEventConvert, prevTxid, prevVout, newCovenantAddress)
}

func buildRedeemLog(txid [32]byte, vout uint32, covenantAddress common.Address, sourceType types.SourceType) mevmtypes.EvmLog {
	log := buildEvmLogWithTxidVoutAndAddress(HashOfEventRedeem, txid, vout, covenantAddress)
	o := uint256.NewInt(uint64(sourceType)).Bytes32()
	AddDataToEvmLog(&log, o[:])
	return log
}

func buildDeletedLog(txid [32]byte, vout uint32, covenantAddress common.Address, sourceType types.SourceType) mevmtypes.EvmLog {
	log := buildEvmLogWithTxidVoutAndAddress(HashOfEventDeleted, txid, vout, covenantAddress)
	o := uint256.NewInt(uint64(sourceType)).Bytes32()
	AddDataToEvmLog(&log, o[:])
	return log
}

//event ChangeAddr(uint256 prevTxid, uint32 prevVout, address newCovenantAddr, uint256 txid, uint32 vout)
func buildChangeAddrLog(prevTxid [32]byte, prevVout uint32, newCovenantAddr common.Address, txid [32]byte, vout uint32) mevmtypes.EvmLog {
	log := buildEvmLogWithTxidVoutAndAddress(HashOfEventChangeAddr, prevTxid, prevVout, newCovenantAddr)
	o := uint256.NewInt(uint64(vout)).Bytes32()
	AddDataToEvmLog(&log, append(txid[:], o[:]...))
	return log
}
