package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseChainID(t *testing.T) {
	id, err := parseChainID("moeing-1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id.Uint64())

	id, err = parseChainID("moeing-123")
	require.NoError(t, err)
	require.Equal(t, uint64(123), id.Uint64())

	id, err = parseChainID("moning-1")
	require.Error(t, err)
	require.Equal(t, "invalid chain ID: moning-1", err.Error())

	id, err = parseChainID("moeing-abc")
	require.Error(t, err)
	require.Equal(t, "invalid chain ID: moeing-abc", err.Error())
}
