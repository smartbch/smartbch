const Storage = artifacts.require("Storage");

contract("Storage", async (accounts) => {

    it('getStorageAt', async () => {
        const contract0 = await Storage.new({ from: accounts[0] });
        const contract1 = await Storage.new({ from: accounts[1] });

        await testStorage(accounts[1], contract0, 0, 0x0123);
        await testStorage(accounts[2], contract0, 1, 0x1234);
        await testStorage(accounts[3], contract0, 2, 0x2345);
        await testStorage(accounts[4], contract1, 3, 0x3456);
        await testStorage(accounts[5], contract1, 4, 0x4567);
        await testStorage(accounts[6], contract1, 5, 0x5678);
    });

});

async function testStorage(caller, contract, idx, val) {
    await contract.set(idx, val, { from: caller });
    assert.equal(await contract.get(idx), val);
    assert.equal(await web3.eth.getStorageAt(contract.address, idx), val);
}
