// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "./interfaces/SEP101.sol";

contract SEP101Proxy is SEP101 {

    SEP101 constant public agent = SEP101(address(0x2711));

    function set(bytes calldata key, bytes calldata value) override external {
        agent.set(key, value);
    }
    function get(bytes calldata key) override external view returns (bytes memory) {
        return agent.get(key);
    }

}
