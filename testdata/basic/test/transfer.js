contract("Transfer", async (accounts) => {

    it('transfer eth', async () => {
        const acc0 = accounts[0];
        const acc1 = accounts[1];
        const bal0 = await web3.eth.getBalance(acc0);
        const bal1 = await web3.eth.getBalance(acc1);
        await web3.eth.sendTransaction({from: acc0, to: acc1, value: 10000});
        assert.equal(await web3.eth.getBalance(acc0), BigInt(bal0) - BigInt(10000));
        assert.equal(await web3.eth.getBalance(acc1), BigInt(bal1) + BigInt(10000));
    });

});
