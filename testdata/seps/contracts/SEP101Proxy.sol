// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "./interfaces/ISEP101.sol";

contract SEP101Proxy is ISEP101 {

    bytes4 public constant _SELECTOR_SET = bytes4(keccak256(bytes("set(bytes,bytes)")));
    bytes4 public constant _SELECTOR_GET = bytes4(keccak256(bytes("get(bytes)")));

    address constant public agent = address(0x2712);
    bytes public resultOfGet;

    function set(bytes calldata key, bytes calldata value) override external {
        (bool success, bytes memory data) = agent.delegatecall(abi.encodeWithSelector(_SELECTOR_SET, key, value));
        require(success, string(data));
    }
    function get(bytes calldata key) override external returns (bytes memory) {
        (bool success, bytes memory data) = agent.delegatecall(abi.encodeWithSelector(_SELECTOR_GET, key));
        require(success, string(data));
        resultOfGet = abi.decode(data, (bytes));
        return resultOfGet;
    }

    // // solhint-disable-next-line no-complex-fallback
    // fallback() payable external {
    //     // solhint-disable-next-line no-inline-assembly
    //     address _impl = address(0x2712);
    //     assembly {
    //         let ptr := mload(0x40)
    //         calldatacopy(ptr, 0, calldatasize())
    //         let result := delegatecall(gas(), _impl, ptr, calldatasize(), 0, 0)
    //         let size := returndatasize()
    //         returndatacopy(ptr, 0, size)
    //         switch result
    //         case 0 { revert(ptr, size) }
    //         default { return(ptr, size) }
    //     }
    // }

}
