package bigutils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseU256(t *testing.T) {
	u, ok := ParseU256("65536")
	require.True(t, ok)
	require.Equal(t, "0x10000", u.Hex())

	u, ok = ParseU256("0x10000")
	require.True(t, ok)
	require.Equal(t, "0x10000", u.Hex())
}
