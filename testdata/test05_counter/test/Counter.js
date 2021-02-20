const Counter = artifacts.require("Counter");

contract("Counter", async (accounts) => {

    it('counter getter', async () => {
        counter = await Counter.deployed();
        assert.equal((await counter.counter.call()), 0, "counter != 0");
    });

});
