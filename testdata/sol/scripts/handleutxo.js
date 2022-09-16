const CC = artifacts.require("CCSystem");

const ccAddress = '0x0000000000000000000000000000000000002714';

async function main() {
    const accounts = await web3.eth.getAccounts();
    const balance = await web3.eth.getBalance(accounts[0]);
    console.log('acc0:', accounts[0], 'balance:', web3.utils.fromWei(balance, 'ether'));

    let cc = new web3.eth.Contract([
        {
            "inputs": [],
            "name": "handleUTXOs",
            "outputs": [],
            "stateMutability": "nonpayable",
            "type": "function"
        }
    ], ccAddress)
    const tx = await cc.methods.handleUTXOs().send({
        from: accounts[0],
        gasPrice: 100000000000,
        gas: 400000
    });
    console.log(tx)
}

module.exports = async function(callback) {
    main()
        .then(callback)
        .catch(error => {
            console.error(error);
            process.exit(1);
        });
}