const Contract1 = artifacts.require("Contract1");
const Contract2 = artifacts.require("Contract2");
const Contract3 = artifacts.require("Contract3");

contract("Contracts", async (accounts) => {

    it('call', async () => {
        const contract3 = await Contract3.new();
        console.log(contract3.address);

        const contract2 = await Contract2.new(contract3.address);
        console.log(contract2.address);

        const contract1 = await Contract1.new(contract2.address, contract3.address);
        console.log(contract1.address);

        // await contract1.call2(123);
        // await contract1.call3(123);
        await contract3.callMe(123);
    });

});
