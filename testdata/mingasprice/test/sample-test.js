const { expect } = require("chai");
const { ethers } = require("hardhat");
const {BigNumber} = require("ethers");

describe("Vote", function () {
  it("proposal and vote", async function () {
    const MinGasPriceVote = await ethers.getContractFactory("MinGasPriceVote");
    const minGasPriceVote = await MinGasPriceVote.deploy();
    await minGasPriceVote.deployed();

    const staking = await MinGasPriceVote.attach("0x0000000000000000000000000000000000002710");
    let val1, val2, val3;
    [val1, val2, val3] = await ethers.getSigners();

    console.log("test invalid vote: not in proposal");
    const tx1 = await staking.connect(val2).vote(3 * 10 **9);
    await expect(tx1.wait()).to.be.reverted;

    console.log("test normal proposal");
    const target = 5 * 10**9;
    const tx2 = await staking.connect(val1).proposal(target);
    await tx2.wait();
    expect(await staking.getVote(val1.address)).to.equal(target);

    console.log("test repeat proposal");
    const tx3 = await staking.connect(val1).proposal(target);
    await expect(tx3.wait()).to.be.reverted;

    console.log("test normal vote");
    const tx4 = await staking.connect(val2).vote(3 * 10 **9);
    await tx4.wait();
    expect(await staking.getVote(val2.address)).to.equal(3 * 10**9);

    console.log("test invalid vote: not validator");
    const tx = await staking.connect(val3).vote(3 * 10 **9);
    await expect(tx.wait()).to.be.reverted;

    console.log("test invalid vote: gas price too small");
    const tx5 = await staking.connect(val3).vote(3 * 10 **6);
    await expect(tx5.wait()).to.be.reverted;

    console.log("test proposal not finished");
    const tx6 = await staking.connect(val1).executeProposal();
    await expect(tx6.wait()).to.be.reverted;

    await new Promise(r => setTimeout(r, 22 * 1000));

    console.log("test invalid vote: proposal has finished");
    const tx7 = await staking.connect(val3).vote(3 * 10 **9);
    await expect(tx7.wait()).to.be.reverted;

    console.log("test normal execute proposal");
    const tx8 = await staking.connect(val1).executeProposal();
    await tx8.wait();

    console.log("test invalid execute proposal: not in proposal");
    const tx9 = await staking.connect(val1).executeProposal();
    await expect(tx9.wait()).to.be.reverted;

    const gasPrice = await ethers.provider.getGasPrice();
    console.log(gasPrice);
    expect(gasPrice).to.equal(BigNumber.from((3 + 5 * 10)*10**9).div(BigNumber.from(11)).add(BigNumber.from(4 * 10**9)).div(2));
  });
});
