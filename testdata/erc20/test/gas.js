const MyToken = artifacts.require("MyToken");

const _1e18 = 10n ** 18n;

contract("MyToken", async (accounts) => {

    it('gas', async () => {
        const mytk = await MyToken.new(100000000n * _1e18);

        let result = await mytk.transfer(accounts[1], 200n * _1e18);
        console.log("gas cost of transfer()     :", result.receipt.gasUsed);
        assert.equal(await mytk.balanceOf(accounts[1]), 200n * _1e18);

        result = await mytk.approve(accounts[2], 500n * _1e18);
        console.log("gas cost of approve()      :", result.receipt.gasUsed);
        assert.equal(await mytk.allowance(accounts[0], accounts[2]), 500n * _1e18);

        result = await mytk.transferFrom(accounts[0], accounts[3], 300n * _1e18, { from: accounts[2] });
        console.log("gas cost of transferFrom() :", result.receipt.gasUsed);
        assert.equal(await mytk.balanceOf(accounts[3]), 300n * _1e18);
    });

});
