pragma solidity ^0.8.0;

contract TestHost {

  constructor() {}

  receive() external payable {}

  fallback() external payable {}

  function blkhash(uint blockNumber) public view returns (bytes32) {
    return blockhash(blockNumber);
  }

  function blocknumber() public view returns (uint) {
    return block.number;
  }

  function blockdifficulty() public view returns (uint) {
    return block.difficulty;
  }

  function blockgaslimit() public view returns (uint) {
    return block.gaslimit;
  }

  function blockcoinbase() public view returns (address) {
    return block.coinbase;
  }

  function blocktimestamp() public view returns (uint) {
    return block.timestamp;
  }

  function txsignatrue() public view returns (bytes4) {
    return msg.sig;
  }
}