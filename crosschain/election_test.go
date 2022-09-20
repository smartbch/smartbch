package crosschain_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/crosschain"
	cctypes "github.com/smartbch/smartbch/crosschain/types"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
	"github.com/smartbch/smartbch/param"
)

const (
	// CCOperatorsGovForTest.sol
	operatorsGovBytecode = `0x608060405234801561001057600080fd5b506113e1806100206000396000f3fe6080604052600436106100865760003560e01c8063692ea80211610059578063692ea80214610147578063a5a0e919146102ff578063c743dabb14610312578063e28d490614610329578063ea18875d1461038e57600080fd5b80632504a2161461008b5780633d5ec47e146100ad5780635b56ad67146101085780636199eef614610134575b600080fd5b34801561009757600080fd5b506100ab6100a636600461124e565b6103a3565b005b3480156100b957600080fd5b506100cd6100c8366004611270565b6107af565b604080516001600160a01b03958616815294909316602085015263ffffffff9091169183019190915260608201526080015b60405180910390f35b34801561011457600080fd5b5061012669021e19e0c9bab240000081565b6040519081526020016100ff565b6100ab610142366004611289565b6107fe565b34801561015357600080fd5b506100ab6101623660046112b9565b604080516101008101825233815260208101978852908101958652606081019485526080810193845260a0810192835260c08101918252600060e08201818152815460018101835591805291517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563600890920291820180546001600160a01b0319166001600160a01b0390921691909117905596517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56488015594517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56587015592517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56686015590517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e567850155517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e568840155517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e569830155517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56a90910155565b6100ab61030d3660046112fc565b610a56565b34801561031e57600080fd5b506101266283d60081565b34801561033557600080fd5b50610349610344366004611270565b61106a565b604080516001600160a01b0390991689526020890197909752958701949094526060860192909252608085015260a084015260c083015260e0820152610100016100ff565b34801561039a57600080fd5b506100ab6110ca565b60035482106103ee5760405162461bcd60e51b81526020600482015260126024820152716e6f2d737563682d7374616b652d696e666f60701b60448201526064015b60405180910390fd5b6000600383815481106104035761040361133d565b6000918252602090912060039091020180549091506001600160a01b031633146104605760405162461bcd60e51b815260206004820152600e60248201526d6e6f742d796f75722d7374616b6560901b60448201526064016103e5565b81816002015410156104a85760405162461bcd60e51b81526020600482015260116024820152700eed2e8d0c8e4c2ee5ae8dede5adaeac6d607b1b60448201526064016103e5565b600181015442906104ca906283d60090600160a01b900463ffffffff16611369565b106105045760405162461bcd60e51b815260206004820152600a6024820152696e6f742d6d617475726560b01b60448201526064016103e5565b6001818101546001600160a01b031660009081526020919091526040812054815490919081908390811061053a5761053a61133d565b6000918252602090912060089091020180549091506001600160a01b03166105975760405162461bcd60e51b815260206004820152601060248201526f373796b9bab1b416b7b832b930ba37b960811b60448201526064016103e5565b838360020160008282546105ab9190611382565b9091555050600283015460000361060357600385815481106105cf576105cf61133d565b60009182526020822060039091020180546001600160a01b03191681556001810180546001600160c01b0319169055600201555b838160050160008282546106179190611382565b90915550508054336001600160a01b03909116036106a557838160060160008282546106439190611382565b90915550506007810154156106a55769021e19e0c9bab24000008160060154116106a55760405162461bcd60e51b8152602060048201526013602482015272746f6f2d6c6573732d73656c662d7374616b6560681b60448201526064016103e5565b806005015460000361075657600082815481106106c4576106c461133d565b60009182526020808320600890920290910180546001600160a01b0319168155600180820184905560028083018590556003830185905560048301859055600583018590556006830185905560079092018490553384529182905260408320839055805491820181559091527f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace018290555b6107603385611130565b8054604080518781526020810187905233926001600160a01b0316917f4e4da858820d03358af7e05090375c4a8cfaddda3c24e48bd64e376f13d2c6bd91015b60405180910390a35050505050565b600381815481106107bf57600080fd5b60009182526020909120600390910201805460018201546002909201546001600160a01b03918216935090821691600160a01b900463ffffffff169084565b600034116108405760405162461bcd60e51b815260206004820152600f60248201526e6465706f7369742d6e6f7468696e6760881b60448201526064016103e5565b604080516080810182523381526001600160a01b03838116602080840182815263ffffffff4281168688019081523460608801908152600380546001818101835560008381529a51919092027fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b81018054928b166001600160a01b03199093169290921790915594517fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85c860180549451909516600160a01b026001600160c01b031990941698169790971791909117909155517fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85d90910155908352529081205481549091908190839081106109575761095761133d565b6000918252602090912060089091020180549091506001600160a01b038481169116146109b95760405162461bcd60e51b815260206004820152601060248201526f373796b9bab1b416b7b832b930ba37b960811b60448201526064016103e5565b348160050160008282546109cd9190611369565b9091555050336001600160a01b038416036109fc57348160060160008282546109f69190611369565b90915550505b805460035433916001600160a01b0316907febc590f21987ecf5c1f8466f90a1a5d112eec48f1c2e652a65c42034cc2c107a90610a3b90600190611382565b604080519182523460208301520160405180910390a3505050565b8360ff1660021480610a6b57508360ff166003145b610aaf5760405162461bcd60e51b81526020600482015260156024820152740d2dcecc2d8d2c85ae0eac4d6caf25ae0e4caccd2f605b1b60448201526064016103e5565b69021e19e0c9bab2400000341015610afc5760405162461bcd60e51b815260206004820152601060248201526f6465706f7369742d746f6f2d6c65737360801b60448201526064016103e5565b336000908152600160205260409020548015610b4d5760405162461bcd60e51b815260206004820152601060248201526f1bdc195c985d1bdc8b595e1a5cdd195960821b60448201526064016103e5565b60005415610bcc57336001600160a01b031660008081548110610b7257610b7261133d565b60009182526020909120600890910201546001600160a01b031603610bcc5760405162461bcd60e51b815260206004820152601060248201526f1bdc195c985d1bdc8b595e1a5cdd195960821b60448201526064016103e5565b60025415610d10576002805460009190610be890600190611382565b81548110610bf857610bf861133d565b906000526020600020015490506002805480610c1657610c16611395565b60019003818190600052602060002001600090559055604051806101000160405280336001600160a01b031681526020018760ff168152602001868152602001858152602001848152602001348152602001348152602001600081525060008281548110610c8657610c8661133d565b6000918252602080832084516008939093020180546001600160a01b0319166001600160a01b03909316929092178255838101516001808401919091556040808601516002850155606086015160038501556080860151600485015560a0860151600585015560c0860151600685015560e090950151600790930192909255338352522055610ed3565b604080516101008101825233815260ff87166020820190815291810186815260608201868152608083018681523460a0850181815260c08601918252600060e0870181815281546001808201845583805298517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563600890920291820180546001600160a01b0319166001600160a01b0390921691909117905598517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e5648a015595517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56589015593517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56688015591517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56787015590517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e568860155517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56985015590517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56a909301929092559054610ec29190611382565b336000908152600160205260409020555b60408051608081018252338082526020820181815263ffffffff4281168486019081523460608601818152600380546001810182556000829052975197027fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b810180546001600160a01b03998a166001600160a01b031990911617905594517fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85c860180549451909516600160a01b026001600160c01b03199094169716969096179190911790915592517fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85d9091015591517ff60326b9d09410d6ce5b777e4d80932914842920440df63a64bd2413b2f5ff1c9161101a91899189918991899160ff959095168552602085019390935260408401919091526060830152608082015260a00190565b60405180910390a2600354339081907febc590f21987ecf5c1f8466f90a1a5d112eec48f1c2e652a65c42034cc2c107a9061105790600190611382565b60408051918252346020830152016107a0565b6000818154811061107a57600080fd5b6000918252602090912060089091020180546001820154600283015460038401546004850154600586015460068701546007909701546001600160a01b0390961697509395929491939092909188565b60008054806110db576110db611395565b60008281526020812060086000199093019283020180546001600160a01b0319168155600181018290556002810182905560038101829055600481018290556005810182905560068101829055600701559055565b804710156111805760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a20696e73756666696369656e742062616c616e636500000060448201526064016103e5565b6000826001600160a01b03168260405160006040518083038185875af1925050503d80600081146111cd576040519150601f19603f3d011682016040523d82523d6000602084013e6111d2565b606091505b50509050806112495760405162461bcd60e51b815260206004820152603a60248201527f416464726573733a20756e61626c6520746f2073656e642076616c75652c207260448201527f6563697069656e74206d6179206861766520726576657274656400000000000060648201526084016103e5565b505050565b6000806040838503121561126157600080fd5b50508035926020909101359150565b60006020828403121561128257600080fd5b5035919050565b60006020828403121561129b57600080fd5b81356001600160a01b03811681146112b257600080fd5b9392505050565b60008060008060008060c087890312156112d257600080fd5b505084359660208601359650604086013595606081013595506080810135945060a0013592509050565b6000806000806080858703121561131257600080fd5b843560ff8116811461132357600080fd5b966020860135965060408601359560600135945092505050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b8082018082111561137c5761137c611353565b92915050565b8181038181111561137c5761137c611353565b634e487b7160e01b600052603160045260246000fdfea2646970667358221220a3c835f91cc8da6ad6e8a2dc893652550bd95dd5d756fefada33671bcdac53ae64736f6c63430008100033`

	// CCMonitorsGovForTest.sol
	monitorsGovBytecode = `0x608060405234801561001057600080fd5b50610e4a806100206000396000f3fe60806040526004361061007b5760003560e01c80635a627dbc1161004e5780635a627dbc146101525780636d9898581461015a578063939624ab1461016d578063f8aabdd41461018d57600080fd5b80630eb1b5861461008057806344a58781146100a957806347998be314610100578063562aa20114610122575b600080fd5b34801561008c57600080fd5b5061009660005481565b6040519081526020015b60405180910390f35b3480156100b557600080fd5b506100c96100c4366004610ccb565b6102eb565b604080516001600160a01b0390971687526020870195909552938501929092526060840152608083015260a082015260c0016100a0565b34801561010c57600080fd5b5061012061011b366004610ccb565b600055565b005b34801561012e57600080fd5b5061014261013d366004610ce4565b61033b565b60405190151581526020016100a0565b6101206103ba565b610120610168366004610d14565b6104d9565b34801561017957600080fd5b50610120610188366004610ccb565b61092f565b34801561019957600080fd5b506101206101a8366004610d4f565b6040805160c081018252338152602081019586529081019384526060810192835260808101918252600060a08201818152600180548082018255925291517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6600690920291820180546001600160a01b0319166001600160a01b0390921691909117905594517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf786015592517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf885015590517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf9840155517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cfa830155517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cfb90910155565b600181815481106102fb57600080fd5b60009182526020909120600690910201805460018201546002830154600384015460048501546005909501546001600160a01b0390941695509193909286565b600154600090810361034f57506000919050565b6001600160a01b038216600090815260026020526040812054600180549192918390811061037f5761037f610d81565b6000918252602090912060069091020180549091506001600160a01b0385811691161480156103b2575060008160050154115b949350505050565b6001546103e25760405162461bcd60e51b81526004016103d990610d97565b60405180910390fd5b33600090815260026020526040812054600180549192918390811061040957610409610d81565b6000918252602090912060069091020180549091506001600160a01b031633146104455760405162461bcd60e51b81526004016103d990610d97565b600034116104875760405162461bcd60e51b815260206004820152600f60248201526e6465706f7369742d6e6f7468696e6760881b60448201526064016103d9565b3481600401600082825461049b9190610dd2565b909155505060405134815233907fb218e74e1d7548db0bff4bc81f75f5b5d41d66cf9151f0311fbcb7344bd9d0339060200160405180910390a25050565b8260ff16600214806104ee57508260ff166003145b6105325760405162461bcd60e51b81526020600482015260156024820152740d2dcecc2d8d2c85ae0eac4d6caf25ae0e4caccd2f605b1b60448201526064016103d9565b69152d02c7e14af680000034101561057f5760405162461bcd60e51b815260206004820152601060248201526f6465706f7369742d746f6f2d6c65737360801b60448201526064016103d9565b3360009081526002602052604090205480156105cf5760405162461bcd60e51b815260206004820152600f60248201526e1b5bdb9a5d1bdc8b595e1a5cdd1959608a1b60448201526064016103d9565b6001541561064e57336001600160a01b031660016000815481106105f5576105f5610d81565b60009182526020909120600690910201546001600160a01b03160361064e5760405162461bcd60e51b815260206004820152600f60248201526e1b5bdb9a5d1bdc8b595e1a5cdd1959608a1b60448201526064016103d9565b6003541561077457600380546000919061066a90600190610deb565b8154811061067a5761067a610d81565b90600052602060002001549050600380548061069857610698610dfe565b600190038181906000526020600020016000905590556040518060c00160405280336001600160a01b031681526020018660ff1681526020018581526020018481526020013481526020016000815250600182815481106106fb576106fb610d81565b6000918252602080832084516006939093020180546001600160a01b0319166001600160a01b03909316929092178255838101516001830155604080850151600280850191909155606086015160038501556080860151600485015560a0909501516005909301929092553383529290925220556108dd565b6040805160c08101825233815260ff861660208201908152918101858152606082018581523460808401908152600060a085018181526001805480820182559281905295517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6600690930292830180546001600160a01b0319166001600160a01b0390921691909117905595517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf782015592517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf884015590517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf9830155517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cfa82015591517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cfb9092019190915580546108cc9190610deb565b336000908152600260205260409020555b6040805160ff861681526020810185905290810183905234606082015233907f3ca1553e5e289d81faa2afbcf3f32d4df4c14526b7e2a81cd1bb25d1b1415b5d9060800160405180910390a250505050565b60015461094e5760405162461bcd60e51b81526004016103d990610d97565b33600090815260026020526040812054600180549192918390811061097557610975610d81565b6000918252602090912060069091020180549091506001600160a01b031633146109b15760405162461bcd60e51b81526004016103d990610d97565b82816004015410156109f95760405162461bcd60e51b81526020600482015260116024820152700eed2e8d0c8e4c2ee5ae8dede5adaeac6d607b1b60448201526064016103d9565b82816004016000828254610a0d9190610deb565b9091555050600481015469152d02c7e14af68000001115610ac657600581015415610a6e5760405162461bcd60e51b81526020600482015260116024820152706d6f6e69746f722d69732d61637469766560781b60448201526064016103d9565b620d2f0060005442610a809190610deb565b10610ac65760405162461bcd60e51b81526020600482015260166024820152756f7574736964652d756e7374616b652d77696e646f7760501b60448201526064016103d9565b8060040154600003610b695760018281548110610ae557610ae5610d81565b60009182526020808320600690920290910180546001600160a01b031916815560018082018490556002808301859055600380840186905560048401869055600590930185905533855290925260408320839055805491820181559091527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b018290555b610b733384610bad565b60405183815233907f551255bcb3977c75cd031d9bc6d8233f1491dd8bcfe857b996b0afb99b5c3d399060200160405180910390a2505050565b80471015610bfd5760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a20696e73756666696369656e742062616c616e636500000060448201526064016103d9565b6000826001600160a01b03168260405160006040518083038185875af1925050503d8060008114610c4a576040519150601f19603f3d011682016040523d82523d6000602084013e610c4f565b606091505b5050905080610cc65760405162461bcd60e51b815260206004820152603a60248201527f416464726573733a20756e61626c6520746f2073656e642076616c75652c207260448201527f6563697069656e74206d6179206861766520726576657274656400000000000060648201526084016103d9565b505050565b600060208284031215610cdd57600080fd5b5035919050565b600060208284031215610cf657600080fd5b81356001600160a01b0381168114610d0d57600080fd5b9392505050565b600080600060608486031215610d2957600080fd5b833560ff81168114610d3a57600080fd5b95602085013595506040909401359392505050565b60008060008060808587031215610d6557600080fd5b5050823594602084013594506040840135936060013592509050565b634e487b7160e01b600052603260045260246000fd5b6020808252600b908201526a3737ba16b6b7b734ba37b960a91b604082015260600190565b634e487b7160e01b600052601160045260246000fd5b80820180821115610de557610de5610dbc565b92915050565b81810381811115610de557610de5610dbc565b634e487b7160e01b600052603160045260246000fdfea26469706673582212204ae39d91c29562422c093b0a60a7f4ad9087773e140476d413ba9558760c225864736f6c63430008100033`
)

var (
	operatorsGovABI = ethutils.MustParseABI(`
[
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "pubkeyPrefix",
          "type": "uint256"
        },
        {
          "internalType": "bytes32",
          "name": "pubkeyX",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "rpcUrl",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "intro",
          "type": "bytes32"
        },
        {
          "internalType": "uint256",
          "name": "totalStakedAmt",
          "type": "uint256"
        },
        {
          "internalType": "uint256",
          "name": "selfStakedAmt",
          "type": "uint256"
        }
      ],
      "name": "addOperator",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "removeLastOperator",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    }
]
`)

	monitorsGovABI = ethutils.MustParseABI(`
[
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "ts",
          "type": "uint256"
        }
      ],
      "name": "setLastElectionTime",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "lastElectionTime",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "pubkeyPrefix",
          "type": "uint256"
        },
        {
          "internalType": "bytes32",
          "name": "pubkeyX",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "intro",
          "type": "bytes32"
        },
        {
          "internalType": "uint256",
          "name": "stakedAmt",
          "type": "uint256"
        }
      ],
      "name": "addMonitor",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    }
]
`)
)

var (
	noOpLogger           = log.NewNopLogger()
	operatorMinStakedAmt = big.NewInt(0).Mul(big.NewInt(param.OperatorMinStakedBCH), big.NewInt(1e18))
	monitorMinStakedAmt  = big.NewInt(0).Mul(big.NewInt(param.MonitorMinStakedBCH), big.NewInt(1e18))
)

func TestOperatorsGovStorageRW(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	tx, _, contractAddr := _app.DeployContractInBlockWithGas(key,
		testutils.HexToBytes(operatorsGovBytecode), testutils.DefaultGasLimit*2, testutils.DefaultGasPrice)
	_app.EnsureTxSuccess(tx.Hash())

	addOperator1 := packAddOperatorData(02, "pubkeyX_o1", "12.34.56.78:9011", "operator#1", big.NewInt(1012), big.NewInt(1011))
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOperator1)
	_app.EnsureTxSuccess(tx.Hash())

	addOperator2 := packAddOperatorData(03, "pubkeyX_o2", "12.34.56.78:9012", "operator#2", big.NewInt(2012), big.NewInt(2011))
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOperator2)
	_app.EnsureTxSuccess(tx.Hash())

	addOperator3 := packAddOperatorData(02, "pubkeyX_o3", "12.34.56.78:9013", "operator#3", big.NewInt(3012), big.NewInt(3011))
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOperator3)
	_app.EnsureTxSuccess(tx.Hash())

	// read data from Go
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)

	accInfo := ctx.GetAccount(contractAddr)
	seq := accInfo.Sequence()

	operators := crosschain.ReadOperatorInfos(ctx, seq)
	require.Len(t, operators, 3)
	require.Equal(t, addr, operators[0].Addr)
	require.Equal(t, "027075626b6579585f6f3100000000000000000000000000000000000000000000",
		hex.EncodeToString(operators[0].Pubkey))
	require.Equal(t, "12.34.56.78:9011", bytes32ToStr(operators[0].RpcUrl))
	require.Equal(t, "operator#1", bytes32ToStr(operators[0].Intro))
	require.Equal(t, uint64(1012), operators[0].TotalStakedAmt.Uint64())
	require.Equal(t, uint64(1011), operators[0].SelfStakedAmt.Uint64())
	require.Equal(t, uint64(0), operators[0].ElectedTime.Uint64())

	require.Equal(t, addr, operators[1].Addr)
	require.Equal(t, "037075626b6579585f6f3200000000000000000000000000000000000000000000",
		hex.EncodeToString(operators[1].Pubkey))
	require.Equal(t, "12.34.56.78:9012", bytes32ToStr(operators[1].RpcUrl))
	require.Equal(t, "operator#2", bytes32ToStr(operators[1].Intro))
	require.Equal(t, uint64(2012), operators[1].TotalStakedAmt.Uint64())
	require.Equal(t, uint64(2011), operators[1].SelfStakedAmt.Uint64())
	require.Equal(t, uint64(0), operators[1].ElectedTime.Uint64())

	require.Equal(t, addr, operators[2].Addr)
	require.Equal(t, "027075626b6579585f6f3300000000000000000000000000000000000000000000",
		hex.EncodeToString(operators[2].Pubkey))
	require.Equal(t, "12.34.56.78:9013", bytes32ToStr(operators[2].RpcUrl))
	require.Equal(t, "operator#3", bytes32ToStr(operators[2].Intro))
	require.Equal(t, uint64(3012), operators[2].TotalStakedAmt.Uint64())
	require.Equal(t, uint64(3011), operators[2].SelfStakedAmt.Uint64())
	require.Equal(t, uint64(0), operators[2].ElectedTime.Uint64())

	//operators[1].votes = 123
	//ctx2 := _app.GetRunTxContext()
	crosschain.WriteOperatorElectedTime(ctx, seq, 1, 123)
	//ctx2.Close(true)
	operators = crosschain.ReadOperatorInfos(ctx, seq)
	require.Equal(t, uint64(0), operators[0].ElectedTime.Uint64())
	require.Equal(t, uint64(123), operators[1].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operators[2].ElectedTime.Uint64())
}

func TestMonitorsGovStorageRW(t *testing.T) {
	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	tx, _, contractAddr := _app.DeployContractInBlockWithGas(key,
		testutils.HexToBytes(monitorsGovBytecode), testutils.DefaultGasLimit*2, testutils.DefaultGasPrice)
	_app.EnsureTxSuccess(tx.Hash())

	addMonitor1 := packAddMonitorData(02, "pubkeyX_m1", "monitor#1", big.NewInt(8001))
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addMonitor1)
	_app.EnsureTxSuccess(tx.Hash())

	addMonitor2 := packAddMonitorData(03, "pubkeyX_m2", "monitor#2", big.NewInt(8002))
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addMonitor2)
	_app.EnsureTxSuccess(tx.Hash())

	addMonitor3 := packAddMonitorData(02, "pubkeyX_m3", "monitor#3", big.NewInt(8003))
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addMonitor3)
	_app.EnsureTxSuccess(tx.Hash())

	setLastElectionTime := monitorsGovABI.MustPack("setLastElectionTime", big.NewInt(223344))
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, setLastElectionTime)
	_app.EnsureTxSuccess(tx.Hash())

	// read data from Go
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)

	accInfo := ctx.GetAccount(contractAddr)
	seq := accInfo.Sequence()

	monitors := crosschain.ReadMonitorInfos(ctx, seq)
	require.Len(t, monitors, 3)
	require.Equal(t, addr, monitors[0].Addr)
	require.Equal(t, "027075626b6579585f6d3100000000000000000000000000000000000000000000",
		hex.EncodeToString(monitors[0].Pubkey))
	require.Equal(t, "monitor#1", bytes32ToStr(monitors[0].Intro))
	require.Equal(t, uint64(8001), monitors[0].StakedAmt.Uint64())
	require.Equal(t, uint64(0), monitors[0].ElectedTime.Uint64())

	require.Equal(t, addr, monitors[1].Addr)
	require.Equal(t, "037075626b6579585f6d3200000000000000000000000000000000000000000000",
		hex.EncodeToString(monitors[1].Pubkey))
	require.Equal(t, "monitor#2", bytes32ToStr(monitors[1].Intro))
	require.Equal(t, uint64(8002), monitors[1].StakedAmt.Uint64())
	require.Equal(t, uint64(0), monitors[1].ElectedTime.Uint64())

	require.Equal(t, addr, monitors[2].Addr)
	require.Equal(t, "027075626b6579585f6d3300000000000000000000000000000000000000000000",
		hex.EncodeToString(monitors[2].Pubkey))
	require.Equal(t, "monitor#3", bytes32ToStr(monitors[2].Intro))
	require.Equal(t, uint64(8003), monitors[2].StakedAmt.Uint64())
	require.Equal(t, uint64(0), monitors[2].ElectedTime.Uint64())

	//operators[1].votes = 123
	//ctx2 := _app.GetRunTxContext()
	crosschain.WriteMonitorElectedTime(ctx, seq, 1, 12345)
	//ctx2.Close(true)
	monitors = crosschain.ReadMonitorInfos(ctx, seq)
	require.Equal(t, uint64(0), monitors[0].ElectedTime.Uint64())
	require.Equal(t, uint64(12345), monitors[1].ElectedTime.Uint64())
	require.Equal(t, uint64(0), monitors[2].ElectedTime.Uint64())

	lastElectionTime := crosschain.ReadMonitorsLastElectionTime(ctx, seq)
	require.Equal(t, uint64(223344), lastElectionTime.Uint64())

	crosschain.WriteMonitorsLastElectionTime(ctx, seq, 556677)
	lastElectionTime = crosschain.ReadMonitorsLastElectionTime(ctx, seq)
	require.Equal(t, uint64(556677), lastElectionTime.Uint64())
}

func TestOperatorsElection(t *testing.T) {
	key, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	tx, _, contractAddr := _app.DeployContractInBlockWithGas(key,
		testutils.HexToBytes(operatorsGovBytecode), testutils.DefaultGasLimit*2, testutils.DefaultGasPrice)
	_app.EnsureTxSuccess(tx.Hash())

	ctx := _app.GetRpcContext()
	accInfo := ctx.GetAccount(contractAddr)
	seq := accInfo.Sequence()
	ctx.Close(false)

	// add 9 valid operator candidates
	for i := 0; i < 9; i++ {
		stakedAmt := big.NewInt(0).Add(operatorMinStakedAmt, big.NewInt(int64(i)))
		data := packAddOperatorData(02,
			fmt.Sprintf("pk#%d", i),
			fmt.Sprintf("rpc#%d", i),
			fmt.Sprintf("op#%d", i), stakedAmt, stakedAmt)
		tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, data)
		_app.EnsureTxSuccess(tx.Hash())
	}

	// not enough operator candidates
	ctx = _app.GetRpcContext()
	require.Equal(t, crosschain.OperatorElectionNotEnoughCandidates,
		crosschain.ElectOperators_(ctx, seq, 12345, noOpLogger))
	ctx.Close(false)

	// add 1 invalid operator candidate
	data := packAddOperatorData(03, "pk123", "rpc123", "op123", big.NewInt(123), big.NewInt(123))
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, data)
	_app.EnsureTxSuccess(tx.Hash())
	ctx = _app.GetRpcContext()
	require.Equal(t, crosschain.OperatorElectionNotEnoughCandidates,
		crosschain.ElectOperators_(ctx, seq, 12345, noOpLogger))
	ctx.Close(false)

	// add 3 valid operator candidates
	for i := 9; i < 12; i++ {
		stakedAmt := big.NewInt(0).Add(operatorMinStakedAmt, big.NewInt(int64(i)))
		data := packAddOperatorData(02,
			fmt.Sprintf("pk#%d", i),
			fmt.Sprintf("rpc#%d", i),
			fmt.Sprintf("op#%d", i), stakedAmt, stakedAmt)
		tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, data)
		_app.EnsureTxSuccess(tx.Hash())
	}

	// first election
	ctx = _app.GetRunTxContext()
	require.Equal(t, crosschain.OperatorElectionOK,
		crosschain.ElectOperators_(ctx, seq, 0x12345, noOpLogger))
	operatorInfos := crosschain.ReadOperatorInfos(ctx, seq)
	ctx.Close(true)
	require.Len(t, operatorInfos, 13)
	require.Equal(t, uint64(0), operatorInfos[0].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[1].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[2].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[3].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[4].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[5].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[6].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[7].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[8].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[9].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[10].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[11].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[12].ElectedTime.Uint64())

	ctx = _app.GetRunTxContext()
	require.Equal(t, crosschain.OperatorElectionNotChanged,
		crosschain.ElectOperators_(ctx, seq, 0x12345, noOpLogger))
	ctx.Close(false)

	// add 4 valid operator candidates
	for i := 12; i < 16; i++ {
		stakedAmt := big.NewInt(0).Add(operatorMinStakedAmt, big.NewInt(int64(i)))
		data := packAddOperatorData(03,
			fmt.Sprintf("pk#%d", i),
			fmt.Sprintf("rpc#%d", i),
			fmt.Sprintf("op#%d", i), stakedAmt, stakedAmt)
		tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, data)
		_app.EnsureTxSuccess(tx.Hash())
	}

	ctx = _app.GetRunTxContext()
	require.Equal(t, crosschain.OperatorElectionChangedTooMany,
		crosschain.ElectOperators_(ctx, seq, 0x12345, noOpLogger))
	ctx.Close(false)

	// remove last candidate
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0,
		operatorsGovABI.MustPack("removeLastOperator"))
	_app.EnsureTxSuccess(tx.Hash())

	ctx = _app.GetRunTxContext()
	require.Equal(t, crosschain.OperatorElectionOK,
		crosschain.ElectOperators_(ctx, seq, 0x123456, noOpLogger))
	operatorInfos = crosschain.ReadOperatorInfos(ctx, seq)
	ctx.Close(true)
	require.Len(t, operatorInfos, 16)
	require.Equal(t, uint64(0), operatorInfos[0].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[1].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[2].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[3].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[4].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[5].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[6].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[7].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[8].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[9].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[10].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[11].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[12].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[13].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[14].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[15].ElectedTime.Uint64())
}

func TestMonitorsElection(t *testing.T) {
	key, _ := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	tx, _, contractAddr := _app.DeployContractInBlockWithGas(key,
		testutils.HexToBytes(monitorsGovBytecode), testutils.DefaultGasLimit*2, testutils.DefaultGasPrice)
	_app.EnsureTxSuccess(tx.Hash())

	ctx := _app.GetRpcContext()
	accInfo := ctx.GetAccount(contractAddr)
	seq := accInfo.Sequence()
	ctx.Close(false)

	// add 5 valid monitor candidates
	for i := 0; i < 5; i++ {
		stakedAmt := big.NewInt(0).Add(monitorMinStakedAmt, big.NewInt(int64(i)))
		data := packAddMonitorData(02, fmt.Sprintf("pk#%d", i), fmt.Sprintf("op#%d", i), stakedAmt)
		tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, data)
		_app.EnsureTxSuccess(tx.Hash())
	}
	// add 2 invalid monitor candidates
	for i := 5; i < 7; i++ {
		stakedAmt := big.NewInt(123)
		data := packAddMonitorData(02, fmt.Sprintf("pk#%d", i), fmt.Sprintf("op#%d", i), stakedAmt)
		tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, data)
		_app.EnsureTxSuccess(tx.Hash())
	}

	// invalid nomination count
	ctx = _app.GetRunTxContext()
	require.Equal(t, crosschain.MonitorElectionInvalidNominationCount,
		crosschain.ElectMonitors_(ctx, seq, make([]*cctypes.Nomination, 2), 123, noOpLogger))
	require.Equal(t, crosschain.MonitorElectionInvalidNominationCount,
		crosschain.ElectMonitors_(ctx, seq, make([]*cctypes.Nomination, 4), 123, noOpLogger))
	ctx.Close(false)

	// invalid nominations
	ctx = _app.GetRunTxContext()
	nominations := []*cctypes.Nomination{
		{Pubkey: toBytes33(02, "pk#1"), NominatedCount: 1},
		{Pubkey: toBytes33(02, "pk#2"), NominatedCount: 1},
		{Pubkey: toBytes33(02, "pk#5"), NominatedCount: 1}, // invalid
	}
	require.Equal(t, crosschain.MonitorElectionInvalidNominations,
		crosschain.ElectMonitors_(ctx, seq, nominations, 123, noOpLogger))
	nominations = []*cctypes.Nomination{
		{Pubkey: toBytes33(02, "pk#2"), NominatedCount: 1},
		{Pubkey: toBytes33(02, "pk#3"), NominatedCount: 0}, // invalid
		{Pubkey: toBytes33(02, "pk#4"), NominatedCount: 1},
	}
	require.Equal(t, crosschain.MonitorElectionInvalidNominations,
		crosschain.ElectMonitors_(ctx, seq, nominations, 123, noOpLogger))
	ctx.Close(false)

	// first election
	ctx = _app.GetRunTxContext()
	nominations = []*cctypes.Nomination{
		{Pubkey: toBytes33(02, "pk#2"), NominatedCount: 100},
		{Pubkey: toBytes33(02, "pk#3"), NominatedCount: 200},
		{Pubkey: toBytes33(02, "pk#4"), NominatedCount: 300},
	}
	require.Equal(t, crosschain.MonitorElectionOK,
		crosschain.ElectMonitors_(ctx, seq, nominations, 0x12345, noOpLogger))
	monitorInfos := crosschain.ReadMonitorInfos(ctx, seq)
	lastElectionTime := crosschain.ReadMonitorsLastElectionTime(ctx, seq)
	ctx.Close(true)
	require.Len(t, monitorInfos, 7)
	require.Equal(t, uint64(0), monitorInfos[0].ElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[1].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), monitorInfos[2].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), monitorInfos[3].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), monitorInfos[4].ElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[5].ElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[6].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), lastElectionTime.Uint64())

	// changed too many
	ctx = _app.GetRunTxContext()
	nominations = []*cctypes.Nomination{
		{Pubkey: toBytes33(02, "pk#0"), NominatedCount: 100},
		{Pubkey: toBytes33(02, "pk#1"), NominatedCount: 200},
		{Pubkey: toBytes33(02, "pk#3"), NominatedCount: 300},
	}
	require.Equal(t, crosschain.MonitorElectionChangedTooMany,
		crosschain.ElectMonitors_(ctx, seq, nominations, 0x12345, noOpLogger))
	ctx.Close(false)

	// election ok
	ctx = _app.GetRunTxContext()
	nominations = []*cctypes.Nomination{
		{Pubkey: toBytes33(02, "pk#1"), NominatedCount: 100},
		{Pubkey: toBytes33(02, "pk#2"), NominatedCount: 200},
		{Pubkey: toBytes33(02, "pk#3"), NominatedCount: 300},
	}
	require.Equal(t, crosschain.MonitorElectionOK,
		crosschain.ElectMonitors_(ctx, seq, nominations, 0x123456, noOpLogger))
	monitorInfos = crosschain.ReadMonitorInfos(ctx, seq)
	lastElectionTime = crosschain.ReadMonitorsLastElectionTime(ctx, seq)
	ctx.Close(true)
	require.Len(t, monitorInfos, 7)
	require.Equal(t, uint64(0), monitorInfos[0].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), monitorInfos[1].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), monitorInfos[2].ElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), monitorInfos[3].ElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[4].ElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[5].ElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[6].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), lastElectionTime.Uint64())
}

func packAddOperatorData(pubkeyPrefix int64, pubkeyX, rpcURL, intro string,
	totalStakedAmt, selfStakedAmt *big.Int) []byte {

	return operatorsGovABI.MustPack("addOperator",
		big.NewInt(pubkeyPrefix),
		toBytes32(pubkeyX),
		toBytes32(rpcURL),
		toBytes32(intro),
		totalStakedAmt,
		selfStakedAmt,
	)
}
func packAddMonitorData(pubkeyPrefix int64, pubkeyX, intro string, stakedAmt *big.Int) []byte {
	return monitorsGovABI.MustPack("addMonitor",
		big.NewInt(pubkeyPrefix),
		toBytes32(pubkeyX),
		toBytes32(intro),
		stakedAmt,
	)
}

func toBytes32(s string) [32]byte {
	out := [32]byte{}
	copy(out[:], s)
	return out
}
func toBytes33(pubkeyPrefix uint8, pubkeyX string) [33]byte {
	out := [33]byte{pubkeyPrefix}
	copy(out[1:], pubkeyX)
	return out
}

func bytes32ToStr(bs []byte) string {
	return strings.TrimRight(string(bs), string([]byte{0}))
}
