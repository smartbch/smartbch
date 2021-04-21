// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

interface IStaking {

    function createValidator(address rewardTo, bytes32 introduction, bytes32 pubkey) external;
    function editValidator(address rewardTo, bytes32 introduction) external;
    function retire() external;
    function increaseMinGasPrice() external;
    function decreaseMinGasPrice() external;

}
