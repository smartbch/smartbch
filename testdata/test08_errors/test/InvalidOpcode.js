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
   >       "to": "0xeb79f8fc7213c8433c2fd9a69b33ea1f7bb4e9b3",
   >       "data": "0x12f28d510000000000000000000000000000000000000000000000000000000000000064"
   >     }
   >   ]
   > }
 <   {
 <     "id": 15,
 <     "jsonrpc": "2.0",
 <     "result": "0x67c6014e429dbdda1bce7fb6dddac27525b5c9adbc0f5bdf745a41948efb616d",
 <     "error": {
 <       "message": "VM Exception while processing transaction: invalid opcode",
 <       "code": -32000,
 <       "data": {
 <         "0x67c6014e429dbdda1bce7fb6dddac27525b5c9adbc0f5bdf745a41948efb616d": {
 <           "error": "invalid opcode",
 <           "program_counter": 201,
 <           "return": "0x"
 <         },
 <         "stack": "RuntimeError: VM Exception while processing transaction: invalid opcode\n    at Function.RuntimeError.fromResults (/Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/utils/runtimeerror.js:94:13)\n    at BlockchainDouble.processBlock (/Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/blockchain_double.js:627:24)\n    at runMicrotasks (<anonymous>)\n    at processTicksAndRejections (internal/process/task_queues.js:93:5)",
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
   >       "to": "0xeb79f8fc7213c8433c2fd9a69b33ea1f7bb4e9b3",
   >       "data": "0x12f28d510000000000000000000000000000000000000000000000000000000000000064"
   >     },
   >     "latest"
   >   ],
   >   "id": 1615259255123
   > }
 <   {
 <     "id": 1615259255123,
 <     "jsonrpc": "2.0",
 <     "error": {
 <       "message": "VM Exception while processing transaction: invalid opcode",
 <       "code": -32000,
 <       "data": {
 <         "0x6141a1a3860ff100cafbf4de7ff00942f2a160d85dc62430fd71fea59eb1ab61": {
 <           "error": "invalid opcode",
 <           "program_counter": 201,
 <           "return": "0x"
 <         },
 <         "stack": "RuntimeError: VM Exception while processing transaction: invalid opcode\n    at Function.RuntimeError.fromResults (/Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/utils/runtimeerror.js:94:13)\n    at /Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/blockchain_double.js:568:26",
 <         "name": "RuntimeError"
 <       }
 <     }
 <   }
*/
    it('invalid opcode', async () => {
        const contract = await Errors.new({ from: accounts[0] });
        try {
            await await contract.setN_invalidOpcode(100);
            throw null;
        } catch (error) {
            assert(error, "Expected an error but did not get one");
            assert.equal(error.message, 
                "Returned error: VM Exception while processing transaction: invalid opcode");
        }
    })

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
   >       "data": "0x12f28d510000000000000000000000000000000000000000000000000000000000000064",
   >       "to": "0x820386c268bf39a36332fb022fb45a8c9a9a330f"
   >     }
   >   ]
   > }
 <   {
 <     "id": 24,
 <     "jsonrpc": "2.0",
 <     "error": {
 <       "message": "VM Exception while processing transaction: invalid opcode",
 <       "code": -32000,
 <       "data": {
 <         "stack": "RuntimeError: VM Exception while processing transaction: invalid opcode\n    at Function.RuntimeError.fromResults (/Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/utils/runtimeerror.js:94:13)\n    at module.exports (/Applications/Ganache.app/Contents/Resources/static/node/node_modules/ganache-core/lib/utils/gas/guestimation.js:142:32)",
 <         "name": "RuntimeError"
 <       }
 <     }
 <   }
*/
    it('invalid opcode, estimateGas', async () => {
        const contract = await Errors.new({ from: accounts[0] });
        try {
            await await contract.setN_invalidOpcode.estimateGas(100);
            throw null;
        } catch (error) {
            assert(error, "Expected an error but did not get one");
            assert.equal(error.message, 
                "Returned error: VM Exception while processing transaction: invalid opcode");
        }
    })

});
