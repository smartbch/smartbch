package param

import (
	"os"
	"path/filepath"

	"github.com/tendermint/tendermint/config"
)

const (
	DefaultRpcEthGetLogsMaxResults = 10000
	DefaultRetainBlocks            = -1
)

type ChainConfig struct {
	//ChainID *big.Int `json:"chainId"` // chainId identifies the current chain and is used for replay protection
	//
	//HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)
	//
	//DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
	//DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whether the nodes supports or opposes the DAO hard-fork
	//
	//// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	//EIP150Block *big.Int    `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	//EIP150Hash  common.Hash `json:"eip150Hash,omitempty"`  // EIP150 HF hash (needed for header only clients as only gas pricing changed)
	//
	//EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
	//EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block
	//
	//ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	//ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)
	//PetersburgBlock     *big.Int `json:"petersburgBlock,omitempty"`     // Petersburg switch block (nil = same as Constantinople)
	//IstanbulBlock       *big.Int `json:"istanbulBlock,omitempty"`       // Istanbul switch block (nil = no fork, 0 = already on istanbul)
	//MuirGlacierBlock    *big.Int `json:"muirGlacierBlock,omitempty"`    // Eip-2384 (bomb delay) switch block (nil = no fork, 0 = already activated)
	//
	//YoloV2Block *big.Int `json:"yoloV2Block,omitempty"` // YOLO v2: Gas repricings TODO @holiman add EIP references
	//EWASMBlock  *big.Int `json:"ewasmBlock,omitempty"`  // EWASM switch block (nil = no fork, 0 = already activated)
	//
	//// Various consensus engines
	//Ethash *EthashConfig `json:"ethash,omitempty"`
	//Clique *CliqueConfig `json:"clique,omitempty"`
	NodeConfig *config.Config
	//app config:
	AppDataPath  string `json:"app_data_path,omitempty"`
	ModbDataPath string `json:"modb_data_path,omitempty"`

	// rpc config
	RpcEthGetLogsMaxResults int

	// db config
	RetainBlocks int64

	// Use LiteDB instead of MoDB
	UseLiteDB bool
}

var (
	home                = os.ExpandEnv("$HOME/.smartbchd")
	defaultAppDataPath  = filepath.Join(home, "data", "app")
	defaultModbDataPath = filepath.Join(home, "data", "modb")
)

func DefaultConfig() *ChainConfig {
	os.LookupEnv("HOME")
	c := &ChainConfig{
		NodeConfig:              config.DefaultConfig(),
		AppDataPath:             defaultAppDataPath,
		ModbDataPath:            defaultModbDataPath,
		RpcEthGetLogsMaxResults: DefaultRpcEthGetLogsMaxResults,
		RetainBlocks:            DefaultRetainBlocks,
	}
	c.NodeConfig.TxIndex.Indexer = "null"
	return c
}
