contract("SendRawTx", async (accounts) => {

    it('tx', async () => {
        try {
            const txData = '0xf88406808398967f94ad65e98865806baa2fe238aac5aee210dadea8a080a46a627842000000000000000000000000ab5d62788e207646fa60eb3eebdc4358c7f5686c42a073457090f7bb8c528efa517217f60b707ea1d9b3a568fd9edf2a2f823e764e5fa07184deb49ab9bf74ad31199dca9e8a442fcc5da7a5562b066d3068aec76a06a6';
            await web3.eth.sendSignedTransaction(txData);
            throw null;
        } catch (error) {
            assert(error, "Expected an error but did not get one");
            assert(error.message.includes("invalid sender"));
        }
    });

});
