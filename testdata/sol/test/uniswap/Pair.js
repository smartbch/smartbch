// https://ethereum.stackexchange.com/questions/87523/deploy-pre-compiled-bytecode-using-truffle-migrations-deployer-api
const factoryJson = require('@uniswap/v2-core/build/UniswapV2Factory.json');
const pairJson = require('@uniswap/v2-core/build/UniswapV2Pair.json');
const erc20Json = require('@uniswap/v2-core/build/ERC20.json');
const _contract = require('@truffle/contract');
const UniswapV2Factory = _contract(factoryJson);
const UniswapV2Pair = _contract(pairJson);
const ERC20 = _contract(erc20Json);
UniswapV2Factory.setProvider(web3._provider);
UniswapV2Pair.setProvider(web3._provider);
ERC20.setProvider(web3._provider);

const gasPrice = 10000000000;

contract("UniswapV2Pair", async (accounts) => {

    // it('deploy ERC20', async () => {
    //     const tokenA = await ERC20.new(100000000, { from: accounts[0] });
    // });

    it('deploy UniswapV2', async () => {
        const tokenA = await ERC20.new(100000000, { from: accounts[0], gasPrice: gasPrice });
        const tokenB = await ERC20.new(100000000, { from: accounts[0], gasPrice: gasPrice });

        const uniFactory = await UniswapV2Factory.new(accounts[0], { from: accounts[0], gasPrice: gasPrice });
        // console.log(await uniFactory.feeToSetter(), accounts[0]);

        await uniFactory.createPair(tokenA.address, tokenB.address, { from: accounts[0], gasPrice: gasPrice });
        const pairAddr = await uniFactory.getPair(tokenA.address, tokenB.address);
        const uniPair = await UniswapV2Pair.at(pairAddr);

        console.log("pair address:", pairAddr);
        console.log("tokenA address:", tokenA.address);
        console.log("tokenB address:", tokenB.address);
        console.log("token0 address:", await uniPair.token0());
        console.log("token1 address:", await uniPair.token1());
        // assert.equal(await uniPair.token0(), tokenA.address);
        // assert.equal(await uniPair.token1(), tokenB.address);
    });

});
