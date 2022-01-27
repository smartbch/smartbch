// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract SEP101Proxy_DELEGATECALL {

    // solhint-disable-next-line no-complex-fallback
    fallback() payable external {
        // solhint-disable-next-line no-inline-assembly
        address _impl = address(0x2712);
        assembly {
            let ptr := mload(0x40)
            calldatacopy(ptr, 0, calldatasize())
            let result := delegatecall(gas(), _impl, ptr, calldatasize(), 0, 0)
            let size := returndatasize()
            returndatacopy(ptr, 0, size)
            switch result
            case 0 { revert(ptr, size) }
            default { return(ptr, size) }
        }
    }

}
