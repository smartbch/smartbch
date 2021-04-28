const ChainID = artifacts.require("ChainID");

contract("ChainID", async (accounts) => {

    it('getChainID', async () => {
        const contract = await ChainID.new({ from: accounts[0] });
        assert.equal(await contract.getChainID(), 0x2711);
    });

});
