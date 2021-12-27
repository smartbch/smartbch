// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract TestSEP109 {

	function verify(uint alpha, bytes calldata pk, bytes calldata pi, bytes calldata beta) external returns (bool) {
		require(pk.length == 33, 'pk.length != 33');
		(bool ok, bytes memory retData) = address(0x2713).call(abi.encodePacked(alpha, pk, pi));
		return ok && keccak256(retData) == keccak256(beta);
	}

}
