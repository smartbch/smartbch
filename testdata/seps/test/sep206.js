const IERC20 = artifacts.require("IERC20");

contract("SEP206", async (accounts) => {

    it('basic info', async () => {
        const sep206 = new IERC20("0x0000000000000000000000000000000000002711");
        assert.equal(await sep206.name(), "BCH");
        assert.equal(await sep206.symbol(), "BCH");
        assert.equal(await sep206.decimals(), 18);
        assert.equal((await sep206.totalSupply()).toString(), "21000000000000000000000000");
    });

});
