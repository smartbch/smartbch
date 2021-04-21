const StakingTest = artifacts.require("StakingTest");

contract("StakingTest", async (accounts) => {

    let testContract;

    before(async () => {
        testContract = await StakingTest.new();
    });

    it('call staking from contract: createValidator', async () => {
        try {
            await testContract.createValidator(accounts[0], "intro", "pubkey");
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('call staking from contract: editValidator', async () => {
        try {
            await testContract.editValidator(accounts[1], "intro1");
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('call staking from contract: retire', async () => {
        try {
            await testContract.retire();
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('call staking from contract: increaseMinGasPrice', async () => {
        try {
            await testContract.increaseMinGasPrice();
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('call staking from contract: decreaseMinGasPrice', async () => {
        try {
            await testContract.decreaseMinGasPrice();
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

});
