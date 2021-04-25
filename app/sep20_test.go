package app_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/smartbch/smartbch/internal/testutils"
)

// testdata/erc20/contracts/MyToken.sol
var _myTokenCreationBytecode = testutils.HexToBytes(`608060405234801561001057600080fd5b50610493806100206000396000f3fe608060405234801561001057600080fd5b50600436106100625760003560e01c8063095ea7b31461006757806318160ddd1461009757806323b872dd146100b557806370a08231146100e5578063a9059cbb14610115578063dd62ed3e14610145575b600080fd5b610081600480360381019061007c9190610357565b610175565b60405161008e91906103b1565b60405180910390f35b61009f6101e2565b6040516100ac91906103cc565b60405180910390f35b6100cf60048036038101906100ca9190610308565b6101eb565b6040516100dc91906103b1565b60405180910390f35b6100ff60048036038101906100fa91906102a3565b6101f4565b60405161010c91906103cc565b60405180910390f35b61012f600480360381019061012a9190610357565b6101ff565b60405161013c91906103b1565b60405180910390f35b61015f600480360381019061015a91906102cc565b61026c565b60405161016c91906103cc565b60405180910390f35b60008273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040516101d491906103cc565b60405180910390a392915050565b6000606f905090565b60009392505050565b600060de9050919050565b60008273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef8460405161025e91906103cc565b60405180910390a392915050565b600061014d905092915050565b6000813590506102888161042f565b92915050565b60008135905061029d81610446565b92915050565b6000602082840312156102b557600080fd5b60006102c384828501610279565b91505092915050565b600080604083850312156102df57600080fd5b60006102ed85828601610279565b92505060206102fe85828601610279565b9150509250929050565b60008060006060848603121561031d57600080fd5b600061032b86828701610279565b935050602061033c86828701610279565b925050604061034d8682870161028e565b9150509250925092565b6000806040838503121561036a57600080fd5b600061037885828601610279565b92505060206103898582860161028e565b9150509250929050565b61039c816103f9565b82525050565b6103ab81610425565b82525050565b60006020820190506103c66000830184610393565b92915050565b60006020820190506103e160008301846103a2565b92915050565b60006103f282610405565b9050919050565b60008115159050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b610438816103e7565b811461044357600080fd5b50565b61044f81610425565b811461045a57600080fd5b5056fea2646970667358221220e22b0a7593e5da3dd5752b880380e08aa601231874f179cff886d04113bbbc7f64736f6c63430008000033`)

func TestERC20Events(t *testing.T) {
	key1, _ := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2)
	defer _app.Destroy()

	_, contractAddr := _app.DeployContractInBlock(1, key1, _myTokenCreationBytecode)
	require.NotEmpty(t, _app.GetCode(contractAddr))

	data := sep206ABI.MustPack("transfer", addr2, big.NewInt(100))
	tx2 := _app.MakeAndExecTxInBlock(3, key1, contractAddr, 0, data)

	blk3 := _app.GetBlock(3)
	require.Equal(t, int64(3), blk3.Number)
	require.Len(t, blk3.Transactions, 1)
	txInBlk3 := _app.GetTx(blk3.Transactions[0])
	require.Equal(t, gethtypes.ReceiptStatusSuccessful, txInBlk3.Status)
	require.Equal(t, "success", txInBlk3.StatusStr)
	require.Equal(t, tx2.Hash(), gethcmn.Hash(txInBlk3.Hash))
	require.Len(t, txInBlk3.Logs, 1)
	// TODO: check more fields
}

func TestGetSep20FromToAddressCount(t *testing.T) {
	key1, addr1 := testutils.GenKeyAndAddr()
	key2, addr2 := testutils.GenKeyAndAddr()
	key3, addr3 := testutils.GenKeyAndAddr()
	key4, addr4 := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key1, key2, key3, key4)
	defer _app.Destroy()

	_, contractAddr := _app.DeployContractInBlock(1, key1, _myTokenCreationBytecode)
	require.NotEmpty(t, _app.GetCode(contractAddr))

	// addr1 => addr2
	tx1 := _app.MakeAndExecTxInBlock(3, key1, contractAddr, 0,
		sep206ABI.MustPack("transfer", addr2, big.NewInt(100)))

	// addr1 => addr3
	tx2 := _app.MakeAndExecTxInBlock(5, key1, contractAddr, 0,
		sep206ABI.MustPack("transfer", addr3, big.NewInt(100)))

	// addr1 => addr4
	tx3 := _app.MakeAndExecTxInBlock(7, key1, contractAddr, 0,
		sep206ABI.MustPack("transfer", addr4, big.NewInt(100)))

	time.Sleep(200 * time.Millisecond)
	require.NotNil(t, "success", _app.GetTx(tx1.Hash()).StatusStr)
	require.NotNil(t, "success", _app.GetTx(tx2.Hash()).StatusStr)
	require.NotNil(t, "success", _app.GetTx(tx3.Hash()).StatusStr)

	// TODO: fix me
	require.Equal(t, int64(3), _app.GetSep20FromAddressCount(contractAddr, addr1))
	require.Equal(t, int64(1), _app.GetSep20ToAddressCount(contractAddr, addr2))
}
