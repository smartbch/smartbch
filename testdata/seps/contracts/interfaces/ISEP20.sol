// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "./IERC20.sol";

// https://github.com/smartbch/docs/blob/main/smartbch-evolution-proposals-seps/sep-20.md
interface ISEP20 is IERC20 {

    function owner() external view returns (address);
    function increaseAllowance(address _spender, uint256 _delta) external returns (bool success);
    function decreaseAllowance(address _spender, uint256 _delta) external returns (bool success);

}