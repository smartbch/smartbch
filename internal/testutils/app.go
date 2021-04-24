package testutils

import (
	"bytes"
	"math/big"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

func (_app *TestApp) DeployContractInBlock(height int64, privKey string, nonce uint64, data []byte) *gethtypes.Transaction {
	txData := &gethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(0),
		Gas:      1000000,
		To:       nil,
		Value:    big.NewInt(0),
		Data:     data,
	}
	tx := gethtypes.NewTx(txData)
	tx = MustSignTx(tx, _app.ChainID().ToBig(), privKey)
	_app.ExecTxInBlock(height, tx)
	return tx
}

func (_app *TestApp) MakeAndExecTxInBlock(height int64, privKey string, nonce uint64,
	toAddr gethcmn.Address, val int64, data []byte) *gethtypes.Transaction {

	return _app.MakeAndExecTxInBlockWithGasPrice(height, privKey, nonce, toAddr, val, data, 0)
}
func (_app *TestApp) MakeAndExecTxInBlockWithGasPrice(height int64, privKey string, nonce uint64,
	toAddr gethcmn.Address, val int64, data []byte, gasPrice int64) *gethtypes.Transaction {

	txData := &gethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(gasPrice),
		Gas:      1000000,
		To:       &toAddr,
		Value:    big.NewInt(val),
		Data:     data,
	}
	tx := gethtypes.NewTx(txData)
	tx = MustSignTx(tx, _app.ChainID().ToBig(), privKey)
	_app.ExecTxInBlock(height, tx)
	return tx
}

func (_app *TestApp) ExecTxInBlock(height int64, tx *gethtypes.Transaction) {
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
