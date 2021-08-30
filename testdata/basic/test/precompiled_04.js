const Precompiled04 = artifacts.require("Precompiled04");

contract("Precompiled", async (accounts) => {

    it('0x04', async () => {
        const p04 = await Precompiled04.new();

        const ret1 = await p04.callDatacopy1.call('0x11223344556677889900');
        assert.equal(ret1, '0x11223344556677889900');

        const ret2 = await p04.callDatacopy2.call('0x111111112222222233333333444444445555555566666666777777778888888899999999aaaaaaaa');
        assert.equal(ret2, '0x11111111222222111111112222222233333333444444445555555566666666777777778888888899');
    });

});
