// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "./SEP101Proxy.sol";

contract SEP101Proxy2 is SEP101Proxy {

    function set_zero_len_key(bytes calldata value) external {
        agent.staticcall(abi.encodeWithSelector(_SELECTOR_SET, new bytes(0), value));
    }
    function get_zero_len_key() external returns (bytes memory) {
        (bool success, bytes memory data) = agent.staticcall(abi.encodeWithSelector(_SELECTOR_GET, new bytes(0)));
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
