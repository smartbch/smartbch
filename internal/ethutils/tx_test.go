package ethutils_test

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
)

func TestTxSig(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	_, addr2 := testutils.GenKeyAndAddr()

	chainIDEpoch := big.NewInt(1)

	tx := ethutils.NewTx(123, &addr2, big.NewInt(100), 100000, big.NewInt(1), nil)
	tx = testutils.MustSignTx(tx, chainIDEpoch, key1)

	signer := types.NewEIP155Signer(chainIDEpoch)
	sender, err := signer.Sender(tx)
	require.NoError(t, err)
	require.Equal(t, addr1, sender)

	txBytes, err := ethutils.EncodeTx(tx)
	require.NoError(t, err)

	tx2 := &types.Transaction{}
	err = tx2.DecodeRLP(rlp.NewStream(bytes.NewReader(txBytes), 0))
	require.NoError(t, err)
	require.Equal(t, tx.Value(), tx2.Value())
	require.Equal(t, tx.Hash(), tx2.Hash())

	sender, err = signer.Sender(tx2)
	require.NoError(t, err)
	require.Equal(t, addr1, sender)
}

func TestTxSig2(t *testing.T) {
	txBytes, err := hex.DecodeString("f892108609184e72a0008276c09445e063622e94424963c3fe4c354005bd9b28eb60849184e72aa9d46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f07244567525a03ea71b7c07cb4f56c50948c6937ea46a03868f08abb42973425fe80f8c4aa344a0217655357cfcc0c9ce52be739f6a12093a78cdadf040a9c60e16b7c689fea87c")
	require.NoError(t, err)

	tx := &types.Transaction{}
	err = tx.DecodeRLP(rlp.NewStream(bytes.NewReader(txBytes), 0))
	require.NoError(t, err)

	signer := types.NewEIP155Signer(big.NewInt(1))
	sender, err := signer.Sender(tx)
	require.NoError(t, err)
	require.Equal(t, "0xFaD1182406c4456c84148F6A679EF97E1d321958", sender.Hex())
}
