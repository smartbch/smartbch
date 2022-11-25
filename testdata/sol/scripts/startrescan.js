const CC = artifacts.require("CCSystem");

const ccAddress = '0x0000000000000000000000000000000000002714';

async function main() {
    const args = process.argv.slice(5);
    console.log(args);
    const height = args[0];

    const accounts = await web3.eth.getAccounts();
    const balance = await web3.eth.getBalance(accounts[0]);
    console.log('acc0:', accounts[0], 'balance:', web3.utils.fromWei(balance, 'ether'));

    let cc = new web3.eth.Contract([
        {
            "inputs": [
                {
                    "internalType": "uint256",
                    "name": "mainFinalizedBlockHeight",
                    "type": "uint256"
                }
            ],
            "name": "startRescan",
            "outputs": [],
            "stateMutability": "nonpayable",
            "type": "function"
        }
    ], ccAddress)
    const tx = await cc.methods.startRescan(height).send({
        from: accounts[0],
        gasPrice: 100000000000,
        gas: 400000
    });
    console.log(tx);
}

module.exports = async function(callback) {
    main()
        .then(callback)
        .catch(error => {
            console.error(error);
            process.exit(1);
        });
}