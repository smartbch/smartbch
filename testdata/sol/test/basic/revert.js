const Errors = artifacts.require("Errors");

contract("Errors", async (accounts) => {

    it('revert', async () => {
        const contract = await Errors.new({ from: accounts[0] });
        try {
            await contract.setN_revert(100);
            throw null;
        } catch (error) {
            assert(error, "Expected an error but did not get one");
            // console.log(web3.utils.hexToAscii('0x' + error.receipt.outData));
            // assert.equal(error.message, 
            //     "Returned error: VM Exception while processing transaction: revert n must be less than 10 -- Reason given: n must be less than 10.");
        }
    });

    it('revert, estimateGas', async () => {
        const contract = await Errors.new({ from: accounts[0] });
        try {
            await contract.setN_revert.estimateGas(100);
            throw null;
        } catch (error) {
            assert(error, "Expected an error but did not get one");
            // assert.equal(error.message, 
            //     "Returned error: VM Exception while processing transaction: revert n must be less than 10");
        }
    });

    it('estimateGas', async () => {
        const contract = await Errors.new({ from: accounts[0] });
        await contract.setN_revert.estimateGas(1);
    });

});
