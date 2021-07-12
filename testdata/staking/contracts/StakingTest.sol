// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "./IStaking.sol";

contract StakingTest is IStaking {

    address constant public stakingAddr = address(0x2710);

    function createValidator(address rewardTo, bytes32 introduction, bytes32 pubkey) external override {
        bytes4 _selector = bytes4(keccak256(bytes("createValidator(address,bytes32,bytes32)")));
        (bool ok, bytes memory data) = stakingAddr.call(abi.encodeWithSelector(_selector, rewardTo, introduction, pubkey));
        require(ok, string(data));
    }

    function editValidator(address rewardTo, bytes32 introduction) external override {
        bytes4 _selector = bytes4(keccak256(bytes("editValidator(address,bytes32)")));
        (bool ok, bytes memory data) = stakingAddr.call(abi.encodeWithSelector(_selector, rewardTo, introduction));
        require(ok, string(data));
    }

    function retire() external override {
        bytes4 _selector = bytes4(keccak256(bytes("retire()")));
        (bool ok, bytes memory data) = stakingAddr.call(abi.encodeWithSelector(_selector));
        require(ok, string(data));
    }

    function increaseMinGasPrice() external override {
        bytes4 _selector = bytes4(keccak256(bytes("increaseMinGasPrice()")));
        (bool ok, bytes memory data) = stakingAddr.call(abi.encodeWithSelector(_selector));
        require(ok, string(data));
    }

    function decreaseMinGasPrice() external override {
        bytes4 _selector = bytes4(keccak256(bytes("decreaseMinGasPrice()")));
        (bool ok, bytes memory data) = stakingAddr.call(abi.encodeWithSelector(_selector));
        require(ok, string(data));
    }

    function sumVotingPower(address[] calldata addrList) external override returns (uint summedPower, uint totalPower) {
        bytes4 _selector = bytes4(keccak256(bytes("sumVotingPower(address[])")));
        (bool ok, bytes memory data) = stakingAddr.call(abi.encodeWithSelector(_selector, addrList));
        require(ok, string(data));
        return (0, 0);
    }
}
