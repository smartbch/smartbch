const SEP101Proxy = artifacts.require("SEP101Proxy");

const shortKey = "0x1234";
const shortVal = "0x5678";
const longKey = "0x0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789";
const longVal = "0x9876543210987654321098765432109876543210987654321098765432109876543210987654321098765432109876543210";

contract("SEP101", async (accounts) => {

    let sep101Proxy;

    before(async () => {
        sep101Proxy = await SEP101Proxy.new()
    });

    it('get/set delegate_call', async () => {
        await sep101Proxy.set(shortKey, shortVal);
        await sep101Proxy.get(shortKey);
        assert.equal(await sep101Proxy.resultOfGet(), shortVal);

        await sep101Proxy.set(longKey, longVal);
        await sep101Proxy.get(longKey);
        assert.equal(await sep101Proxy.resultOfGet(), longVal);
    });

});
