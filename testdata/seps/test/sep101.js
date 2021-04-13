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
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
        try {
            await sep101.get_call(shortKey);
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
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

        await sep101Proxy.set(shortKey, []);
        await sep101Proxy.get(shortKey);
        assert.equal(await sep101Proxy.resultOfGet(), null);
    });

    it('set: zero-length key', async () => {
        try {
            await sep101Proxy.set([], shortVal);
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('set: key too large', async () => {
        const maxLenKey = "0x" + "ab".repeat(256);
        await sep101Proxy.set(maxLenKey, shortVal);
        await sep101Proxy.get(maxLenKey);
        assert.equal(await sep101Proxy.resultOfGet(), shortVal);

        try {
            await sep101Proxy.set(maxLenKey + "0", shortVal);
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('set: val too large', async () => {
        const maxLenVal = "0x" + "cd".repeat(24576);
        await sep101Proxy.set(shortKey, maxLenVal);
        await sep101Proxy.get(shortKey);
        assert.equal(await sep101Proxy.resultOfGet(), maxLenVal);

        try {
            await sep101Proxy.set(shortKey, maxLenVal + "0");
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('get/set: call_code', async () => {
    //     await sep101Proxy.set_callcode(shortKey, shortVal);
    //     await sep101Proxy.get_callcode(shortKey);
    //     assert.equal(await sep101Proxy.resultOfGet(), shortVal);

    //     await sep101Proxy.set_callcode(longKey, longVal);
    //     await sep101Proxy.get_callcode(longKey);
    //     assert.equal(await sep101Proxy.resultOfGet(), longVal);
    });

    it('get/set: call', async () => {
        try {
            await sep101Proxy.set_call(shortKey, shortVal);
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
        try {
            await sep101Proxy.get_call(shortKey);
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('get/set: staticcall', async () => {
        try {
            await sep101Proxy.set_staticcall(shortKey, shortVal);
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
        try {
            await sep101Proxy.get_staticcall(shortKey);
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

});
