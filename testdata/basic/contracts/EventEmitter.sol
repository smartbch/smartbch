// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract EventEmitter {

    event Event1(address indexed addr);
    event Event2(address indexed addr, uint256 value);

    function emitEvent1() public {
      emit Event1(msg.sender);
    }

    function emitEvent2(uint256 n) public {
      emit Event2(msg.sender, n);
    }

}
