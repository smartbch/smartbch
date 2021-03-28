const Counter = artifacts.require("Counter");

module.exports = async function (deployer, network, accounts) {
    await deployer.deploy(Counter);
    console.log('Counter:', Counter.address);
};
