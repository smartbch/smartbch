// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

interface SEP101 {

    function set(bytes calldata key, bytes calldata value) external;
    function get(bytes calldata key) external returns (bytes memory);

}