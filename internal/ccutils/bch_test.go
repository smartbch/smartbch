package ccutils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetMultiSigP2SHAddr(t *testing.T) {
	redeemScriptWithoutConstructorArgs := "5d79009c63005e7a5f7a607a01117a01127a01137a56577a587a597a5a7a5b7a5c7a5d7a5e7a5f7a607a5aafc3519dc4519d00cd0376a914567a7e0288ac7e886d6d51675d79519c63005e7a5f7a607a01117a01127a01137a56577a587a597a5a7a5b7a5c7a5d7a5e7a5f7a607a5aaf00c600cc9d00cd02a914567a7e01877e886d6d51675d7a529d005d7a5e7a525d7a5e7a5f7a53af00c600cc9d00cd02a9145c7a7e01877e885601189502b40095b26d6d6d6d6d75516868"
	monitorPks := []string{
		"03d3a6177f842f4f2001d0656bb25ed4fe9c405ad105b1b4d4540848b4dffe8fd4",
		"023d74cef6dd43d2f427f6ea95d810e6f21788ec5572355b845dcddef22d727279",
		"02b8dcbf9108eeaef41224164bdc99872b13925bc8acd006152c29206fccc6b226",
	}
	operatorPks := []string{
		"03cf258b0f844e241a8a98c577671d130550086a70d1860d8aea233a0e3fc0d416",
		"032ea7f33796a1c59168b651e4a9785c45f6008e73a7452914987a5e36a9d1e88d",
		"0355de3b81ab920d31c9e3384662483002a8698b7b218579a74a9c401fafc804fd",
		"027e04942eb74fa58a28d18e07e330acd92aabab4e6bf2676628dffd038054373c",
		"034791574eaab305a02d77159adbdad0f3427f3e790604d7f55a6367472b78dc57",
		"0391b1b73a9cd1551ec53d10c2c4cfcce0ad96cc74a7523fb58c7848b8478d96d3",
		"03758c82f207cdec43834788fb9310e5b226db45ae640e6d49cbbeed8f08c8a636",
		"0347b38d253ae488213c1f3e4f6e11a98ad2c8e0ab23baef2e4c0226f123fe7b38",
		"02cfeedaf17ff28b4778b3d14f7a7659087c30717086b1641770e372ae0ffb5832",
		"03345a114a65de42476cebe60bf5c522f52ba8578775948a22465d0fb18f175d45",
	}

	addr, err := GetMultiSigP2SHAddr(redeemScriptWithoutConstructorArgs, operatorPks, monitorPks)
	require.NoError(t, err)
	require.Equal(t, "ppfuwr4yrfmjjkjvys6z80vx5mtvm8wqdylpy9m70n", addr)
}