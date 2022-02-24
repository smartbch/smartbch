// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract TestSEP109 {

	function verify(uint alpha, bytes calldata pk, bytes calldata pi, uint beta) external view returns (bool) {
		require(pk.length == 33, 'pk.length != 33');
		(bool ok, bytes memory retData) = address(0x2713).staticcall(abi.encodePacked(alpha, pk, pi));
		return ok && abi.decode(retData, (uint)) == beta;
	}

}
