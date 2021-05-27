# Changelog



## v0.1.6 (not released yet)

* JSON-RPC

  * sbch_getTxListByHeight returns more detailed tx info



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



