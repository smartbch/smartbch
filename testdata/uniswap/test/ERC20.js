const _contract = require('@truffle/contract');
const erc20Json = require('@uniswap/v2-periphery/build/ERC20.json');
const ERC20 = _contract(erc20Json);
ERC20.setProvider(web3._provider);

const gasPrice = 10000000000;

contract("UniswapV2Pair", async (accounts) => {

    it('deploy ERC20', async () => {
        const tokenA = await ERC20.new('10000000000000000000000', { from: accounts[0], gasPrice: gasPrice });
        const tokenB = await ERC20.new('10000000000000000000000', { from: accounts[0], gasPrice: gasPrice });
    });

});
