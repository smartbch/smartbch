# Changelog

## v0.4.0 (not released yet)

BCH holders can elect validators through the XHedge smart contract.

An optimized scheme to decide the minimum gas price.

Refine SEP206:

  An EOA can only call SEP206's contract address once in a block

  When using "transferFrom", the source account must be left 0.001 BCH.

Add SEP109 to support VRF (verifiable random function)

Allows EOA to send transactions into mempool continuously, across several successive blocks

Add debug API (debug_* RPC endpoints)

Add internal transactions' information in transaction receipt

Support querying main chain blocks in parallel.

Rpcclient will retry after failure for better robustness.


* JSON-RPC
  * Support eth_subscribe (newHeads and logs)
  * Add sbch_getTransactionReceipt
  * Add sbch_getCCEpochs
  * Add sbch_getCurrEpoch
  * The following methods return internal transactions
    * sbch_getTxListByHeight
    * sbch_getTxListByHeightWithRange
  * The following methods support extra block parameter now
    * eth_getBalance
    * eth_getCode
    * eth_getTransactionCount
    * eth_getStorageAt
    * eth_call
    * eth_estimateGas
* Command
  * Add `--http.api`, `--ws.api` and `--archive-mode` options to `smartbchd start` command

## v0.3.5

* JSON-RPC
  * Add request logs
  * Improve eth_estimateGas
* Library
  * Upgrade moeingdb to v0.3.4
  * Upgrade moeingevm to v0.3.3

## v0.3.4

* JSON-RPC
  * Changed hardcoded HTTPS ports to parameters (PR#23)
  * Fixed a bug of eth_getBlockByNumber
  * Fixed eth_gasPrice
  * Fixed transaction's V,R,S (issue#25)
* Command
  * `smartbchd start` will not ignore `--home` option
  * `smartbchd start` can disable HTTPS-RPC and WSS-RPC server now
* Library
  * Upgrade moeingdb to v0.3.3
  * Upgrade moeingevm to v0.3.2


## v0.3.3

* Command
  * Add `--rpc-only` option to `smartbchd start` command
  * Fix a bug and improve `smartbchd staking` command
* Staking
  * Fixed getrawtransaction rpc call (PR#21)


## v0.3.2

* JSON-RPC
  * Add sbch_healthCheck
  * Fix a bug of eth_getLogs
* Library
  * Upgrade moeingevm to v0.3.1


## v0.3.0

* Command
  * Add `smartbchd version` command
* JSON-RPC
  * Improve web3_clientVersion
  * Ignore the height argument and always return latest status
* Consensus
  * Fix several staking bugs
* Library
  * MoeingADS uses internal multiple shards to boost performance
  * MoeingEVM update evmone to 0.8.0



## v0.2.0

* JSON-RPC
  * Add sbch_getTxListByHeightWithRange
  * Add tm_validatorsInfo
  * Add sbch_getEpochs
  * Fix bugs of several endpoints
* Command
  * Improve `smartbchd staking` command
* Consensus
  * Continue to enhance multi-validator support
  * Integrate with BCHN special testnode



## v0.1.6

* JSON-RPC

  * **sbch_getTxListByHeight** returns more detailed tx info
  * Add placeholder implementation for txpool namespace 
    * txpool_content
    * txpool_status
    * txpool_inspect

* Add toolkits for stress test

* Mempool
  * Add signature cache and SEP206 sender set to speed up tx-rechecking
  * Refuse incoming TXs when a lot of TXs need rechecking

* Consensus

  * Enhance multi-validator support
  * Customize BlockMaxBytes and BlockMaxGas for testing
  * Add some staking-related sub commands

* Docker

  * Refine docker scripts

* Storage

  * Sync MoeingADS to fix some bugs
  * Add pruning calls to MoeingADS

* Move the faucet out from this repo



## v0.1.5

* JSON-RPC

  * sbch_queryTxBySrc/Dst/Addr & sbch_queryLogs

    * allow `startHeight` to be greater than `endHeight`
    * add `limit` param

    please refer to  [JSON-RPC docs](https://github.com/smartbch/docs/blob/main/deverlopers-guide/jsonrpc.md#sbch_queryTxBySrc) for more detailed change

* Add config option to support lite history DB



## v0.1.3

* Fix some bugs



## v0.1.2

* JSON-RPC
  * Add tm_nodeInfo
* Fix some bugs



## v0.1.1

* SEP
  * Add initial SEP101 implementation
  * Add initial SEP206 implementation
* Consensus
  * Add initial multi-validator support
  * Add staking functions
* JSON-RPC
  * Fix some small bugs
  * Add endpoints for notification counters
* MoeingDB
  * Add cache to speedup the BLOCKHASH instruction for EVM



