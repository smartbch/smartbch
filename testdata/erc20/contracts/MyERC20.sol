// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "./ERC20.sol";

contract MyERC20 is ERC20 {
    constructor() ERC20(100000000, "MyERC20", 18, "MYERC") {}
}
