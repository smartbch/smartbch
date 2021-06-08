# Changelog



## v0.1.7 (not released yet)

* JSON-RPC
  * Add sbch_getTxListByHeightWithRange



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



