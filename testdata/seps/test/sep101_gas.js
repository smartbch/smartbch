const ISEP101 = artifacts.require("ISEP101");
const SEP101Proxy = artifacts.require("SEP101Proxy");

contract("SEP101", async (accounts) => {

    it('gas', async () => {
        const proxy = await SEP101Proxy.new();

        for (let i = 1; i < 256; i += 5) {
            let key = "ab".repeat(i);
            let val = "cd".repeat(i * 96);
            let setResult = await proxy.set("0x" + key, "0x" + val);
            let getResult = await proxy.get.sendTransaction("0x" + key);

            let keyLen = padStart(key.length / 2, 3);
            let valLen = padStart(val.length / 2, 5);
            let setGas = padStart(setResult.receipt.gasUsed, 7);
            let getGas = padStart(getResult.receipt.gasUsed, 7);
            console.log(`key len: ${keyLen}, val len: ${valLen}, set gas: ${setGas}, get gas: ${getGas}`);
        }
    });

});

function padStart(n, w) {
    return n.toString().padStart(w, ' ');
}
