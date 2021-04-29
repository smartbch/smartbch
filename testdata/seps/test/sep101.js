const ISEP101 = artifacts.require("ISEP101");
const SEP101Proxy_DELEGATECALL = artifacts.require("SEP101Proxy_DELEGATECALL");
const SEP101Proxy_CALLCODE     = artifacts.require("SEP101Proxy_CALLCODE");
const SEP101Proxy_CALL         = artifacts.require("SEP101Proxy_CALL");
const SEP101Proxy_STATICCALL   = artifacts.require("SEP101Proxy_STATICCALL");

const shortKey = "0x1234";
const shortVal = "0x5678";
const longKey  = "0x0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789";
const longVal  = "0x9876543210987654321098765432109876543210987654321098765432109876543210987654321098765432109876543210";

const shortKeySHA256 = "0x3a103a4e5729ad68c02a678ae39accfbc0ae208096437401b7ceab63cca0622f";
const longKeySHA256  = "0x1ce7576ec9158575ea90caa3acd6a0ae8c4e014bcc8fc34f3bb801b90760dbc0";

contract("SEP101", async (accounts) => {

    it('get/set: from EOA', async () => {
        const sep101 = new ISEP101("0x0000000000000000000000000000000000002712");
        try {
            await sep101.set(shortKey, shortVal);
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
        try {
            await sep101.get(shortKey);
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

});

contract("SEP101Proxy_DELEGATECALL", async (accounts) => {

    let sep101Proxy;

    before(async () => {
        sep101Proxy = await SEP101Proxy_DELEGATECALL.new();
        sep101Proxy = new ISEP101(sep101Proxy.address);
    });

    it('get/set: delegate_call', async () => {
        await sep101Proxy.set(shortKey, shortVal);
        assert.equal(await sep101Proxy.get(shortKey), shortVal);

        await sep101Proxy.set(longKey, longVal);
        assert.equal(await sep101Proxy.get(longKey), longVal);

        // read by getStorageAt()
        assert.equal(await web3.eth.getStorageAt(sep101Proxy.address, shortKeySHA256), shortVal);
        assert.equal(await web3.eth.getStorageAt(sep101Proxy.address, longKeySHA256), longVal);
    });

    it('get: non-existing key', async () => {
        assert.equal(await sep101Proxy.get("0x99999999"), null);
    });

    it('set: zero-length val', async () => {
        await sep101Proxy.set(shortKey, []);
        assert.equal(await sep101Proxy.get(shortKey), null);
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
        assert.equal(await sep101Proxy.get(maxLenKey), shortVal);

        try {
            await sep101Proxy.set(maxLenKey + "ff", shortVal);
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

    it('set: val too large', async () => {
        const maxLenVal = "0x" + "cd".repeat(24*1024);
        await sep101Proxy.set(shortKey, maxLenVal);
        assert.equal(await sep101Proxy.get(shortKey), maxLenVal);

        try {
            await sep101Proxy.set(shortKey, maxLenVal + "ff");
            throw null;
        } catch (e) {
            assert(e, "Expected an error but did not get one");
        }
    });

});

contract("SEP101Proxy_CALLCODE", async (accounts) => {

    it('get/set: call_code', async () => {
        let sep101Proxy = await SEP101Proxy_CALLCODE.new();
        sep101Proxy = new ISEP101(sep101Proxy.address);

        await sep101Proxy.set(shortKey, shortVal);
        assert.equal(await sep101Proxy.get(shortKey), shortVal);

        await sep101Proxy.set(longKey, longVal);
        assert.equal(await sep101Proxy.get(longKey), longVal);

        // read by getStorageAt()
        assert.equal(await web3.eth.getStorageAt(sep101Proxy.address, shortKeySHA256), shortVal);
        assert.equal(await web3.eth.getStorageAt(sep101Proxy.address, longKeySHA256), longVal);
    });

});

contract("SEP101Proxy_CALL", async (accounts) => {

    it('get/set: call', async () => {
        let sep101Proxy = await SEP101Proxy_CALL.new();
        sep101Proxy = new ISEP101(sep101Proxy.address);

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

});

contract("SEP101Proxy_STATICCALL", async (accounts) => {

    it('get/set: staticcall', async () => {
        let sep101Proxy = await SEP101Proxy_STATICCALL.new();
        sep101Proxy = new ISEP101(sep101Proxy.address);
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
