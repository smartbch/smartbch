const SEP101Proxy = artifacts.require("SEP101ProxyForGasTest2");

contract("SEP101ProxyForGasTest2", async (accounts) => {

    it('gas', async () => {
        const proxy = await SEP101Proxy.new();

        for (let i = 1; i < 256; i += 5) {
            let key = "ab".repeat(i);
            let val = "cd".repeat(i * 96);
            let setResult = await invokeSet(accounts[0], proxy.address, "0x" + key, "0x" + val);
            let getResult = await invokeGet(accounts[0], proxy.address, "0x" + key);

            let keyLen = padStart(key.length / 2, 3);
            let valLen = padStart(val.length / 2, 5);
            let setGas = padStart(setResult.gasUsed, 7);
            let getGas = padStart(getResult.gasUsed, 7);
            console.log(`key len: ${keyLen}, val len: ${valLen}, set gas: ${setGas}, get gas: ${getGas}`);
        }
    });

});

function padStart(n, w) {
    return n.toString().padStart(w, ' ');
}

async function invokeSet(invoker, proxy, key, val) {
    const data = web3.eth.abi.encodeFunctionCall({
      "inputs": [
        {
          "internalType": "bytes",
          "name": "key",
          "type": "bytes"
        },
        {
          "internalType": "bytes",
          "name": "value",
          "type": "bytes"
        }
      ],
      "name": "set",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    [key, val]);

    return await web3.eth.sendTransaction({from: invoker, to: proxy, data: data});
}

async function invokeGet(invoker, proxy, key) {
    const data = web3.eth.abi.encodeFunctionCall({
      "inputs": [
        {
          "internalType": "bytes",
          "name": "key",
          "type": "bytes"
        }
      ],
      "name": "get",
      "outputs": [
        {
          "internalType": "bytes",
          "name": "",
          "type": "bytes"
        }
      ],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    [key]);

    return await web3.eth.sendTransaction({from: invoker, to: proxy, data: data});
}
