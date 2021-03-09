const Errors = artifacts.require("Errors");

contract("Errors", async (accounts) => {

/*
   > {
   >   "jsonrpc": "2.0",
   >   "id": 15,
   >   "method": "eth_sendTransaction",
   >   "params": [
   >     {
   >       "from": "0x60625836b783cd68fe512e22070eae570d6ad669",
   >       "gas": "0x6691b7",
   >       "gasPrice": "0x4a817c800",
   >       "to": "0x79ffeedc0ef53e0fb56e77ead0580b77ddd2d634",
   >       "data": "0xe0ada09a0000000000000000000000000000000000000000000000000000000000000064"
   >     }
   >   ]
   > }
 <   {
 <     "id": 15,
 <     "jsonrpc": "2.0",
 <     "result": "0x0e3948e116527283dd144bf2cb5fc4acbf6b69ab5bd5f12b2f53efe87fc5696d",
 <     "error": {
 <       "message": "VM Exception while processing transaction: revert n must be less than 10",
 <       "code": -32000,
 <       "data": {
 <         "0x0e3948e116527283dd144bf2cb5fc4acbf6b69ab5bd5f12b2f53efe87fc5696d": {
 <           "error": "revert",
 <           "program_counter": 294,
 <           "return": "0x08c379a0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000166e206d757374206265206c657373207468616e20313000000000000000000000",
 <           "reason": "n must be less than 10"
 <         },
 <         "stack": "RuntimeError: VM Exception while processing transaction: revert n must be less than 10\n    at Function.RuntimeError.fromResults (/Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/utils/runtimeerror.js:94:13)\n    at BlockchainDouble.processBlock (/Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/blockchain_double.js:627:24)\n    at runMicrotasks (<anonymous>)\n    at processTicksAndRejections (internal/process/task_queues.js:93:5)",
 <         "name": "RuntimeError"
 <       }
 <     }
 <   }
   > {
   >   "jsonrpc": "2.0",
   >   "method": "eth_call",
   >   "params": [
   >     {
   >       "from": "0x60625836b783cd68fe512e22070eae570d6ad669",
   >       "gas": "0x6691b7",
   >       "gasPrice": "0x4a817c800",
   >       "to": "0x79ffeedc0ef53e0fb56e77ead0580b77ddd2d634",
   >       "data": "0xe0ada09a0000000000000000000000000000000000000000000000000000000000000064"
   >     },
   >     "latest"
   >   ],
   >   "id": 1615258595899
   > }
 <   {
 <     "id": 1615258595899,
 <     "jsonrpc": "2.0",
 <     "error": {
 <       "message": "VM Exception while processing transaction: revert n must be less than 10",
 <       "code": -32000,
 <       "data": {
 <         "0x7a4bdf16dddeacd6cd17d4e315efbd8913a30836460d7fb36af19af455cd230a": {
 <           "error": "revert",
 <           "program_counter": 294,
 <           "return": "0x08c379a0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000166e206d757374206265206c657373207468616e20313000000000000000000000",
 <           "reason": "n must be less than 10"
 <         },
 <         "stack": "RuntimeError: VM Exception while processing transaction: revert n must be less than 10\n    at Function.RuntimeError.fromResults (/Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/utils/runtimeerror.js:94:13)\n    at /Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/blockchain_double.js:568:26",
 <         "name": "RuntimeError"
 <       }
 <     }
 <   }
*/
    it('revert', async () => {
        const contract = await Errors.new({ from: accounts[0] });
        try {
            await await contract.setN_revert(100);
            throw null;
        } catch (error) {
            assert(error, "Expected an error but did not get one");
            assert.equal(error.message, 
                "Returned error: VM Exception while processing transaction: revert n must be less than 10 -- Reason given: n must be less than 10.");
        }
    });

/*
   > {
   >   "jsonrpc": "2.0",
   >   "id": 24,
   >   "method": "eth_estimateGas",
   >   "params": [
   >     {
   >       "from": "0x60625836b783cd68fe512e22070eae570d6ad669",
   >       "gas": "0x6691b7",
   >       "gasPrice": "0x4a817c800",
   >       "data": "0xe0ada09a0000000000000000000000000000000000000000000000000000000000000064",
   >       "to": "0x62f1a7730b7e5fd8bab7fb86a7e049723b46ee15"
   >     }
   >   ]
   > }
 <   {
 <     "id": 24,
 <     "jsonrpc": "2.0",
 <     "error": {
 <       "message": "VM Exception while processing transaction: revert n must be less than 10",
 <       "code": -32000,
 <       "data": {
 <         "stack": "RuntimeError: VM Exception while processing transaction: revert n must be less than 10\n    at Function.RuntimeError.fromResults (/Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/utils/runtimeerror.js:94:13)\n    at module.exports (/Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/utils/gas/guestimation.js:142:32)",
 <         "name": "RuntimeError"
 <       }
 <     }
 <   }
*/
    it('revert, estimateGas', async () => {
        const contract = await Errors.new({ from: accounts[0] });
        try {
            await await contract.setN_revert.estimateGas(100);
            throw null;
        } catch (error) {
            assert(error, "Expected an error but did not get one");
            assert.equal(error.message, 
                "Returned error: VM Exception while processing transaction: revert n must be less than 10");
        }
    });

});
