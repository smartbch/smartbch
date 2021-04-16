contract("Transfer", async (accounts) => {

    it('transfer eth', async () => {
        console.log(accounts);
        console.log(await web3.eth.getAccounts());
        console.log(await web3.eth.getBalance(accounts[0]));

        // await web3.eth.sendTransaction({
        //     from : accounts[0], 
        //     to   : accounts[1], 
        //     value: 10000,
        // });
    });

});
