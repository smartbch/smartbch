package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseChainID(t *testing.T) {
	_, err := parseChainID("smartbch-1")
	require.Error(t, err)

	_, err = parseChainID("123")
	require.Error(t, err)

	_, err = parseChainID("0x")
	require.Error(t, err)

	id, err := parseChainID("0x123")
	require.NoError(t, err)
	require.Equal(t, id.Uint64(), uint64(0x123))
}
