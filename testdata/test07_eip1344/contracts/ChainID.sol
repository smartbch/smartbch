// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract ChainID {

  function getChainID() public view returns (uint) {
    return block.chainid;
  }

}
