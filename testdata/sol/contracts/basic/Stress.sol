// SPDX-License-Identifier: MIT
pragma solidity >=0.7.0;

contract Stress {
    mapping(uint32 => uint) private data;

    function run0(address to, uint32 offset) payable external {
        to.call{value: msg.value, gas: 9000}(new bytes(0));
        for(uint32 i = offset; i < offset + 10; i++) {
            data[i] = msg.value/3;
        }
    }

    function run(address to, uint256 param) payable external {
        to.call{value: msg.value, gas: 9000}(new bytes(0));
        uint32 a = uint32(param>>(32*0));
        uint32 b = uint32(param>>(32*1));
        uint32 c = uint32(param>>(32*2));
        uint32 x = uint32(param>>(32*3));
        uint32 y = uint32(param>>(32*4));
        uint32 z = uint32(param>>(32*5));
        data[c] = (data[a] + data[b] + msg.value)/2;
        data[z] = (data[x] + data[y] + msg.value)/2;
    }

    function run2(address to, address addr1, address addr2, uint256 param) payable external {
        to.call{value: msg.value/2, gas: 9000}(new bytes(0));
        uint32 a = uint32(param>>(32*0));
        uint32 b = uint32(param>>(32*1));
        uint32 c = uint32(param>>(32*2));
        uint32 x = uint32(param>>(32*3));
        uint32 y = uint32(param>>(32*4));
        uint32 z = uint32(param>>(32*5));
        data[c] = (data[a] + data[b] + msg.value)/2;
        data[z] = (data[x] + data[y] + msg.value)/2;
        Stress(addr1).run{value: msg.value/3}(addr2, param);
        Stress(addr2).run{value: msg.value/9}(addr1, param);
    }

    function get(uint32 d) external view returns (uint) {
        return data[d];
    }
}
