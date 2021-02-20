# Single Node Test



## start node

注意：chain ID目前必须是`moeing-(\d+)`这种模式。

```bash
$ go run github.com/moeing-chain/moeing-chain/cmd/moeingd init m1 --chain-id moeing-1
$ go run github.com/moeing-chain/moeing-chain/cmd/moeingd start
```



测试地址：

```
0xAb5D62788E207646fA60EB3eEbDC4358C7F5686c
0x3E144eB45c5fF912B2b29B2823FA674C972e9EC0
0x0C60B56403637dC9059fff3603a58DB3D5D76D38
0xEAB1B601da26611D134299845035214a046508B8
0x44F9ba3cfa79f1504f1C2d1eb0389fbB32e5A00c
0xb53e0a1dCf2ad9FA6ec8da77121B1765E68e768f
0x09F236e4067f5FcA5872d0c09f92Ce653377aE41
0xc5787370b6188b2b6F947117BB2F68ADF732b207
0xB9D95550558d2a163F77F5A523dFe605746cB95B
0xee5d82886766296640d8CA194e997341a0DeDEDe
```



## Test HTTP RPC



### eth_blockNumber

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    -H "Content-Type: application/json" http://localhost:8545

{"jsonrpc":"2.0","id":1,"result":"0x403"}
```



### eth_chainId

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
    -H "Content-Type: application/json" http://localhost:8545

{"jsonrpc":"2.0","id":1,"result":"0x1"}
```



### eth_coinbase

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_coinbase","params":[],"id":1}' \
    -H "Content-Type: application/json" http://localhost:8545

{"jsonrpc":"2.0","id":1,"result":"0x5125c9ad1e929422cba5a33da1c983eb0200107d"}
```



### eth_getBalance

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xAb5D62788E207646fA60EB3eEbDC4358C7F5686c", "latest"],"id":1}' \
    -H "Content-Type: application/json" http://localhost:8545

{"jsonrpc":"2.0","id":1,"result":"0x989680"}
```



### eth_getBlockByHash

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockByHash",
	"params":["0xfc1d81f318d1c96ca42cf3ed0f972c8cd9b2d8db8fdb2d0fd27afecf2cffc97d", true],"id":1}
	' -H "Content-Type: application/json" http://localhost:8545

```



### eth_getBlockByNumber

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber",
	"params":["0x2", true],"id":1}
	' -H "Content-Type: application/json" http://localhost:8545

```



### eth_getBlockTransactionCountByHash

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockTransactionCountByHash",
	"params":["0x35235261541972F209B80FDD3356299B2804EA13D0EFFB4835DC8E0AAA656A41"],"id":1}
	' -H "Content-Type: application/json" http://localhost:8545

```



### eth_getBlockTransactionCountByNumber

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockTransactionCountByNumber",
	"params":["0xe8"],"id":1}
	' -H "Content-Type: application/json" http://localhost:8545

```



### eth_getCode

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getCode",
	"params":["0x5f56Fe86a4e64678eE44186260174CE2883Ae9ef", "0x2"],"id":1}' \
	-H "Content-Type: application/json" http://localhost:8545

```



### eth_gasPrice

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_gasPrice","params":[],"id":1}' \
    -H "Content-Type: application/json" http://localhost:8545

{"jsonrpc":"2.0","id":1,"result":"0x0"}
```



### eth_getTransactionCount

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getTransactionCount",
	"params":["0xAb5D62788E207646fA60EB3eEbDC4358C7F5686c","latest"],"id":1}' \
	-H "Content-Type: application/json" http://localhost:8545

{"jsonrpc":"2.0","id":1,"result":"0x6"}
```



### eth_protocolVersion

```bash
$ curl -X POST --data '{"jsonrpc":"2.0","method":"eth_protocolVersion","params":[],"id":1}' \
    -H "Content-Type: application/json" http://localhost:8545

{"jsonrpc":"2.0","id":1,"result":"0x3f"}
```



### eth_sendTransaction

转账（`nonce`可以省略）：

```bash
$ curl -X POST --data '{
  "jsonrpc": "2.0",
  "method": "eth_sendTransaction",
  "params":[{
    "from": "0xAb5D62788E207646fA60EB3eEbDC4358C7F5686c",
    "to": "0x3E144eB45c5fF912B2b29B2823FA674C972e9EC0",
    "gas": "0x100000",
    "gasPrice": "0x1",
    "value": "0x100",
    "nonce":"0x0"
  }] , "id":1}' -H "Content-Type: application/json" http://localhost:8545

{"jsonrpc":"2.0","id":1,"result":"0x66be5e4312ac7d97146b704cc446fcd269dbbaa35ec3f82ec457bc9f16063f5d"}
```

部署合约：

```bash
$ curl -X POST --data '{
  "jsonrpc": "2.0",
  "method": "eth_sendTransaction",
  "params":[{
    "from": "0xAb5D62788E207646fA60EB3eEbDC4358C7F5686c",
    "gas": "0x100000",
    "gasPrice": "0x1",
    "value": "0x10000",
    "data": "0x608060405234801561001057600080fd5b5060cc8061001f6000396000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c806361bc221a1460375780636299a6ef146053575b600080fd5b603d607e565b6040518082815260200191505060405180910390f35b607c60048036036020811015606757600080fd5b81019080803590602001909291905050506084565b005b60005481565b8060008082825401925050819055505056fea264697066735822122037865cfcfd438966956583c78d31220c05c0f1ebfd116aced883214fcb1096c664736f6c634300060c0033"
  }],
  "id":1
  }' -H "Content-Type: application/json" http://localhost:8545


{"jsonrpc":"2.0","id":1,"result":"0xe4500d8db6e3169701f57e41aa18e959feb5966a6d72cd0bd0585933d0c5962c"}

# Server logs:
# Submitted contract creation, tx hash: 0xe4500d8db6e3169701f57e41aa18e959feb5966a6d72cd0bd0585933d0c5962c contract addr: 0x5f56Fe86a4e64678eE44186260174CE2883Ae9ef

```









## Test WS-RPC

Install [wscat](https://github.com/websockets/wscat).

```bash
$ wscat -c ws://localhost:8546
Connected (press CTRL+C to quit)
> {"jsonrpc":"2.0","method":"eth_protocolVersion","params":[],"id":1}
< {"jsonrpc":"2.0","id":1,"result":"0x3f"}
```

pub/unpub

```bash
$ wscat -c ws://localhost:8546
Connected (press CTRL+C to quit)
> {"jsonrpc":"2.0", "method":"eth_subscribe", "params":["newHeads"], "id":1}
< ...
```

