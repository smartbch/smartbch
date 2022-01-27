// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract StakingTest2 {

    // solhint-disable-next-line no-complex-fallback
    fallback() payable external {
        // solhint-disable-next-line no-inline-assembly
        address _impl = address(0x2710);
        assembly {
            let ptr := mload(0x40)
            calldatacopy(ptr, 0, calldatasize())
            let result := call(gas(), _impl, 0, ptr, calldatasize(), 0, 0)
            let size := returndatasize()
            returndatacopy(ptr, 0, size)
            switch result
            case 0 { revert(ptr, size) }
            default { return(ptr, size) }
        }
    }

}
