const ISEP101 = artifacts.require("ISEP101");
const SEP101Proxy1 = artifacts.require("SEP101Proxy");
const SEP101Proxy2 = artifacts.require("SEP101Proxy2");

const shortKey = "0x1234";
const shortVal = "0x5678";
const longKey = "0x0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789";
const longVal = "0x9876543210987654321098765432109876543210987654321098765432109876543210987654321098765432109876543210";

contract("SEP101", async (accounts) => {

    it('get/set: from EOA', async () => {
        const sep101 = new ISEP101("0x0000000000000000000000000000000000002712");
        try {
            await sep101.set_call(shortKey, shortVal);
            fail("error expected");
        } catch (e) {}
        try {
            await sep101.get_call(shortKey);
            fail("error expected");
        } catch (e) {}
    });

});

contract("SEP101Proxy1", async (accounts) => {

    let sep101Proxy;

    before(async () => {
        sep101Proxy = await SEP101Proxy1.new()
    });

    it('get/set: delegate_call', async () => {
        await sep101Proxy.set(shortKey, shortVal);
        await sep101Proxy.get(shortKey);
        assert.equal(await sep101Proxy.resultOfGet(), shortVal);

        await sep101Proxy.set(longKey, longVal);
        await sep101Proxy.get(longKey);
        assert.equal(await sep101Proxy.resultOfGet(), longVal);
    });

});

contract("SEP101Proxy2", async (accounts) => {

    let sep101Proxy;

    before(async () => {
        sep101Proxy = await SEP101Proxy2.new()
    });

    it('get: non-existing key', async () => {
        await sep101Proxy.get("0x999999999");
        assert.equal(await sep101Proxy.resultOfGet(), null);
    });

    it('set: zero-length val', async () => {
        await sep101Proxy.set_zero_len_val(shortKey);
        await sep101Proxy.get(shortKey);
        assert.equal(await sep101Proxy.resultOfGet(), null);
    });

    // it('get/set: call_code', async () => {
    // //     await sep101Proxy.set_callcode(shortKey, shortVal);
    // //     await sep101Proxy.get_callcode(shortKey);
    // //     assert.equal(await sep101Proxy.resultOfGet(), shortVal);

    // //     await sep101Proxy.set_callcode(longKey, longVal);
    // //     await sep101Proxy.get_callcode(longKey);
    // //     assert.equal(await sep101Proxy.resultOfGet(), longVal);
    // });

    it('get/set: call', async () => {
        try {
            await sep101Proxy.set_call(shortKey, shortVal);
            fail("error expected");
        } catch (e) {}
        try {
            await sep101Proxy.get_call(shortKey);
            fail("error expected");
        } catch (e) {}
    });

    it('get/set: staticcall', async () => {
        try {
            await sep101Proxy.set_staticcall(shortKey, shortVal);
            fail("error expected");
        } catch (e) {}
        try {
            await sep101Proxy.get_staticcall(shortKey);
            fail("error expected");
        } catch (e) {}
    });

});
