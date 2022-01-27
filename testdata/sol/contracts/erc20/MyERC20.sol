// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "./SimpleERC20.sol";

contract MyERC20 is SimpleERC20 {
    constructor() SimpleERC20(100000000, "MyERC20", 18, "MYERC") {}
}
