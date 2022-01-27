const MyToken = artifacts.require("MyToken");

contract("MyToken(openzeppelin/token/ERC20)", async (accounts) => {

    let mytk;

    before(async () => {
        mytk = await MyToken.new(100000000);
    })

    it('events: Transfer', async () =>  {
        const from = accounts[0];
        const to = accounts[1]

        let result = await mytk.transfer(to, 1234, { from: from });
        // console.log(result);
        const transferLog = result.logs.find(element => element.event.match('Transfer'));
        assert.equal(transferLog.args.from, from);
        assert.equal(transferLog.args.to, to);
        assert.equal(transferLog.args.value, 1234);
    });

    it('events: Approval', async () =>  {
        const owner = accounts[0];
        const spender = accounts[2];

        let result = await mytk.approve(spender, 1234, { from: owner });
        // console.log(result);
        const approvalLog = result.logs.find(element => element.event.match('Approval'));
        assert.equal(approvalLog.args.owner, owner);
        assert.equal(approvalLog.args.spender, spender);
        assert.equal(approvalLog.args.value, 1234);
    });

});
