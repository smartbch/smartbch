package testutils

import (
	"bytes"

	gethtypes "github.com/ethereum/go-ethereum/core/types"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

type App interface {
	abci.Application
	WaitLock()
	TestValidatorPubkey() crypto.PubKey
}

func ExecTxInBlock(_app App, height int64, tx *gethtypes.Transaction) {
	_app.BeginBlock(abci.RequestBeginBlock{
		Header: tmproto.Header{
			Height:          height,
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
