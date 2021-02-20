const Counter = artifacts.require("Counter");

module.exports = async function (deployer, network, accounts) {
    await deployer.deploy(Counter);
    console.log('Counter:', Counter.address);

    let counter = await Counter.deployed();
    console.log(counter);

    let x = await counter.counter.call();
    console.log(x);
};
