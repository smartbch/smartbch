// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract Contract1 {

    event Call(uint256 id);

    uint256 public counter;
    address public contract2;
    address public contract3;

    constructor(address _contract2, address _contract3) {
        contract2 = _contract2;
        contract3 = _contract3;
    }

    function call2(uint256 id) public returns (uint256) {
        emit Call(id);
        counter++;
        Contract2(contract2).call3(id + 1);
        Contract2(contract2).call3(id + 5);
        return id << 64;
    }
    function call3(uint256 id) public returns (uint256) {
        emit Call(id);
        counter++;
        Contract3(contract3).callMe(id + 1);
        Contract3(contract3).callMe(id + 5);
        return id << 64;
    }

}

contract Contract2 {

    uint256 public counter;
    address public contract3;

    constructor(address _contract3) {
        contract3 = _contract3;
    }

    function call3(uint256 id) public returns (uint256) {
        counter++;
        Contract3(contract3).callMe(id + 1);
        Contract3(contract3).callMeStatic(id + 2);
        return id << 64;
    }

}

contract Contract3 {

    uint256 public counter;

    function callMe(uint256 id) public returns (uint256) {
        counter++;
        return id << 64;
    }
    function callMeStatic(uint256 id) public pure  returns (uint256) {
        return id << 64;
    }

}
