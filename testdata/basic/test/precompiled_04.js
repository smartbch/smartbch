const Precompiled04 = artifacts.require("Precompiled04");

contract("Precompiled", async (accounts) => {

    it('0x04', async () => {
        const p04 = await Precompiled04.new();

        const ret1 = await p04.callDatacopy1.call('0x11223344556677889900');
        assert.equal(ret1, '0x11223344556677889900');

        const ret2 = await p04.callDatacopy2.call('0x11223344556677889900');
        assert.equal(ret2, '0x11223344556677112233');
    });

});
