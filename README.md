# ETH-JUDE Atomic Swaps

This is a prototype of ETH<->JUDE atomic swaps

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

Start judecoin-wallet-rpc for Alice:
```
./judecoin-wallet-rpc  --rpc-bind-port 16064 --password "" --disable-rpc-login --wallet-dir .
```

##### Compiling contract bindings

Download solc v0.6.12

```
./solc-static-linux --bin contracts/contracts/Swap.sol -o contracts/bin/ --overwrite
./solc-static-linux --abi contracts/contracts/Swap.sol -o contracts/abi/ --overwrite
```

Generate the bindings
```
./scripts/generate-bindings.sh
```

##### Testing
To run tests on the go bindings, execute
```
go test ./swap-contract
```
./abigen --abi contracts/abi/Swap.abi --pkg swap --type Swap --out swap.go --bin contracts/bin/Swap.bin 
```