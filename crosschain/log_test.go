package crosschain

import (
	"github.com/smartbch/smartbch/crosschain/types"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

func TestBuildNewRedeemable(t *testing.T) {
	txid := [32]byte{0x1}
	vout := uint32(1)
	address := common.Address{0x1}
	log := buildNewRedeemable(txid, vout, address)

	require.Equal(t, 4, len(log.Topics))
	require.Equal(t, HashOfEventNewRedeemable, [32]byte(log.Topics[0]))
	require.Equal(t, txid, [32]byte(log.Topics[1]))
	require.Equal(t, vout, uint32(uint256.NewInt(0).SetBytes32(log.Topics[2].Bytes()).Uint64()))
	require.Equal(t, address.Hash(), log.Topics[3])
}

func TestBuildRedeemLog(t *testing.T) {
	txid := [32]byte{0x1}
	vout := uint32(1)
	address := common.Address{0x1}
	log := buildRedeemLog(txid, vout, address, types.FromBurnRedeem)
	require.Equal(t, 4, len(log.Topics))
	require.Equal(t, HashOfEventRedeem, [32]byte(log.Topics[0]))
	require.Equal(t, txid, [32]byte(log.Topics[1]))
	require.Equal(t, vout, uint32(uint256.NewInt(0).SetBytes32(log.Topics[2].Bytes()).Uint64()))
	require.Equal(t, address.Hash(), log.Topics[3])
	require.Equal(t, uint64(types.FromBurnRedeem), uint256.NewInt(0).SetBytes(log.Data).Uint64())
}

func TestBuildChangeAddrLog(t *testing.T) {
	prevTxid := [32]byte{0x02}
	prevVout := uint32(2)
	address := common.Address{0x1}
	txid := [32]byte{0x1}
	vout := uint32(1)
	log := buildChangeAddrLog(prevTxid, prevVout, address, txid, vout)
	require.Equal(t, 4, len(log.Topics))
	require.Equal(t, HashOfEventChangeAddr, [32]byte(log.Topics[0]))
	require.Equal(t, prevTxid, [32]byte(log.Topics[1]))
	require.Equal(t, prevVout, uint32(uint256.NewInt(0).SetBytes32(log.Topics[2].Bytes()).Uint64()))
	require.Equal(t, address.Hash(), log.Topics[3])
	o := uint256.NewInt(uint64(vout)).Bytes32()
	require.Equal(t, append(txid[:], o[:]...), log.Data)
}
