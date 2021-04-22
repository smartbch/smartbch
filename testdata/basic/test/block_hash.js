const BlockHash = artifacts.require("BlockHash");

contract("BlockHash", async (accounts) => {

    it('getBlockHash', async () => {
        const contract = await BlockHash.new();

        for (let i = 0; i < 5; i++) {
            await web3.eth.sendTransaction({from: accounts[0], to: accounts[1], value: 10000});
        }

        for (let i = 0; i < 5; i++) {
            await web3.eth.sendTransaction({from: accounts[0], to: accounts[1], value: 10000});

            let blockNum = await web3.eth.getBlockNumber();
            for (let j = 0; j < 5; j++, blockNum--) {
                let block = await web3.eth.getBlock(blockNum);
                let blockHash = await contract.getBlockHash(blockNum);
                console.log("block number: ", blockNum);
                console.log(block.hash);
                console.log(blockHash);
                assert.equal(blockHash, block.hash);
            }
        }
    });

});
