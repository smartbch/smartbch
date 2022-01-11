// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract BlockNum {

    function getHeight() external view returns (uint) {
        return block.number;
    }

}
