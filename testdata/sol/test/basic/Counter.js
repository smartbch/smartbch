const Counter = artifacts.require("Counter");

contract("Counter", async (accounts) => {

    let counter;

    before(async () => {
        counter = await Counter.new();
    });

    it('call getter', async () => {
        // let counter = await Counter.deployed();
        assert.equal((await counter.counter.call()), 0, "counter != 0");
    });

    it('call update', async () => {
        // let counter = await Counter.deployed();
        await counter.update.call(0x12345);
        assert.equal((await counter.counter.call()), 0, "counter != 1");
    });

    it('get caller', async () => {
         // let counter = await Counter.deployed();
         console.log("caller:", await counter.getCaller.call());
    });

    it('get caller web3', async () => {
        // let counter = await Counter.deployed();
        web3.eth.call({
            from: "0x1234567890123456789012345678901234567890",
            to: counter.address, // contract address
            data: "0xab470f05"
        });
        console.log("caller:", await counter.getCaller.call());
    });

});
