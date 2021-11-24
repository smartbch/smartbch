const Errors = artifacts.require("Errors");

contract("Errors", async (accounts) => {

    let contract;

    before(async () => {
        contract = await Errors.new({ from: accounts[0] });
    });

    it('revert, returns gas', async () => {
        // console.log(await web3.eth.getBalance(accounts[0]));
        try  {
            await contract.setN_revert(100, { 
                gasPrice: (10n**10n).toString(),
                gas: (10n**7n).toString(),
            });
        } catch(error) {
            assert(error, "Expected an error but did not get one");
            console.log(error.receipt);
        }
        // console.log(await web3.eth.getBalance(accounts[0]));
    });

    it('revert', async () => {
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
        await contract.setN_revert.estimateGas(1);
    });

});
