const Errors = artifacts.require("Errors");

contract("Errors", async (accounts) => {

    it('invalid opcode', async () => {
        const contract = await Errors.new({ from: accounts[0] });
        try {
            await contract.setN_invalidOpcode(100);
            throw null;
        } catch (error) {
            assert(error, "Expected an error but did not get one");
            // assert.equal(error.message, 
            //     "Returned error: VM Exception while processing transaction: invalid opcode");
        }
    })

    it('invalid opcode, estimateGas', async () => {
        const contract = await Errors.new({ from: accounts[0] });
        try {
            await contract.setN_invalidOpcode.estimateGas(100);
            throw null;
        } catch (error) {
            assert(error, "Expected an error but did not get one");
            // assert.equal(error.message, 
            //     "Returned error: VM Exception while processing transaction: invalid opcode");
        }
    })

});
