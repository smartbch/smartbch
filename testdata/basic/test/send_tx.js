contract("SendTx", async (accounts) => {

    it('bad from addr', async () => {
        try {
            await web3.eth.sendTransaction({
                from : "0x1234567890123456789012345678901234567890",
                to   : "0x09F236e4067f5FcA5872d0c09f92Ce653377aE41", 
                nonce: 1,
                value: 10000
            });
            throw null;
        } catch (error) {
            assert(error, "Expected an error but did not get one");
            // console.log(error.message);
            // assert(error.message.includes("unknown account"));
        }
    });

    it('bad from addr, no nonce', async () => {
        try {
            await web3.eth.sendTransaction({
                from : "0x1234567890123456789012345678901234567890",
                to   : "0x09F236e4067f5FcA5872d0c09f92Ce653377aE41", 
                value: 10000
            });
            throw null;
        } catch (error) {
            assert(error, "Expected an error but did not get one");
            // console.log(error.message);
            // assert(error.message.includes("unknown account"));
        }
    });

});
