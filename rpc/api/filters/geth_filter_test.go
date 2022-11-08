package filters

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/filters"
)

func TestFilterCriteria1(t *testing.T) {
	data := `
{
  "fromBlock": "0x1",
  "toBlock": "0x2",
  "address": "0x8888f1f195afa192cfee860698584c030f4c9db1",
  "topics": [
    ["0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b", null],
    [null, "0x0000000000000000000000000aff3454fce5edbc8cca8697c15331677e6ebccc"]
  ]
}
`
	fc := filters.FilterCriteria{}
	err := json.Unmarshal([]byte(data), &fc)
	require.NoError(t, err)
	require.Equal(t, [][]common.Hash{[]common.Hash(nil), []common.Hash(nil)}, fc.Topics)
}

func TestFilterCriteria2(t *testing.T) {
	data := `
{
  "fromBlock": "0x1",
  "toBlock": "0x2",
  "address": "0x8888f1f195afa192cfee860698584c030f4c9db1",
  "topics": [
    null,
    "0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b"
  ]
}
`
	fc := filters.FilterCriteria{}
	err := json.Unmarshal([]byte(data), &fc)
	require.NoError(t, err)
	require.Equal(t, [][]common.Hash{
		[]common.Hash(nil),
		{common.HexToHash("0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b")},
	}, fc.Topics)
}
