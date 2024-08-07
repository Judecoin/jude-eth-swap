package swap

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/noot/atomic-swap/judecoin"
)

const (
	keyAlice = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"
	keyBob   = "6cbed15c793ce57650b9877cf6fa156fbef513c4e6134f022a85b1ffdd59b2a1"
)

func reverse(s []byte) []byte {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func setBigIntLE(s []byte) *big.Int {
	s = reverse(s)
	return big.NewInt(0).SetBytes(s)
}

func TestDeploySwap(t *testing.T) {
	conn, err := ethclient.Dial("http://127.0.0.1:8545")
	require.NoError(t, err)

	pk_a, err := crypto.HexToECDSA(keyAlice)
	require.NoError(t, err)

	authAlice, err := bind.NewKeyedTransactorWithChainID(pk_a, big.NewInt(1337)) // ganache chainID
	require.NoError(t, err)

	address, tx, swapContract, err := DeploySwap(authAlice, conn, [32]byte{}, [32]byte{})
	require.NoError(t, err)

	t.Log(address)
	t.Log(tx)
	t.Log(swapContract)
}

func encodePublicKey(pub *ecdsa.PublicKey) [64]byte {
	px := pub.X.Bytes()
	py := pub.Y.Bytes()
	var p [64]byte
	copy(p[0:32], px)
	copy(p[32:64], py)
	return p
}

func TestSwap_Claim(t *testing.T) {
	// Alice generates key
	keyPairAlice, err := judecoin.GenerateKeys()
	// keyPairAlice, err := crypto.GenerateKey()
	require.NoError(t, err)
	pubKeyAlice := keyPairAlice.PublicKeyPair().SpendKey().Bytes()

	// Bob generates key
	keyPairBob, err := judecoin.GenerateKeys()
	require.NoError(t, err)
	pubKeyBob := keyPairBob.PublicKeyPair().SpendKey().Bytes()

	secretBob := keyPairBob.Bytes()

	// setup
	conn, err := ethclient.Dial("ws://127.0.0.1:8545")
	require.NoError(t, err)

	pk_a, err := crypto.HexToECDSA(keyAlice)
	require.NoError(t, err)
	pk_b, err := crypto.HexToECDSA(keyBob)
	require.NoError(t, err)

	authAlice, err := bind.NewKeyedTransactorWithChainID(pk_a, big.NewInt(1337)) // ganache chainID
	require.NoError(t, err)
	authBob, err := bind.NewKeyedTransactorWithChainID(pk_b, big.NewInt(1337)) // ganache chainID
	require.NoError(t, err)

	var pkAliceFixed [32]byte
	copy(pkAliceFixed[:], pubKeyAlice)
	var pkBobFixed [32]byte
	copy(pkBobFixed[:], pubKeyBob)
	_, _, swap, err := DeploySwap(authAlice, conn, pkBobFixed, pkAliceFixed)
	require.NoError(t, err)

	txOpts := &bind.TransactOpts{
		From:   authAlice.From,
		Signer: authAlice.Signer,
	}

	txOptsBob := &bind.TransactOpts{
		From:   authBob.From,
		Signer: authBob.Signer,
	}

	// Bob tries to claim before Alice has called ready, should fail
	s := big.NewInt(0).SetBytes(reverse(secretBob))
	_, err = swap.Claim(txOptsBob, s)
	require.Errorf(t, err, "'isReady == false' cannot claim yet!")

	// Alice calls set_ready on the contract
	_, err = swap.SetReady(txOpts)
	require.NoError(t, err)

	watchForEvent(t, swap)

	_, err = swap.Claim(txOptsBob, s)
	require.NoError(t, err)

	time.Sleep(time.Second * 10)

	// TODO check whether Bob's account balance has increased

}

func TestSwap_Refund(t *testing.T) {

}

func watchForEvent(t *testing.T, contract *Swap) {
	watchOpts := &bind.WatchOpts{
		Context: context.Background(),
	}

	ch := make(chan *SwapCalculatedPublicKey)

	sub, err := contract.WatchCalculatedPublicKey(watchOpts, ch)
	require.NoError(t, err)

	defer sub.Unsubscribe()

	go func() {
		for event := range ch {
			if event == nil {
				continue
			}

			fmt.Println("got event")
			fmt.Println(event.Px, event.Py)
			return
		}
	}()
}
