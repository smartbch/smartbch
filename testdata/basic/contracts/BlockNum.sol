// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract BlockNum {

    function getHeight() external view returns (uint) {
        return block.number;
    }

    function getBalance(address addr) external view returns (uint) {
        return addr.balance;
    }

    function getCodeSize(address addr) external view returns (uint) {
        uint size;
        assembly { size := extcodesize(addr) }
        return size;
    }

}
