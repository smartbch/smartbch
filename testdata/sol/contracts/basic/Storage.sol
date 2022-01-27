// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract Storage {

    uint256[16] private slots;

    function set(uint256 key, uint256 val) public {
        slots[key] = val;
    }
    function get(uint256 key) public view returns (uint256) {
        return slots[key];
    }

}
