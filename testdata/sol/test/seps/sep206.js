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

        const newAddr = web3.utils.randomHex(20);;
        assert.equal(await sep206.balanceOf(newAddr), 0);
    });

    it('transfer: ok', async () => {
        await testTransfer(accounts[1], accounts[2], 0);
        await testTransfer(accounts[1], accounts[2], 10000);

        const newAddr = web3.utils.randomHex(20);
        await testTransfer(accounts[3], newAddr, 0);
        await testTransfer(accounts[3], newAddr, 10000);
    });

    it('transfer: insufficient balance', async () => {
        const bal = await web3.eth.getBalance(accounts[0]);
        const amt = BigInt(bal) + 1n;
        try {
            await sep206.transfer(accounts[1], amt, { from: accounts[0] });
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('approve&allowance: ok', async () => {
        const owner = accounts[0];
        const spender = web3.utils.randomHex(20);
        await web3.eth.sendTransaction({from: owner, to: spender, value: 1, gasPrice: 10000000000});
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
        const spender = web3.utils.randomHex(20);
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

    it('transferFrom: ok', async () => {
        await testTransferFrom(accounts[4], accounts[5], accounts[6], 0);
        await testTransferFrom(accounts[4], accounts[5], accounts[6], 12345);

        const newAddr = web3.utils.randomHex(20);
        await testTransferFrom(accounts[4], newAddr, accounts[6], 0);
        await testTransferFrom(accounts[4], newAddr, accounts[6], 12345);
    });

    it('transferFrom: insufficient balance', async () => {
        const from = accounts[0];
        const to = accounts[1]
        const spender = accounts[2];

        const bal = await web3.eth.getBalance(from);
        const amt = BigInt(bal) + 1n;
        await sep206.approve(spender, amt, { from: from });
        try {
            await sep206.transferFrom(from, to, amt, { from: spender });
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('transferFrom: insufficient allowance', async () => {
        const from = accounts[0];
        const to = accounts[1]
        const spender = accounts[2];

        const amt = '1000000000000000000';
        await sep206.approve(spender, '100000000000000000', { from: from });
        try {
            await sep206.transferFrom(from, to, amt, { from: spender });
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('transferFrom: non-existed from addr', async () => {
        const from = web3.utils.randomHex(20);
        const to = accounts[1]
        const spender = accounts[2];

        try {
            await sep206.transferFrom(from, to, 1, { from: spender });
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('events: Transfer', async () =>  {
        const from = accounts[0];
        const to = accounts[1]

        let result = await sep206.transfer(to, 1234, { from: from });
        const transferLog = result.logs.find(element => element.event.match('Transfer'));
        // console.log(transferLog);
        assert.equal(transferLog.args._from, from);
        assert.equal(transferLog.args._to, to);
        assert.equal(transferLog.args._value, 1234);
    });

    it('events: Approval', async () =>  {
        const owner = accounts[0];
        const spender = accounts[2];

        let result = await sep206.approve(spender, 1234, { from: owner });
        const approvalLog = result.logs.find(element => element.event.match('Approval'));
        // console.log(approvalLog);
        assert.equal(approvalLog.args._owner, owner);
        assert.equal(approvalLog.args._spender, spender);
        assert.equal(approvalLog.args._value, 1234);
    });
});

async function testTransfer(from, to, amt) {
    const balFrom = await web3.eth.getBalance(from);
    const balTo = await web3.eth.getBalance(to);
    await sep206.transfer(to, amt, { from: from });
    // assert.equal(await web3.eth.getBalance(from), (BigInt(balFrom) - BigInt(amt)).toString());
    assert.equal(await web3.eth.getBalance(to), (BigInt(balTo) + BigInt(amt)).toString());
}

async function testTransferFrom(from, to, spender, amt) {
    const balFrom = await web3.eth.getBalance(from);
    const balTo = await web3.eth.getBalance(to);

    await sep206.approve(spender, amt + 100, { from: from });
    assert.equal(await sep206.allowance(from, spender), amt + 100);

    await sep206.transferFrom(from, to, amt, { from: spender });
    // assert.equal(await web3.eth.getBalance(from), (BigInt(balFrom) - BigInt(amt)).toString());
    assert.equal(await web3.eth.getBalance(to), (BigInt(balTo) + BigInt(amt)).toString());
    assert.equal(await sep206.allowance(from, spender), 100);
}
