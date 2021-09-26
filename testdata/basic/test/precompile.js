const gasPrice = 10000000000;

contract("Precompile", async (accounts) => {

    it('sha256', async () => {
        const nonce = await web3.eth.getTransactionCount(accounts[0]);
        const tx = await web3.eth.sendTransaction({
            from    : accounts[0],
            to      : "0x0000000000000000000000000000000000000002", 
            nonce   : nonce,
            value   : 1,
            data    : "0x1234",
            gasPrice: gasPrice
        });
        // console.log(tx);

        const data = await web3.eth.call({
            from : accounts[0],
            to   : "0x0000000000000000000000000000000000000002", 
            nonce: nonce,
            value: 0,
            data : "0x1234",
        });
        assert.equal(data, "0x3a103a4e5729ad68c02a678ae39accfbc0ae208096437401b7ceab63cca0622f");
    });

});
