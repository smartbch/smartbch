package testutils

import (
	"bytes"
	"math/big"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

type App interface {
	abci.Application
	ChainID() *uint256.Int
	WaitLock()
	TestValidatorPubkey() crypto.PubKey
}

func MakeAndExecTxInBlock(_app App, height int64, privKey string,
	nonce uint64, toAddr gethcmn.Address, val int64, data []byte) *gethtypes.Transaction {
	txData := &gethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(0),
		Gas:      1000000,
		To:       &toAddr,
		Value:    big.NewInt(val),
		Data:     data,
	}
	tx := gethtypes.NewTx(txData)
	tx = MustSignTx(tx, _app.ChainID().ToBig(), privKey)
	ExecTxInBlock(_app, height, tx)
	return tx
}

func ExecTxInBlock(_app App, height int64, tx *gethtypes.Transaction) {
	_app.BeginBlock(abci.RequestBeginBlock{
		Header: tmproto.Header{
			Height:          height,
			Time:            time.Now(),
			ProposerAddress: _app.TestValidatorPubkey().Address(),
		},
	})
	if tx != nil {
		_app.DeliverTx(abci.RequestDeliverTx{
			Tx: mustEncodeTx(tx),
		})
	}
	_app.EndBlock(abci.RequestEndBlock{Height: height})
	_app.Commit()
	_app.WaitLock()

	_app.BeginBlock(abci.RequestBeginBlock{
		Header: tmproto.Header{Height: height + 1},
	})
	_app.DeliverTx(abci.RequestDeliverTx{})
	_app.EndBlock(abci.RequestEndBlock{Height: height + 1})
	_app.Commit()
	_app.WaitLock()
}

func mustEncodeTx(tx *gethtypes.Transaction) []byte {
	buf := &bytes.Buffer{}
	if err := tx.EncodeRLP(buf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
