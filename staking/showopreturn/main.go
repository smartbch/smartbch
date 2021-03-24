package main

import (
	"github.com/smartbch/smartbch/staking"
)

func main() {
	client := staking.NewRpcClient("http://127.0.0.1:8332/", "user", "dummypassword")

	//client.PrintAllOpReturn(1, 219139)
	client.PrintAllOpReturn(519995, 679995)
}
