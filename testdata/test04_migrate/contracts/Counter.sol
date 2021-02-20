// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract Counter {

  int public counter;

  function update(int n) public {
    counter += n;
  }

}
