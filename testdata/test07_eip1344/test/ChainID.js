const ChainID = artifacts.require("ChainID");

contract("ChainID", async (accounts) => {

    it('getChainID', async () => {
        const contract = await ChainID.new({ from: accounts[0] });
        assert.equal(1, await contract.getChainID());
    });

});
