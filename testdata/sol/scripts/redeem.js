const CC = artifacts.require("CCSystem");

const ccAddress = '0x0000000000000000000000000000000000002714';

async function main() {
    const accounts = await web3.eth.getAccounts();
    const balance = await web3.eth.getBalance(accounts[0]);
    console.log('acc0:', accounts[0], 'balance:', web3.utils.fromWei(balance, 'ether'));

    const args = process.argv.slice(5);
    console.log(args);

    const txid = args[0];
    const index = args[1];
    const targetAddress = args[2];
    const amount = args[3];

    let cc = new web3.eth.Contract([
        {
            "inputs": [
                {
                    "internalType": "uint256",
                    "name": "txid",
                    "type": "uint256"
                },
                {
                    "internalType": "uint256",
                    "name": "index",
                    "type": "uint256"
                },
                {
                    "internalType": "address",
                    "name": "targetAddress",
                    "type": "address"
                }
            ],
            "name": "redeem",
            "outputs": [],
            "stateMutability": "nonpayable",
            "type": "function"
        }
    ], '0x0000000000000000000000000000000000002714')
    const tx = await cc.methods.redeem(
        txid,
        index,
        targetAddress
    ).send({
        from: accounts[0],
        value: amount,
        gasPrice: 100000000000,
        gas: 400000
    });
    console.log(tx);
}

module.exports = async function (callback) {
    main()
        .then(callback)
        .catch(error => {
            console.error(error);
            process.exit(1);
        });
}