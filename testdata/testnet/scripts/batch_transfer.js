const toAddrs = [
"0x47De0Bbe028dADd438B4426B39fA0bC73FCFdBcD",
"0x652E4a0b289ea5C7FEEF349C51D8C217cb8D5347",
// ...
];

async function main() {
    const accounts = await web3.eth.getAccounts();

    for (let i = 0; i < toAddrs.length; i++) {
        const addr = toAddrs[i];
        console.log("send 100 BCH to ", addr);
        await web3.eth.sendTransaction({
            from    : accounts[0],
            to      : addr, 
            value   : "0x56bc75e2d63100000", 
            gasPrice: 0
        });
    }
}

module.exports = async function(callback) {
    main()
        .then(callback)
        .catch(error => {
            console.error(error);
            process.exit(1);
        });
}
