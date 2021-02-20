contract("Transfer", async (accounts) => {

    it('all accounts', async () => {
        assert.equal(accounts.length, 10);
        assert.equal(await web3.eth.getBalance(accounts[0]), 1000000000000000000);
        assert.equal(await web3.eth.getBalance(accounts[1]), 1000000000000000000);
    });

    it('transfer eth', async () => {
        await web3.eth.sendTransaction({from: accounts[0], to: accounts[1], value: 10000});
        assert.equal(await web3.eth.getBalance(accounts[1]), 1000000000000010000);
        assert.equal(await web3.eth.getBalance(accounts[0]),  999999999999990000);
    });

});
