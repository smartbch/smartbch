// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

interface SEP101 {

    function set(uint key, bytes calldata value) external;
    function get(uint key) external view returns (bytes memory);

}