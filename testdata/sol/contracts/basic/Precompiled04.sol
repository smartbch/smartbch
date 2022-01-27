// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

// https://blocksecteam.medium.com/the-analysis-of-the-wild-exploitation-of-cve-2021-39137-f1c9ffcdd210
// https://docs.klaytn.com/smart-contract/precompiled-contracts#address-0x-04-datacopy-data
contract Precompiled04 {


    function callDatacopy1(bytes memory data) public returns (bytes memory) {
        bytes memory ret = new bytes(data.length);
        assembly {
            let len := mload(data)
            if iszero(call(gas(), 0x04, 0, add(data, 0x20), len, add(ret, 0x20), len)) {
                invalid()
            }
        }

        return ret;
    }

    function callDatacopy2(bytes memory data) public returns (bytes memory) {
        assembly {
            let len := mload(data)
            let len2 := sub(len, 7)
            if iszero(call(gas(), 0x04, 0, add(data, 0x20), len2, add(data, 0x27), len2)) {
                invalid()
            }
        }

        return data;
    }

    function callDatacopy3(bytes memory data, uint offset) public returns (bytes memory) {
        assembly {
            let len := mload(data)
            let len2 := sub(len, offset)
            if iszero(call(gas(), 0x04, 0, add(data, 0x20), len2, add(data, add(0x20, offset)), len2)) {
                invalid()
            }
        }

        return data;
    }

}
