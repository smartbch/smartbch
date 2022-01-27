// SPDX-License-Identifier: MIT
pragma solidity >=0.7.0;

contract Errors {

    uint256 public n;

    // test revert
    function setN_revert(uint256 _n) public {
        require(_n < 10, "n must be less than 10");
        n = _n;
    }

    // test invalid opcode
    function setN_invalidOpcode(uint256 _n) public {
        assert(_n < 10);
        n = _n;
    }

}
