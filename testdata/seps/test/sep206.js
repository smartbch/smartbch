const ISEP20 = artifacts.require("ISEP20");
const sep206 = new ISEP20("0x0000000000000000000000000000000000002711");

contract("SEP206", async (accounts) => {

    it('basic info', async () => {
        assert.equal(await sep206.name(), "BCH");
        assert.equal(await sep206.symbol(), "BCH");
        assert.equal(await sep206.decimals(), 18);
        assert.equal((await sep206.totalSupply()).toString(), "21000000000000000000000000");
    });

    it('balance', async () => {
        assert.equal(await sep206.balanceOf(accounts[0]),
            await web3.eth.getBalance(accounts[0]));
        assert.equal(await sep206.balanceOf("0xADD0000000000000000000000000000000000001"), 0);
    });

    it('transfer: ok', async () => {
        await testTransfer(accounts[1], accounts[2], 0);
        await testTransfer(accounts[1], accounts[2], 10000);

        const newAddr = "0xADD0000000000000000000000000000000000002";
        await testTransfer(accounts[3], newAddr, 0);
        await testTransfer(accounts[3], newAddr, 10000);
    });

    it('transfer: amt exceed balance', async () => {
        const bal = await web3.eth.getBalance(accounts[0]);
        const amt = BigInt(bal) + 1n;
        try {
            await sep206.transfer(accounts[1], amt, { from: accounts[0], gasPrice: 0 });
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('approve&allowance: ok', async () => {
        const owner = accounts[0];
        const spender = accounts[1];
        assert.equal(await sep206.allowance(owner, spender), 0);

        await sep206.approve(spender, 1234, { from: owner });
        assert.equal(await sep206.allowance(owner, spender), 1234);

        await sep206.increaseAllowance(spender, 123, { from: owner });
        assert.equal(await sep206.allowance(owner, spender), 1357);

        await sep206.decreaseAllowance(spender, 345, { from: owner });
        assert.equal(await sep206.allowance(owner, spender), 1012);
    });

    it('approve&allowance: to non-existed addr', async () => {
        const owner = accounts[0];
        const spender = "0xADD0000000000000000000000000000000000003";
        assert.equal(await sep206.allowance(owner, spender), 0);

        await sep206.approve(spender, 1234, { from: owner });
        assert.equal(await sep206.allowance(owner, spender), 1234);

        await sep206.increaseAllowance(spender, 123, { from: owner });
        assert.equal(await sep206.allowance(owner, spender), 1357);

        await sep206.decreaseAllowance(spender, 345, { from: owner });
        assert.equal(await sep206.allowance(owner, spender), 1012);
    });

    it('increase/decreaseAllowance: overflow', async () => {
        const owner = accounts[1];
        const spender = accounts[2];
        assert.equal(await sep206.allowance(owner, spender), 0);

        await sep206.approve(spender, "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF3", { from: owner });
        assert.equal((await sep206.allowance(owner, spender)).toString(), 
            0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF3n.toString());

        await sep206.increaseAllowance(spender, 100, { from: owner });
        assert.equal((await sep206.allowance(owner, spender)).toString(), 
            0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFn.toString());

        await sep206.decreaseAllowance(spender, "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF1", { from: owner });
        assert.equal(await sep206.allowance(owner, spender), 14);

        await sep206.decreaseAllowance(spender, 200, { from: owner });
        assert.equal(await sep206.allowance(owner, spender), 0);
    });

});

async function testTransfer(a, b, amt) {
    const balA = await web3.eth.getBalance(a);
    const balB = await web3.eth.getBalance(b);
    await sep206.transfer(b, amt, { from: a, gasPrice: 0 });
    assert.equal(await web3.eth.getBalance(a), (BigInt(balA) - BigInt(amt)).toString());
    assert.equal(await web3.eth.getBalance(b), (BigInt(balB) + BigInt(amt)).toString());
}
