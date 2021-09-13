const Faucet = artifacts.require("Faucet");

const gasPrice = 1e10;

function getGasFee(result) {
    return result.receipt.gasUsed * gasPrice;
}

contract("Faucet", async (accounts) => {

    it('faucet', async () => {
        const user = accounts[0];

        const faucet = await Faucet.new({ from: user });
        assert.equal(await web3.eth.getBalance(faucet.address), 0);

        // user -> faucet
        const userBal0 = await web3.eth.getBalance(user);
        const result1 = await faucet.send(1e8, { from: user });
        const gasFee1 = getGasFee(result1);
        const userBal1 = await web3.eth.getBalance(user);
        // assert.equal(userBal1, userBal0 - 1e8 - gasFee1);
        assert.equal(await web3.eth.getBalance(faucet.address), 1e8);

        // user <- faucet
        const result2 = await faucet.withdraw(1e5, { from: user });
        const gasFee2 = getGasFee(result2);
        const userBal2 = await web3.eth.getBalance(user);
        // assert.equal(userBal2, userBal1 + 1e5 - gasFee2);
        assert.equal(await web3.eth.getBalance(faucet.address), 1e8 - 1e5 );
    });

});
