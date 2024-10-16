# ETH-JUDE Atomic Swaps

This is a prototype of ETH<->JUDE atomic swaps

### Protocol

Alice has ETH and wants JUDE, Bob has JUDE and wants ETH. They come to an agreement to do the swap and the amounts they will swap.

#### Ethereum smart contract interface

Offline phase:
- Both parties Alice and Bob select a secret: `s_a` & `s_b`, which are used to construct valid points on the ed25519 curve: `P_ed_a` and `P_ed_b` accordingly. The parties share their public keys with each other.

##### Step 1.
Alice creates a smart contract on Ethereum and sends her ether to it. The contract has the following properties:
- it is non-destructible
- it is initiated with `P_ed_a` & `P_ed_b`
- it has a `ready` function, which can only be called by Alice. Once `ready` is invoked (in step 3), Bob can proceed with redeeming his ether. Alice has `t_0` time period to call `ready`, otherwise all funds are transferred to Bob.
- it has a `refund` function that can only be called by Alice, and only before `ready` is called. Once `ready` is invoked, Alice can no longer call `refund` within the next time duration`t_1`. After `t_1` has passed though, and Bob hasn't called `redeem`, `refund` is re-enabled to prevent a dead-lock.
- `refund` takes one parameter from Alice: `s_a`. This allows Alice to get her ether back in case Bob mis-behaves, but as it's public from now on, Bob can still redeem his judecoin (in case he was offline). 
- `redeem` takes one parameter from Bob: `s_b`. At this point, Alice found out Bob's secret and can claim Judecoin by combining her and Bob's secrets.
- both the `refund` & `redeem` functions check, whether the argument `s_x` provided indeed corresponds to the public key `P_ed_x` and only if true, allows fund transfer.

##### Step 2. 
Bob sees the smart contract being created. He sends his judecoin to an account address constructed from `P_ed_a + P_ed_b`.

The funds can only be accessed by an entity having both `s_a` & `s_b`.

##### Step 3.
Alice sees that judecoin has been sent. She calls `ready` on the smart contract.
From this point on, Bob can redeem his ether by calling `redeem(s_b)`.

By redeeming, Bob revealed his secret. Now Alice is the only one that has both `s_a` & `s_b` and she can access the judecoin in the account created from `P_ed_a + P_ed_b`.

#### What could go wrong

##### Step 2.

- Alice locked her ether, but Bob doesn't lock his judecoin.
Alice never calls `ready`, and can instead call `refund(s_a)` within `t_0`, getting her ether back.


##### Step 3.

- Alice called `ready`, but Bob never redeems. Deadlocks are prevented thanks to a second timelock `t_1`, which re-enables Alice to call refund.

- Alice never calls `ready` within `t_1`. Bob can still claim his ETH, after `t_1` has passed, and JUDE remains locked forever.


### Instructions

Start ganache-cli with determinstic keys:
```
ganache-cli -d
```

Start judecoind for regtest:
```
./judecoind --regtest --fixed-difficulty=1 --rpc-bind-port 16061 --offline
```

Start judecoin-wallet-rpc for Bob with some wallet that has regtest judecoin:
```
./judecoin-wallet-rpc  --rpc-bind-port 16063 --password "" --disable-rpc-login --wallet-file test-wallet
```

Determine the address of `test-wallet` by running `judecoin-wallet-cli` and `address all`

Then, mine some blocks on the judecoin test chain by running the following RPC command, replacing the address with the one from the previous step:
```
curl -X POST http://127.0.0.1:16061/json_rpc -d '{"jsonrpc":"2.0","id":"0","method":"generateblocks","params":{ "wallet_address":"6j3gQmw1qHCCoc3iacrJgh79UiuoQgW4KDe4aLh5Dww2N6B8iWhuzB1StgCEWjpt4YNVzfrQLRB82XfgjcYgcnHWSiKGhny","amount_of_blocks":100}' -H 'Content-Type: application/json'
```

This will deposit some JUDE in your account.

Start judecoin-wallet-rpc for Alice:
```
./judecoin-wallet-rpc  --rpc-bind-port 16064 --password "" --disable-rpc-login --wallet-dir .
```

Build binary:
```
./scripts/build.sh
```

This creates an `atomic-swap` binary in the root directory.

To run as Alice, execute:
```
./atomic-swap --amount 1 --alice
```

Alice will print out a libp2p node address, for example `/ip4/127.0.0.1/tcp/9933/p2p/12D3KooWBW1cqB9t5fKP8yZPq3PcWcgbvuNai5ZpAeWFAbs5RNAA`. This will be used for Bob to connect.

To run as Bob and connect to Alice, replace the bootnode in the following line with what Alice logged, and execute:

```
./atomic-swap --amount 1 --bob --bootnodes /ip4/127.0.0.1/tcp/9933/p2p/12D3KooWBW1cqB9t5fKP8yZPq3PcWcgbvuNai5ZpAeWFAbs5RNAA
```

If all goes well, you should see Alice and Bob successfully exchange messages and execute the swap protocol. The result is that Alice now owns the private key to a Judecoin account (and is the only owner of that key) and Bob has the ETH transferred to him.


##### Compiling contract bindings

Download solc v0.8.9

Set `SOLC_BIN` to the downloaded binary
```
export SOLC_BIN=solc
./solc-static-linux --bin contracts/contracts/Swap.sol -o contracts/bin/ --overwrite
./solc-static-linux --abi contracts/contracts/Swap.sol -o contracts/abi/ --overwrite
```

Generate the bindings
```
./scripts/generate-bindings.sh
```

##### Testing
To run tests on the go bindings, execute:
```
go test ./swap-contract
```

```

./abigen --abi contracts/abi/Swap.abi --pkg swap --type Swap --out swap.go --bin contracts/bin/Swap.bin 
```

This will test the main protocol functionality on the ethereum side:
1. Success case, where both parties obey the protocol
2. Case where Bob never locks judecoin on his side. Alice can Refund
3. Case where Bob locks judecoin, but never claims his ether from the contract

Upon Refund/Claim by either side, they reveal the secret to the counterparty, which *always* guarantees that the counteryparty can claim the locked funds on ethereum.
