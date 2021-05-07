// SPDX-License-Identifier: MIT
pragma solidity >=0.7.0;

contract TestAdd {
    mapping(uint32 => uint) private data;

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
}

