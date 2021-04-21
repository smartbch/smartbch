const ISEP20 = artifacts.require("ISEP20");
const sep206 = new ISEP20("0x0000000000000000000000000000000000002711");

// const _1e18 = 10n ** 18n;

contract("SEP206", async (accounts) => {

    it('gas', async () => {
        let result =  await sep206.name.sendTransaction();
        console.log("gas cost of name()             :", result.receipt.gasUsed);

        result =  await sep206.symbol.sendTransaction();
        console.log("gas cost of symbol()           :", result.receipt.gasUsed);

        result =  await sep206.decimals.sendTransaction();
        console.log("gas cost of decimals()         :", result.receipt.gasUsed);

        result =  await sep206.totalSupply.sendTransaction();
        console.log("gas cost of totalSupply()      :", result.receipt.gasUsed);

        result = await sep206.transfer(accounts[1], 200n);
        console.log("gas cost of transfer()         :", result.receipt.gasUsed);
        // assert.equal(await sep206.balanceOf(accounts[1]), 200n);

        result = await sep206.approve(accounts[2], 500n);
        console.log("gas cost of approve()          :", result.receipt.gasUsed);
        assert.equal(await sep206.allowance(accounts[0], accounts[2]), 500n);

        result = await sep206.increaseAllowance(accounts[2], 100n);
        console.log("gas cost of increaseAllowance():", result.receipt.gasUsed);
        assert.equal(await sep206.allowance(accounts[0], accounts[2]), 600n);

        result = await sep206.decreaseAllowance(accounts[2], 200n);
        console.log("gas cost of decreaseAllowance():", result.receipt.gasUsed);
        assert.equal(await sep206.allowance(accounts[0], accounts[2]), 400n);

        result = await sep206.transferFrom(accounts[0], accounts[3], 300n, { from: accounts[2] });
        console.log("gas cost of transferFrom()     :", result.receipt.gasUsed);
        // assert.equal(await sep206.balanceOf(accounts[3]), 300n);
    });

});
