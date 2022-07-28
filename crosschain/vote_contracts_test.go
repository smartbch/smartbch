package crosschain_test

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartbch/smartbch/crosschain"
	"github.com/smartbch/smartbch/internal/ethutils"
	"github.com/smartbch/smartbch/internal/testutils"
)

/*
//SPDX-License-Identifier: Unlicense
pragma solidity ^0.8.0;

contract CCVoting2ForTest {

	struct OperatorOrMonitorInfo {
		address addr;              // address
		uint    pubkeyPrefix;      // 0x02 or 0x03
		bytes32 pubkeyX;           // x
		bytes32 rpcUrl;            // ip:port (not used by monitors)
		bytes32 intro;             // introduction
		uint    totalStakedAmt;    // in BCH
		uint    selfStakedAmt;     // in BCH
		uint    inOfficeStartTime; // 0 means not in office, this field is set from Golang
	}

	struct StakeInfo {
		address staker;
		address monitor;
		address operator;
		uint32  stakedTime;
		uint    stakedAmt;
	}

	uint constant MONITOR_INIT_STAKE = 100_000 ether;
	uint constant OPERATOR_INIT_STAKE = 10_000 ether;
	uint constant MONITOR_MIN_STAKE_PERIOD = 200 days;
	uint constant OPERATOR_MIN_STAKE_PERIOD = 100 days;

	// read by Golang
	OperatorOrMonitorInfo[] monitors;
	OperatorOrMonitorInfo[] operators;

	mapping(address => uint) monitorIdxByAddr;
	mapping(address => uint) operatorIdxByAddr;

	uint lastStakeId;
	mapping(uint => StakeInfo) stakeById;
	mapping(address => uint[]) stakeIdsByAddr;

	function addMonitor(uint pubkeyPrefix,
		                bytes32 pubkeyX,
		                bytes32 rpcUrl,
		                bytes32 intro,
		                uint totalStakedAmt,
		                uint selfStakedAmt) public {
		monitors.push(OperatorOrMonitorInfo(msg.sender,
			pubkeyPrefix, pubkeyX, rpcUrl, intro, totalStakedAmt, selfStakedAmt, 0));
	}

	function addOperator(uint pubkeyPrefix,
		                 bytes32 pubkeyX,
		                 bytes32 rpcUrl,
		                 bytes32 intro,
		                 uint totalStakedAmt,
		                 uint selfStakedAmt) public {
		operators.push(OperatorOrMonitorInfo(msg.sender,
			pubkeyPrefix, pubkeyX, rpcUrl, intro, totalStakedAmt, selfStakedAmt, 0));
	}

}
*/
func TestCCVoting2(t *testing.T) {
	_abi := ethutils.MustParseABI(`
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
      "name": "addMonitor",
      "outputs": [],
      "stateMutability": "nonpayable",
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
    }
  ]
`)

	// CCVoting2ForTest.sol
	creationBytecode := testutils.HexToBytes(`0x608060405234801561001057600080fd5b506103c4806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c8063692ea8021461003b5780639717c20014610057575b600080fd5b610055600480360381019061005091906102c3565b610073565b005b610071600480360381019061006c91906102c3565b610186565b005b60016040518061010001604052803373ffffffffffffffffffffffffffffffffffffffff1681526020018881526020018781526020018681526020018581526020018481526020018381526020016000815250908060018154018082558091505060019003906000526020600020906008020160009091909190915060008201518160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506020820151816001015560408201518160020155606082015181600301556080820151816004015560a0820151816005015560c0820151816006015560e082015181600701555050505050505050565b60006040518061010001604052803373ffffffffffffffffffffffffffffffffffffffff1681526020018881526020018781526020018681526020018581526020018481526020018381526020016000815250908060018154018082558091505060019003906000526020600020906008020160009091909190915060008201518160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506020820151816001015560408201518160020155606082015181600301556080820151816004015560a0820151816005015560c0820151816006015560e082015181600701555050505050505050565b6000813590506102a881610360565b92915050565b6000813590506102bd81610377565b92915050565b60008060008060008060c087890312156102dc57600080fd5b60006102ea89828a016102ae565b96505060206102fb89828a01610299565b955050604061030c89828a01610299565b945050606061031d89828a01610299565b935050608061032e89828a016102ae565b92505060a061033f89828a016102ae565b9150509295509295509295565b6000819050919050565b6000819050919050565b6103698161034c565b811461037457600080fd5b50565b61038081610356565b811461038b57600080fd5b5056fea2646970667358221220e7fb25df78b19bc021c1890ed0410a43b670cfacc9b6486f7ff125722f0d76e664736f6c63430008040033`)

	key, addr := testutils.GenKeyAndAddr()
	_app := testutils.CreateTestApp(key)
	defer _app.Destroy()

	tx, _, contractAddr := _app.DeployContractInBlock(key, creationBytecode)
	_app.EnsureTxSuccess(tx.Hash())

	addMonitor1 := _abi.MustPack("addMonitor",
		big.NewInt(02),
		toBytes32("pubkeyX_m1"),
		toBytes32("12.34.56.78:9001"),
		toBytes32("monitor#1"),
		big.NewInt(1001),
		big.NewInt(1002),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addMonitor1)
	_app.EnsureTxSuccess(tx.Hash())

	addMonitor2 := _abi.MustPack("addMonitor",
		big.NewInt(03),
		toBytes32("pubkeyX_m2"),
		toBytes32("12.34.56.78:9002"),
		toBytes32("monitor#2"),
		big.NewInt(2001),
		big.NewInt(2002),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addMonitor2)
	_app.EnsureTxSuccess(tx.Hash())

	addOperator1 := _abi.MustPack("addOperator",
		big.NewInt(02),
		toBytes32("pubkeyX_o1"),
		toBytes32("12.34.56.78:9011"),
		toBytes32("operator#1"),
		big.NewInt(1011),
		big.NewInt(1012),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOperator1)
	_app.EnsureTxSuccess(tx.Hash())

	addOperator2 := _abi.MustPack("addOperator",
		big.NewInt(03),
		toBytes32("pubkeyX_o2"),
		toBytes32("12.34.56.78:9012"),
		toBytes32("operator#2"),
		big.NewInt(2011),
		big.NewInt(2012),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOperator2)
	_app.EnsureTxSuccess(tx.Hash())

	addOperator3 := _abi.MustPack("addOperator",
		big.NewInt(02),
		toBytes32("pubkeyX_o3"),
		toBytes32("12.34.56.78:9013"),
		toBytes32("operator#3"),
		big.NewInt(3011),
		big.NewInt(3012),
	)
	tx, _ = _app.MakeAndExecTxInBlock(key, contractAddr, 0, addOperator3)
	_app.EnsureTxSuccess(tx.Hash())

	// read data from Go
	ctx := _app.GetRpcContext()
	defer ctx.Close(false)

	accInfo := ctx.GetAccount(contractAddr)
	seq := accInfo.Sequence()

	monitors := crosschain.ReadOperatorOrMonitorArr(ctx, seq, 0)
	require.Len(t, monitors, 2)
	require.Equal(t, addr, monitors[0].Addr)
	require.Equal(t, "027075626b6579585f6d3100000000000000000000000000000000000000000000",
		hex.EncodeToString(monitors[0].Pubkey))
	require.Equal(t, "12.34.56.78:9001",
		strings.TrimRight(string(monitors[0].RpcUrl), string([]byte{0})))
	require.Equal(t, "monitor#1",
		strings.TrimRight(string(monitors[0].Intro), string([]byte{0})))
	require.Equal(t, uint64(1001), monitors[0].SelfStakedAmt.Uint64())
	require.Equal(t, uint64(1002), monitors[0].TotalStakedAmt.Uint64())
	require.Equal(t, uint64(0), monitors[0].InOfficeStartTime.Uint64())

	require.Equal(t, addr, monitors[1].Addr)
	require.Equal(t, "037075626b6579585f6d3200000000000000000000000000000000000000000000",
		hex.EncodeToString(monitors[1].Pubkey))
	require.Equal(t, "12.34.56.78:9002",
		strings.TrimRight(string(monitors[1].RpcUrl), string([]byte{0})))
	require.Equal(t, "monitor#2",
		strings.TrimRight(string(monitors[1].Intro), string([]byte{0})))
	require.Equal(t, uint64(2001), monitors[1].SelfStakedAmt.Uint64())
	require.Equal(t, uint64(2002), monitors[1].TotalStakedAmt.Uint64())
	require.Equal(t, uint64(0), monitors[1].InOfficeStartTime.Uint64())

	operators := crosschain.ReadOperatorOrMonitorArr(ctx, seq, 1)
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
	require.Equal(t, uint64(0), operators[0].InOfficeStartTime.Uint64())

	require.Equal(t, addr, operators[1].Addr)
	require.Equal(t, "037075626b6579585f6f3200000000000000000000000000000000000000000000",
		hex.EncodeToString(operators[1].Pubkey))
	require.Equal(t, "12.34.56.78:9012",
		strings.TrimRight(string(operators[1].RpcUrl), string([]byte{0})))
	require.Equal(t, "operator#2",
		strings.TrimRight(string(operators[1].Intro), string([]byte{0})))
	require.Equal(t, uint64(2011), operators[1].SelfStakedAmt.Uint64())
	require.Equal(t, uint64(2012), operators[1].TotalStakedAmt.Uint64())
	require.Equal(t, uint64(0), operators[1].InOfficeStartTime.Uint64())

	require.Equal(t, addr, operators[2].Addr)
	require.Equal(t, "027075626b6579585f6f3300000000000000000000000000000000000000000000",
		hex.EncodeToString(operators[2].Pubkey))
	require.Equal(t, "12.34.56.78:9013",
		strings.TrimRight(string(operators[2].RpcUrl), string([]byte{0})))
	require.Equal(t, "operator#3",
		strings.TrimRight(string(operators[2].Intro), string([]byte{0})))
	require.Equal(t, uint64(3011), operators[2].SelfStakedAmt.Uint64())
	require.Equal(t, uint64(3012), operators[2].TotalStakedAmt.Uint64())
	require.Equal(t, uint64(0), operators[2].InOfficeStartTime.Uint64())

	//operators[1].votes = 123
	//ctx2 := _app.GetRunTxContext()
	crosschain.WriteOperatorOrMonitorInOfficeStartTime(ctx, seq, 1, 1, 123)
	//ctx2.Close(true)
	operators = crosschain.ReadOperatorOrMonitorArr(ctx, seq, 1)
	require.Equal(t, uint64(0), operators[0].InOfficeStartTime.Uint64())
	require.Equal(t, uint64(123), operators[1].InOfficeStartTime.Uint64())
	require.Equal(t, uint64(0), operators[2].InOfficeStartTime.Uint64())
}

func toBytes32(s string) [32]byte {
	out := [32]byte{}
	copy(out[:], s)
	return out
}
