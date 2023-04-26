# ETH-JUDE Atomic Swaps

This is a prototype of ETH<->JUDE atomic swaps

### Instructions

Start ganache-cli with determinstic keys:
```
ganache-cli -d
```

Start judecoind:
```
judecoind
```

Start judecoin-wallet-rpc:
```
./judecoin-wallet-rpc --rpc-bind-port 16062 --password "" --disable-rpc-login --wallet-dir .
```

##### Compiling contract bindings

Download solc v0.6.12

```
./solc-static-linux --bin contracts/contracts/Swap.sol -o contracts/bin/ --overwrite
./solc-static-linux --abi contracts/contracts/Swap.sol -o contracts/abi/ --overwrite
```

```
abigen --abi contracts/abi/Swap.abi --pkg swap --type Swap --out swap.go --bin contracts/bin/Swap.bin 
```