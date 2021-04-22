// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "./IERC20.sol";

contract FakeERC20 is IERC20 {
    
    function totalSupply() external view override returns (uint256) {
        return 111;
    }

    function balanceOf(address account) external view override returns (uint256) {
        return 222;
    }

    function transfer(address recipient, uint256 amount) external override returns (bool) {
        emit Transfer(msg.sender, recipient, amount);
    }

    function allowance(address owner, address spender) external view override returns (uint256) {
        return 333;
    }

    function approve(address spender, uint256 amount) external override returns (bool) {
        emit Approval(msg.sender, spender, amount);
    }

    function transferFrom(address sender, address recipient, uint256 amount) external override returns (bool) {

    }

}
