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
	// CCOperatorsGovForStorageTest.sol
	operatorsGovBytecode = `0x608060405234801561001057600080fd5b5061146b806100206000396000f3fe6080604052600436106100865760003560e01c8063692ea80211610059578063692ea80214610147578063a5a0e9191461032e578063c743dabb14610341578063e28d490614610358578063ea18875d146103c657600080fd5b80632504a2161461008b5780633d5ec47e146100ad5780635b56ad67146101085780636199eef614610134575b600080fd5b34801561009757600080fd5b506100ab6100a63660046112d8565b6103db565b005b3480156100b957600080fd5b506100cd6100c83660046112fa565b6107ee565b604080516001600160a01b03958616815294909316602085015263ffffffff9091169183019190915260608201526080015b60405180910390f35b34801561011457600080fd5b5061012669021e19e0c9bab240000081565b6040519081526020016100ff565b6100ab610142366004611313565b61083d565b34801561015357600080fd5b506100ab610162366004611343565b604080516101208101825233815260208101978852908101958652606081019485526080810193845260a0810192835260c08101918252600060e082018181526101008301828152825460018101845592805292517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563600990930292830180546001600160a01b0319166001600160a01b0390921691909117905597517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56482015595517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56587015593517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56686015591517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e567850155517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e568840155517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56983015591517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56a82015590517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56b90910155565b6100ab61033c366004611386565b610a95565b34801561034d57600080fd5b506101266283d60081565b34801561036457600080fd5b506103786103733660046112fa565b6110e8565b604080516001600160a01b03909a168a5260208a0198909852968801959095526060870193909352608086019190915260a085015260c084015260e0830152610100820152610120016100ff565b3480156103d257600080fd5b506100ab61114d565b60035482106104265760405162461bcd60e51b81526020600482015260126024820152716e6f2d737563682d7374616b652d696e666f60701b60448201526064015b60405180910390fd5b60006003838154811061043b5761043b6113c7565b6000918252602090912060039091020180549091506001600160a01b031633146104985760405162461bcd60e51b815260206004820152600e60248201526d6e6f742d796f75722d7374616b6560901b604482015260640161041d565b81816002015410156104e05760405162461bcd60e51b81526020600482015260116024820152700eed2e8d0c8e4c2ee5ae8dede5adaeac6d607b1b604482015260640161041d565b60018101544290610502906283d60090600160a01b900463ffffffff166113f3565b1061053c5760405162461bcd60e51b815260206004820152600a6024820152696e6f742d6d617475726560b01b604482015260640161041d565b6001818101546001600160a01b0316600090815260209190915260408120548154909190819083908110610572576105726113c7565b6000918252602090912060099091020180549091506001600160a01b03166105cf5760405162461bcd60e51b815260206004820152601060248201526f373796b9bab1b416b7b832b930ba37b960811b604482015260640161041d565b838360020160008282546105e3919061140c565b9091555050600283015460000361063b5760038581548110610607576106076113c7565b60009182526020822060039091020180546001600160a01b03191681556001810180546001600160c01b0319169055600201555b8381600501600082825461064f919061140c565b90915550508054336001600160a01b03909116036106dd578381600601600082825461067b919061140c565b90915550506007810154156106dd5769021e19e0c9bab24000008160060154116106dd5760405162461bcd60e51b8152602060048201526013602482015272746f6f2d6c6573732d73656c662d7374616b6560681b604482015260640161041d565b806005015460000361079557600082815481106106fc576106fc6113c7565b60009182526020808320600990920290910180546001600160a01b031916815560018082018490556002808301859055600383018590556004830185905560058301859055600683018590556007830185905560089092018490553384529182905260408320839055805491820181559091527f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace018290555b61079f33856111ba565b8054604080518781526020810187905233926001600160a01b0316917f4e4da858820d03358af7e05090375c4a8cfaddda3c24e48bd64e376f13d2c6bd91015b60405180910390a35050505050565b600381815481106107fe57600080fd5b60009182526020909120600390910201805460018201546002909201546001600160a01b03918216935090821691600160a01b900463ffffffff169084565b6000341161087f5760405162461bcd60e51b815260206004820152600f60248201526e6465706f7369742d6e6f7468696e6760881b604482015260640161041d565b604080516080810182523381526001600160a01b03838116602080840182815263ffffffff4281168688019081523460608801908152600380546001818101835560008381529a51919092027fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b81018054928b166001600160a01b03199093169290921790915594517fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85c860180549451909516600160a01b026001600160c01b031990941698169790971791909117909155517fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85d9091015590835252908120548154909190819083908110610996576109966113c7565b6000918252602090912060099091020180549091506001600160a01b038481169116146109f85760405162461bcd60e51b815260206004820152601060248201526f373796b9bab1b416b7b832b930ba37b960811b604482015260640161041d565b34816005016000828254610a0c91906113f3565b9091555050336001600160a01b03841603610a3b5734816006016000828254610a3591906113f3565b90915550505b805460035433916001600160a01b0316907febc590f21987ecf5c1f8466f90a1a5d112eec48f1c2e652a65c42034cc2c107a90610a7a9060019061140c565b604080519182523460208301520160405180910390a3505050565b8360ff1660021480610aaa57508360ff166003145b610aee5760405162461bcd60e51b81526020600482015260156024820152740d2dcecc2d8d2c85ae0eac4d6caf25ae0e4caccd2f605b1b604482015260640161041d565b69021e19e0c9bab2400000341015610b3b5760405162461bcd60e51b815260206004820152601060248201526f6465706f7369742d746f6f2d6c65737360801b604482015260640161041d565b336000908152600160205260409020548015610b8c5760405162461bcd60e51b815260206004820152601060248201526f1bdc195c985d1bdc8b595e1a5cdd195960821b604482015260640161041d565b60005415610c0b57336001600160a01b031660008081548110610bb157610bb16113c7565b60009182526020909120600990910201546001600160a01b031603610c0b5760405162461bcd60e51b815260206004820152601060248201526f1bdc195c985d1bdc8b595e1a5cdd195960821b604482015260640161041d565b60025415610d61576002805460009190610c279060019061140c565b81548110610c3757610c376113c7565b906000526020600020015490506002805480610c5557610c5561141f565b60019003818190600052602060002001600090559055604051806101200160405280336001600160a01b031681526020018760ff16815260200186815260200185815260200184815260200134815260200134815260200160008152602001600081525060008281548110610ccc57610ccc6113c7565b6000918252602080832084516009939093020180546001600160a01b0319166001600160a01b03909316929092178255838101516001808401919091556040808601516002850155606086015160038501556080860151600485015560a0860151600585015560c0860151600685015560e0860151600785015561010090950151600890930192909255338352522055610f51565b604080516101208101825233815260ff87166020820190815291810186815260608201868152608083018681523460a0850181815260c08601918252600060e08701818152610100880182815282546001808201855584805299517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563600990920291820180546001600160a01b0319166001600160a01b0390921691909117905599517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e5648b015596517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e5658a015594517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56689015592517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e567880155517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e568870155517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56986015590517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56a85015590517f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56b909301929092559054610f40919061140c565b336000908152600160205260409020555b60408051608081018252338082526020820181815263ffffffff4281168486019081523460608601818152600380546001810182556000829052975197027fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b810180546001600160a01b03998a166001600160a01b031990911617905594517fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85c860180549451909516600160a01b026001600160c01b03199094169716969096179190911790915592517fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85d9091015591517ff60326b9d09410d6ce5b777e4d80932914842920440df63a64bd2413b2f5ff1c9161109891899189918991899160ff959095168552602085019390935260408401919091526060830152608082015260a00190565b60405180910390a2600354339081907febc590f21987ecf5c1f8466f90a1a5d112eec48f1c2e652a65c42034cc2c107a906110d59060019061140c565b60408051918252346020830152016107df565b600081815481106110f857600080fd5b60009182526020909120600990910201805460018201546002830154600384015460048501546005860154600687015460078801546008909801546001600160a01b0390971698509496939592949193909289565b600080548061115e5761115e61141f565b60008281526020812060096000199093019283020180546001600160a01b031916815560018101829055600281018290556003810182905560048101829055600581018290556006810182905560078101829055600801559055565b8047101561120a5760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a20696e73756666696369656e742062616c616e6365000000604482015260640161041d565b6000826001600160a01b03168260405160006040518083038185875af1925050503d8060008114611257576040519150601f19603f3d011682016040523d82523d6000602084013e61125c565b606091505b50509050806112d35760405162461bcd60e51b815260206004820152603a60248201527f416464726573733a20756e61626c6520746f2073656e642076616c75652c207260448201527f6563697069656e74206d61792068617665207265766572746564000000000000606482015260840161041d565b505050565b600080604083850312156112eb57600080fd5b50508035926020909101359150565b60006020828403121561130c57600080fd5b5035919050565b60006020828403121561132557600080fd5b81356001600160a01b038116811461133c57600080fd5b9392505050565b60008060008060008060c0878903121561135c57600080fd5b505084359660208601359650604086013595606081013595506080810135945060a0013592509050565b6000806000806080858703121561139c57600080fd5b843560ff811681146113ad57600080fd5b966020860135965060408601359560600135945092505050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b80820180821115611406576114066113dd565b92915050565b81810381811115611406576114066113dd565b634e487b7160e01b600052603160045260246000fdfea2646970667358221220966a44d836692fcff37c4b6f6d21b2706cf0f5f9e65d4ac46f2cf83dc0fecde764736f6c63430008100033`

	// CCMonitorsGovForStorageTest.sol
	monitorsGovBytecode = `0x608060405234801561001057600080fd5b50610ece806100206000396000f3fe60806040526004361061007b5760003560e01c80635a627dbc1161004e5780635a627dbc1461015a5780636d98985814610162578063939624ab14610175578063f8aabdd41461019557600080fd5b80630eb1b5861461008057806344a58781146100a957806347998be314610108578063562aa2011461012a575b600080fd5b34801561008c57600080fd5b5061009660005481565b6040519081526020015b60405180910390f35b3480156100b557600080fd5b506100c96100c4366004610d4f565b610321565b604080516001600160a01b0390981688526020880196909652948601939093526060850191909152608084015260a083015260c082015260e0016100a0565b34801561011457600080fd5b50610128610123366004610d4f565b600055565b005b34801561013657600080fd5b5061014a610145366004610d68565b61037a565b60405190151581526020016100a0565b6101286103f9565b610128610170366004610d98565b610518565b34801561018157600080fd5b50610128610190366004610d4f565b6109ac565b3480156101a157600080fd5b506101286101b0366004610dd3565b6040805160e081018252338152602081019586529081019384526060810192835260808101918252600060a0820181815260c08301828152600180548082018255935292517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6600790930292830180546001600160a01b0319166001600160a01b0390921691909117905595517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf782015593517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf885015591517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf9840155517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cfa83015591517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cfb82015590517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cfc90910155565b6001818154811061033157600080fd5b600091825260209091206007909102018054600182015460028301546003840154600485015460058601546006909601546001600160a01b039095169650929491939092919087565b600154600090810361038e57506000919050565b6001600160a01b03821660009081526002602052604081205460018054919291839081106103be576103be610e05565b6000918252602090912060079091020180549091506001600160a01b0385811691161480156103f1575060008160050154115b949350505050565b6001546104215760405162461bcd60e51b815260040161041890610e1b565b60405180910390fd5b33600090815260026020526040812054600180549192918390811061044857610448610e05565b6000918252602090912060079091020180549091506001600160a01b031633146104845760405162461bcd60e51b815260040161041890610e1b565b600034116104c65760405162461bcd60e51b815260206004820152600f60248201526e6465706f7369742d6e6f7468696e6760881b6044820152606401610418565b348160040160008282546104da9190610e56565b909155505060405134815233907fb218e74e1d7548db0bff4bc81f75f5b5d41d66cf9151f0311fbcb7344bd9d0339060200160405180910390a25050565b8260ff166002148061052d57508260ff166003145b6105715760405162461bcd60e51b81526020600482015260156024820152740d2dcecc2d8d2c85ae0eac4d6caf25ae0e4caccd2f605b1b6044820152606401610418565b69152d02c7e14af68000003410156105be5760405162461bcd60e51b815260206004820152601060248201526f6465706f7369742d746f6f2d6c65737360801b6044820152606401610418565b33600090815260026020526040902054801561060e5760405162461bcd60e51b815260206004820152600f60248201526e1b5bdb9a5d1bdc8b595e1a5cdd1959608a1b6044820152606401610418565b6001541561068d57336001600160a01b0316600160008154811061063457610634610e05565b60009182526020909120600790910201546001600160a01b03160361068d5760405162461bcd60e51b815260206004820152600f60248201526e1b5bdb9a5d1bdc8b595e1a5cdd1959608a1b6044820152606401610418565b600354156107c45760038054600091906106a990600190610e6f565b815481106106b9576106b9610e05565b9060005260206000200154905060038054806106d7576106d7610e82565b600190038181906000526020600020016000905590556040518060e00160405280336001600160a01b031681526020018660ff1681526020018581526020018481526020013481526020016000815260200160008152506001828154811061074157610741610e05565b6000918252602080832084516007939093020180546001600160a01b0319166001600160a01b03909316929092178255838101516001830155604080850151600280850191909155606086015160038501556080860151600485015560a0860151600585015560c09095015160069093019290925533835292909252205561095a565b6040805160e08101825233815260ff861660208201908152918101858152606082018581523460808401908152600060a0850181815260c086018281526001805480820182559381905296517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6600790940293840180546001600160a01b0319166001600160a01b0390921691909117905596517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf783015593517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf882015591517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf9830155517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cfa82015590517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cfb82015591517fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cfc9092019190915580546109499190610e6f565b336000908152600260205260409020555b6040805160ff861681526020810185905290810183905234606082015233907f3ca1553e5e289d81faa2afbcf3f32d4df4c14526b7e2a81cd1bb25d1b1415b5d9060800160405180910390a250505050565b6001546109cb5760405162461bcd60e51b815260040161041890610e1b565b3360009081526002602052604081205460018054919291839081106109f2576109f2610e05565b6000918252602090912060079091020180549091506001600160a01b03163314610a2e5760405162461bcd60e51b815260040161041890610e1b565b8281600401541015610a765760405162461bcd60e51b81526020600482015260116024820152700eed2e8d0c8e4c2ee5ae8dede5adaeac6d607b1b6044820152606401610418565b82816004016000828254610a8a9190610e6f565b9091555050600481015469152d02c7e14af68000001115610b4357600581015415610aeb5760405162461bcd60e51b81526020600482015260116024820152706d6f6e69746f722d69732d61637469766560781b6044820152606401610418565b620d2f0060005442610afd9190610e6f565b10610b435760405162461bcd60e51b81526020600482015260166024820152756f7574736964652d756e7374616b652d77696e646f7760501b6044820152606401610418565b8060040154600003610bed5760018281548110610b6257610b62610e05565b60009182526020808320600790920290910180546001600160a01b03191681556001808201849055600280830185905560038084018690556004840186905560058401869055600690930185905533855290925260408320839055805491820181559091527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b018290555b610bf73384610c31565b60405183815233907f551255bcb3977c75cd031d9bc6d8233f1491dd8bcfe857b996b0afb99b5c3d399060200160405180910390a2505050565b80471015610c815760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a20696e73756666696369656e742062616c616e63650000006044820152606401610418565b6000826001600160a01b03168260405160006040518083038185875af1925050503d8060008114610cce576040519150601f19603f3d011682016040523d82523d6000602084013e610cd3565b606091505b5050905080610d4a5760405162461bcd60e51b815260206004820152603a60248201527f416464726573733a20756e61626c6520746f2073656e642076616c75652c207260448201527f6563697069656e74206d617920686176652072657665727465640000000000006064820152608401610418565b505050565b600060208284031215610d6157600080fd5b5035919050565b600060208284031215610d7a57600080fd5b81356001600160a01b0381168114610d9157600080fd5b9392505050565b600080600060608486031215610dad57600080fd5b833560ff81168114610dbe57600080fd5b95602085013595506040909401359392505050565b60008060008060808587031215610de957600080fd5b5050823594602084013594506040840135936060013592509050565b634e487b7160e01b600052603260045260246000fd5b6020808252600b908201526a3737ba16b6b7b734ba37b960a91b604082015260600190565b634e487b7160e01b600052601160045260246000fd5b80820180821115610e6957610e69610e40565b92915050565b81810381811115610e6957610e69610e40565b634e487b7160e01b600052603160045260246000fdfea2646970667358221220fa0479c49a29c210e458cffaac5f7dc1d14bc229f2220bc1bca131ee17d48f9d64736f6c63430008100033`
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

	addOperator1 := packAddOperatorData(02, "pubkeyX_o1", "12.34.56.78:9011", "operator#1", big.NewInt(1011), big.NewInt(1012))
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOperator1)
	_app.EnsureTxSuccess(tx.Hash())

	addOperator2 := packAddOperatorData(03, "pubkeyX_o2", "12.34.56.78:9012", "operator#2", big.NewInt(2011), big.NewInt(2012))
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOperator2)
	_app.EnsureTxSuccess(tx.Hash())

	addOperator3 := packAddOperatorData(02, "pubkeyX_o3", "12.34.56.78:9013", "operator#3", big.NewInt(3011), big.NewInt(3012))
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
	require.Equal(t, "12.34.56.78:9011",
		strings.TrimRight(string(operators[0].RpcUrl), string([]byte{0})))
	require.Equal(t, "operator#1",
		strings.TrimRight(string(operators[0].Intro), string([]byte{0})))
	require.Equal(t, uint64(1011), operators[0].SelfStakedAmt.Uint64())
	require.Equal(t, uint64(1012), operators[0].TotalStakedAmt.Uint64())
	require.Equal(t, uint64(0), operators[0].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operators[0].OldElectedTime.Uint64())

	require.Equal(t, addr, operators[1].Addr)
	require.Equal(t, "037075626b6579585f6f3200000000000000000000000000000000000000000000",
		hex.EncodeToString(operators[1].Pubkey))
	require.Equal(t, "12.34.56.78:9012",
		strings.TrimRight(string(operators[1].RpcUrl), string([]byte{0})))
	require.Equal(t, "operator#2",
		strings.TrimRight(string(operators[1].Intro), string([]byte{0})))
	require.Equal(t, uint64(2011), operators[1].SelfStakedAmt.Uint64())
	require.Equal(t, uint64(2012), operators[1].TotalStakedAmt.Uint64())
	require.Equal(t, uint64(0), operators[1].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operators[1].OldElectedTime.Uint64())

	require.Equal(t, addr, operators[2].Addr)
	require.Equal(t, "027075626b6579585f6f3300000000000000000000000000000000000000000000",
		hex.EncodeToString(operators[2].Pubkey))
	require.Equal(t, "12.34.56.78:9013",
		strings.TrimRight(string(operators[2].RpcUrl), string([]byte{0})))
	require.Equal(t, "operator#3",
		strings.TrimRight(string(operators[2].Intro), string([]byte{0})))
	require.Equal(t, uint64(3011), operators[2].SelfStakedAmt.Uint64())
	require.Equal(t, uint64(3012), operators[2].TotalStakedAmt.Uint64())
	require.Equal(t, uint64(0), operators[2].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operators[2].OldElectedTime.Uint64())

	crosschain.WriteOperatorElectedTime(ctx, seq, 1, 123)
	operators = crosschain.ReadOperatorInfos(ctx, seq)
	require.Equal(t, uint64(0), operators[0].ElectedTime.Uint64())
	require.Equal(t, uint64(123), operators[1].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operators[2].ElectedTime.Uint64())

	crosschain.WriteOperatorOldElectedTime(ctx, seq, 2, 456)
	operators = crosschain.ReadOperatorInfos(ctx, seq)
	require.Equal(t, uint64(0), operators[0].OldElectedTime.Uint64())
	require.Equal(t, uint64(0), operators[1].OldElectedTime.Uint64())
	require.Equal(t, uint64(456), operators[2].OldElectedTime.Uint64())
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
	require.Equal(t, "monitor#1",
		strings.TrimRight(string(monitors[0].Intro), string([]byte{0})))
	require.Equal(t, uint64(8001), monitors[0].StakedAmt.Uint64())
	require.Equal(t, uint64(0), monitors[0].ElectedTime.Uint64())

	require.Equal(t, addr, monitors[1].Addr)
	require.Equal(t, "037075626b6579585f6d3200000000000000000000000000000000000000000000",
		hex.EncodeToString(monitors[1].Pubkey))
	require.Equal(t, "monitor#2",
		strings.TrimRight(string(monitors[1].Intro), string([]byte{0})))
	require.Equal(t, uint64(8002), monitors[1].StakedAmt.Uint64())
	require.Equal(t, uint64(0), monitors[1].ElectedTime.Uint64())

	require.Equal(t, addr, monitors[2].Addr)
	require.Equal(t, "027075626b6579585f6d3300000000000000000000000000000000000000000000",
		hex.EncodeToString(monitors[2].Pubkey))
	require.Equal(t, "monitor#3",
		strings.TrimRight(string(monitors[2].Intro), string([]byte{0})))
	require.Equal(t, uint64(8003), monitors[2].StakedAmt.Uint64())
	require.Equal(t, uint64(0), monitors[2].ElectedTime.Uint64())

	crosschain.WriteMonitorElectedTime(ctx, seq, 1, 12345)
	monitors = crosschain.ReadMonitorInfos(ctx, seq)
	require.Equal(t, uint64(0), monitors[0].ElectedTime.Uint64())
	require.Equal(t, uint64(12345), monitors[1].ElectedTime.Uint64())
	require.Equal(t, uint64(0), monitors[2].ElectedTime.Uint64())

	crosschain.WriteMonitorOldElectedTime(ctx, seq, 2, 54321)
	monitors = crosschain.ReadMonitorInfos(ctx, seq)
	require.Equal(t, uint64(0), monitors[0].OldElectedTime.Uint64())
	require.Equal(t, uint64(0), monitors[1].OldElectedTime.Uint64())
	require.Equal(t, uint64(54321), monitors[2].OldElectedTime.Uint64())

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
	for i := 0; i < 13; i++ {
		require.Equal(t, uint64(0), operatorInfos[12].OldElectedTime.Uint64())
	}

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
	// new elected time
	require.Equal(t, uint64(0), operatorInfos[0].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[1].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[2].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[3].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[4].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[5].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[6].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[7].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[8].ElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[9].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[10].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[11].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[12].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[13].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[14].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), operatorInfos[15].ElectedTime.Uint64())
	// old elected time
	require.Equal(t, uint64(0), operatorInfos[0].OldElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[1].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[2].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[3].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[4].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[5].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[6].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[7].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[8].OldElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[9].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[10].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[11].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), operatorInfos[12].OldElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[13].OldElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[14].OldElectedTime.Uint64())
	require.Equal(t, uint64(0), operatorInfos[15].OldElectedTime.Uint64())
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
	for i := 0; i < 7; i++ {
		require.Equal(t, uint64(0), monitorInfos[i].OldElectedTime.Uint64())
	}

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
	require.Equal(t, uint64(0x123456), lastElectionTime.Uint64())
	// new elected time
	require.Equal(t, uint64(0), monitorInfos[0].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), monitorInfos[1].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), monitorInfos[2].ElectedTime.Uint64())
	require.Equal(t, uint64(0x123456), monitorInfos[3].ElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[4].ElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[5].ElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[6].ElectedTime.Uint64())
	// old elected time
	require.Equal(t, uint64(0), monitorInfos[0].OldElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[1].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), monitorInfos[2].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), monitorInfos[3].OldElectedTime.Uint64())
	require.Equal(t, uint64(0x12345), monitorInfos[4].OldElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[5].OldElectedTime.Uint64())
	require.Equal(t, uint64(0), monitorInfos[6].OldElectedTime.Uint64())
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
