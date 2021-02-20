const Counter = artifacts.require("Counter");

contract("Call", async (accounts) => {

    it('call getter', async () => {
        let counter = await Counter.deployed();
        assert.equal((await counter.counter.call()), 0, "counter != 0");
    });

});
