const gasPrice = 10000000000;

function getGasFee(result) {
    return result.gasUsed * gasPrice;
}

contract("Transfer", async (accounts) => {

    it('transfer eth', async () => {
        const acc0 = accounts[0];
        const acc1 = accounts[1];
        const bal0 = await web3.eth.getBalance(acc0);
        const bal1 = await web3.eth.getBalance(acc1);
        const result = await web3.eth.sendTransaction({from: acc0, to: acc1, value: 10000, gasPrice: gasPrice});
        const gasFee = getGasFee(result);
        assert.equal(await web3.eth.getBalance(acc0), BigInt(bal0) - BigInt(10000) - BigInt(gasFee));
        assert.equal(await web3.eth.getBalance(acc1), BigInt(bal1) + BigInt(10000));
    });

    it('transfer eth to new account', async () => {
        const acc0 = accounts[0];
        const acc1 = web3.utils.randomHex(20);
        const bal0 = await web3.eth.getBalance(acc0);
        const result = await web3.eth.sendTransaction({from: acc0, to: acc1, value: 10000, gasPrice: gasPrice});
        const gasFee = getGasFee(result);
        assert.equal(await web3.eth.getBalance(acc0), BigInt(bal0) - BigInt(10000) - BigInt(gasFee));
        assert.equal(await web3.eth.getBalance(acc1), 10000);
    });

});
