const Faucet = artifacts.require("Faucet");

contract("Faucet", async (accounts) => {

    it('faucet', async () => {
        const user = accounts[0];
        const userBal = await web3.eth.getBalance(user);

        const faucet = await Faucet.new({ from: user, gasPrice: 0 });
        assert.equal(await web3.eth.getBalance(faucet.address), 0);

        // user -> faucet
        await web3.eth.sendTransaction({from: user, to: faucet.address, value: 12345, gasPrice: 0});
        assert.equal(await web3.eth.getBalance(user), userBal - 12345);
        assert.equal(await web3.eth.getBalance(faucet.address), 12345);

        // user <- faucet
        await faucet.withdraw(123, { from: user, gasPrice: 0 });
        assert.equal(await web3.eth.getBalance(user), userBal - 12345 + 123);
        assert.equal(await web3.eth.getBalance(faucet.address), 12345 - 123);
    });

});
