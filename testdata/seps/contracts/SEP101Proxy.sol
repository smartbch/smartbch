// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "./interfaces/ISEP101.sol";

contract SEP101Proxy is ISEP101 {

    bytes4 private constant _SELECTOR_SET = bytes4(keccak256(bytes("set(bytes,bytes)")));
    bytes4 private constant _SELECTOR_GET = bytes4(keccak256(bytes("get(bytes)")));

    address constant public agent = address(0x2712);
    bytes public resultOfGet;

    function set(bytes calldata key, bytes calldata value) override external {
        agent.delegatecall(abi.encodeWithSelector(_SELECTOR_SET, key, value));
    }
    function get(bytes calldata key) override external returns (bytes memory) {
        (bool success, bytes memory data) = agent.delegatecall(abi.encodeWithSelector(_SELECTOR_GET, key));
        resultOfGet = abi.decode(data, (bytes));
        return resultOfGet;
    }

    // CompileError: TypeError: "callcode" has been deprecated in favour of "delegatecall".
    // function set_callcode(bytes calldata key, bytes calldata value) external {
    //     agent.callcode(abi.encodeWithSelector(_SELECTOR_SET, key, value));
    // }
    // function get_callcode(bytes calldata key) external returns (bytes memory) {
    //     (bool success, bytes memory data) = agent.callcode(abi.encodeWithSelector(_SELECTOR_GET, key));
    //     resultOfGet = abi.decode(data, (bytes));
    //     return resultOfGet;
    // }

    function set_call(bytes calldata key, bytes calldata value) external {
        agent.call(abi.encodeWithSelector(_SELECTOR_SET, key, value));
    }
    function get_call(bytes calldata key) external returns (bytes memory) {
        (bool success, bytes memory data) = agent.call(abi.encodeWithSelector(_SELECTOR_GET, key));
        resultOfGet = abi.decode(data, (bytes));
        return resultOfGet;
    }

    function set_staticcall(bytes calldata key, bytes calldata value) external {
        agent.staticcall(abi.encodeWithSelector(_SELECTOR_SET, key, value));
    }
    function get_staticcall(bytes calldata key) external returns (bytes memory) {
        (bool success, bytes memory data) = agent.staticcall(abi.encodeWithSelector(_SELECTOR_GET, key));
        resultOfGet = abi.decode(data, (bytes));
        return resultOfGet;
    }

}
