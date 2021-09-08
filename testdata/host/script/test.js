const assert = require('assert');
const web3 = require('web3');
const abi = require('./abi')
const BN = require('bignumber.js');

const interval = 5 * 1000
const testHostAddress = '0x35f2E649D46A7EfF8ffF24b8dd285e1bfAEE997C'
const amberNodeUrl = 'http://52.77.241.179:8545'
const provider = new web3(new web3.providers.HttpProvider(amberNodeUrl));
const testContract = new provider.eth.Contract(abi.testHostAbi, testHostAddress);

let preBlockNum;
let preCoinbase;
let preTimestamp;
let initBlock = true

async function run() {
    const blockNumRpc = await provider.eth.getBlockNumber();
    //console.log(blockNumRpc);
    const block = await provider.eth.getBlock(blockNumRpc);
    console.log(block)

    const blockNumber = await testContract.methods.blocknumber().call();
    const blockHash = await testContract.methods.blkhash(blockNumber - 1).call();
    const coinbase = await testContract.methods.blockcoinbase().call();
    const timestamp = await testContract.methods.blocktimestamp().call();
    const difficulty = await testContract.methods.blockdifficulty().call();
    const gasLimit = await testContract.methods.blockgaslimit().call();
    const sig = await testContract.methods.txsignatrue().call();

    assert.equal('0xc56f8ace', sig)
    assert.equal(0, difficulty)

    console.log(`gasLimit: %d`, gasLimit)

    if (!initBlock && preBlockNum === blockNumRpc) {
        assert.equal(preCoinbase, block.miner);
        assert.equal(preTimestamp, block.timestamp);
        assert.equal(blockHash, block.hash);
    }
    if (initBlock === true) {
        initBlock = false
    }

    if (blockNumRpc + 1 === Number(blockNumber)) {
        preBlockNum = Number(blockNumber)
        preCoinbase = coinbase
        preTimestamp = timestamp
    }
}

function mainRoutine() {
    setInterval(async function () {
        await run()
    }, interval);
}

mainRoutine()