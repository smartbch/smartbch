// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

// openzeppelin/contracts/token/ERC721/ERC721.sol
contract ERC721 {
    string private _name;
    string private _symbol;
    mapping(uint256 => address) private _owners;
    mapping(address => uint256) private _balances;
    mapping(uint256 => address) private _tokenApprovals;
    mapping(address => mapping(address => bool)) private _operatorApprovals;

    // constructor(string memory name_, string memory symbol_) {
    //     _name = name_;
    //     _symbol = symbol_;
    // }
}

struct Vault {
	uint64 initCollateralRate;
	uint64 minCollateralRate;
	uint64 matureTime;
	uint64 lastVoteTime;
	uint validatorToVote;
	uint96 hedgeValue;
	address oracle;
	uint64 closeoutPenalty;
	uint96 amount; // at most 85 bits (21 * 1e6 * 1e18)
}

contract XHedgeStorage is ERC721 {
	// mapping (uint => Vault) private snToVault;
	uint[128] internal nextSN;
	mapping (uint => uint) public valToVotes; // slot: 134
	uint[] public validators;                 // slot: 135

	// constructor() ERC721("XHedge", "XH") {}

	function addVal(uint val, uint votes) public {
		validators.push(val);
		valToVotes[val] = votes;
	}
}
