package staking

import (
	"bytes"
	"math/big"
	"sort"
	"strings"

	"github.com/holiman/uint256"

	mevmtypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/param"
	"github.com/smartbch/smartbch/staking/types"
)

var (
	SlotValidatorsMap   string = strings.Repeat(string([]byte{0}), 31) + string([]byte{134})
	SlotValidatorsArray string = strings.Repeat(string([]byte{0}), 31) + string([]byte{135})

	CoindayUnit *uint256.Int = uint256.NewInt(0).Mul(uint256.NewInt(24*60*60), uint256.NewInt(Uint64_1e18))
)

func GetAndClearPosVotes(ctx *mevmtypes.Context, xHedgeContractSeq uint64) map[[32]byte]int64 {
	//get validator operator pubkeys in xHedge contract
	validators := ctx.GetDynamicArray(xHedgeContractSeq, SlotValidatorsArray)
	posVotes := make(map[[32]byte]int64, len(validators))
	var pubkey [32]byte
	coindays := uint256.NewInt(0)
	for _, val := range validators {
		copy(pubkey[:], val)
		coindaysBz := ctx.GetAndDeleteValueAtMapKey(xHedgeContractSeq, SlotValidatorsMap, string(val))
		coindays.SetBytes(coindaysBz)
		coindays.Div(coindays, CoindayUnit)
		if ctx.IsXHedgeFork() && ((param.IsAmber && ctx.Height >= 3600000) || !param.IsAmber) {
			if !coindays.IsZero() {
				posVotes[pubkey] = int64(coindays.Uint64())
			}
		} else {
			posVotes[pubkey] = int64(coindays.Uint64())
		}
	}
	ctx.DeleteDynamicArray(xHedgeContractSeq, SlotValidatorsArray)
	return posVotes
}

func GetPosVotes(ctx *mevmtypes.Context, xhedgeContractSeq uint64) map[[32]byte]*big.Int {
	validators := ctx.GetDynamicArray(xhedgeContractSeq, SlotValidatorsArray)
	posVotes := make(map[[32]byte]*big.Int, len(validators))
	var pubkey [32]byte
	coindays := uint256.NewInt(0)
	for _, val := range validators {
		copy(pubkey[:], val)
		coindaysBz := ctx.GetValueAtMapKey(xhedgeContractSeq, SlotValidatorsMap, string(val))
		coindays.SetBytes(coindaysBz)
		//coindays.Div(coindays, CoindayUnit)
		posVotes[pubkey] = coindays.ToBig()
	}
	return posVotes
}

func CreateInitVotes(ctx *mevmtypes.Context, xhedgeContractSeq uint64, activeValidators []*types.Validator) {
	pubkeys := make([][]byte, 0, len(activeValidators))
	for _, v := range activeValidators {
		key := make([]byte, 32)
		copy(key, v.Pubkey[:])
		pubkeys = append(pubkeys, key)
	}
	sort.Slice(pubkeys, func(i, j int) bool {
		return bytes.Compare(pubkeys[i][:], pubkeys[j][:]) < 0
	})
	oneBz := uint256.NewInt(1).PaddedBytes(32)
	for _, key := range pubkeys { // each has a minimum voting power
		ctx.SetValueAtMapKey(xhedgeContractSeq, SlotValidatorsMap, string(key), oneBz)
	}
	ctx.CreateDynamicArray(xhedgeContractSeq, SlotValidatorsArray, pubkeys)
}
