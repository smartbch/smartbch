// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract BlockHash2 {

    bytes32 public lastBlockHash;

    function saveLastBlockHash() public {
        lastBlockHash = blockhash(block.number - 1);
    }

}
