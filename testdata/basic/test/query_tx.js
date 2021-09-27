const BlockHash = artifacts.require("BlockHash");

contract("RPC/QueryTx", async (accounts) => {

    it('NormalTxToAddr', async () => {
        const from = accounts[0];
        const to = accounts[1];
        const result = await web3.eth.sendTransaction(
            {from: from, to: to, value: 10000, gasPrice: 10000000000});

        // console.log(result);
        const txHash = result.transactionHash;

        let tx = await web3.eth.getTransaction(txHash);
        assert.equal(tx.from, from)
        assert.equal(tx.to, to);

        tx = await web3.eth.getTransactionFromBlock(tx.blockNumber, 0);
        assert.equal(tx.from, from)
        assert.equal(tx.to, to);

        tx = await web3.eth.getTransactionFromBlock(tx.blockHash, 0);
        assert.equal(tx.from, from)
        assert.equal(tx.to, to);

        const receipt = await web3.eth.getTransactionReceipt(txHash);
        assert.equal(tx.from, from)
        assert.equal(receipt.to, to.toLowerCase());

        let block = await web3.eth.getBlock(tx.blockNumber, true);
        assert.equal(block.transactions[0].from, from);
        assert.equal(block.transactions[0].to, to);

        block = await web3.eth.getBlock(tx.blockHash, true);
        assert.equal(block.transactions[0].from, from);
        assert.equal(block.transactions[0].to, to); 
    });

    it('ContractCreationTxToAddr', async () => {
        const contract = await BlockHash.new();
        const txHash = contract.transactionHash;

        let tx = await web3.eth.getTransaction(txHash);
        assert.equal(tx.from, accounts[0])
        assert.equal(tx.to, null);

        tx = await web3.eth.getTransactionFromBlock(tx.blockNumber, 0);
        assert.equal(tx.from, accounts[0])
        assert.equal(tx.to, null);

        tx = await web3.eth.getTransactionFromBlock(tx.blockHash, 0);
        assert.equal(tx.from, accounts[0])
        assert.equal(tx.to, null);

        const receipt = await web3.eth.getTransactionReceipt(txHash);
        assert.equal(tx.from, accounts[0])
        assert.equal(receipt.to, null);

        let block = await web3.eth.getBlock(tx.blockNumber, true);
        assert.equal(block.transactions[0].from, accounts[0]);
        assert.equal(block.transactions[0].to, null);

        block = await web3.eth.getBlock(tx.blockHash, true);
        assert.equal(block.transactions[0].from, accounts[0]);
        assert.equal(block.transactions[0].to, null); 
    });

});
