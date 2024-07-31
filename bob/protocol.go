package bob

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/noot/atomic-swap/judecoin"
	"github.com/noot/atomic-swap/swap-contract"
)

const defaultDaemonEndpoint = "http://127.0.0.1:16061/json_rpc"

var _ Bob = &bob{}

// Bob contains the functions that will be called by a user who owns JUDE
// and wishes to swap for ETH.
type Bob interface {
	// GenerateKeys generates Bob's spend and view keys (S_b, V_b)
	// It returns Bob's public spend key and his private view key, so that Alice can see
	// if the funds are locked.
	GenerateKeys() (*judecoin.PublicKey, *judecoin.PrivateViewKey, error)

	// SetAlicePublicKeys sets Alice's public spend and view keys
	SetAlicePublicKeys(*judecoin.PublicKeyPair)

	// SetContract sets the contract in which Alice has locked her ETH.
	SetContract(address ethcommon.Address) error

	// WatchForReady watches for Alice to call Ready() on the swap contract, allowing
	// Bob to call Claim().
	WatchForReady() (<-chan struct{}, error)

	// WatchForRefund watches for the Refund event in the contract.
	// This should be called before LockFunds.
	// If a keypair is sent over this channel, the rest of the protocol should be aborted.
	//
	// If Alice chooses to refund and thus reveals s_a,
	// the private spend and view keys that contain the previously locked judecoin
	// ((s_a + s_b), (v_a + v_b)) are sent over the channel.
	// Bob can then use these keys to move his funds if he wishes.
	WatchForRefund() (<-chan *judecoin.PrivateKeyPair, error)

	// LockFunds locks Bob's funds in the judecoin account specified by public key
	// (S_a + S_b), viewable with (V_a + V_b)
	// It accepts the amount to lock as the input
	// TODO: units
	LockFunds(amount uint) (judecoin.Address, error)

	// ClaimFunds redeem's Bob's funds on ethereum
	ClaimFunds() error
}

type bob struct {
	ctx    context.Context
	t0, t1 time.Time

	privkeys        *judecoin.PrivateKeyPair
	pubkeys         *judecoin.PublicKeyPair
	client          judecoin.Client
	daemonClient    judecoin.DaemonClient
	contract        *swap.Swap
	ethPrivKey      *ecdsa.PrivateKey
	alicePublicKeys *judecoin.PublicKeyPair
	ethClient       *ethclient.Client
	auth            *bind.TransactOpts
}

// NewBob returns a new instance of Bob.
// It accepts an endpoint to a judecoin-wallet-rpc instance where account 0 contains Bob's JUDE.
func NewBob(judecoinEndpoint, ethEndpoint, ethPrivKey string) (*bob, error) {
	pk, err := crypto.HexToECDSA(ethPrivKey)
	if err != nil {
		return nil, err
	}

	ec, err := ethclient.Dial(ethEndpoint)
	if err != nil {
		return nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(pk, big.NewInt(1337)) // ganache chainID
	if err != nil {
		return nil, err
	}

	return &bob{
		ctx:          context.Background(), // TODO: add cancel
		client:       judecoin.NewClient(judecoinEndpoint),
		daemonClient: judecoin.NewClient(defaultDaemonEndpoint), // TODO: pass through flags
		ethClient:    ec,
		ethPrivKey:   pk,
		auth:         auth,
	}, nil
}

// GenerateKeys generates Bob's spend and view keys (S_b, V_b)
func (b *bob) GenerateKeys() (*judecoin.PublicKey, *judecoin.PrivateViewKey, error) {
	var err error
	b.privkeys, err = judecoin.GenerateKeys()
	if err != nil {
		return nil, nil, err
	}

	b.pubkeys = b.privkeys.PublicKeyPair()
	return b.pubkeys.SpendKey(), b.privkeys.ViewKey(), nil
}

func (b *bob) SetAlicePublicKeys(sk *judecoin.PublicKeyPair) {
	b.alicePublicKeys = sk
}

func (b *bob) SetContract(address ethcommon.Address) error {
	var err error
	b.contract, err = swap.NewSwap(address, b.ethClient)
	return err
}

func (b *bob) WatchForReady() (<-chan struct{}, error) {
	watchOpts := &bind.WatchOpts{
		Context: b.ctx,
	}

	done := make(chan struct{})
	ch := make(chan *swap.SwapIsReady)
	defer close(done)

	// watch for Refund() event on chain, calculate unlock key as result
	sub, err := b.contract.WatchIsReady(watchOpts, ch)
	if err != nil {
		return nil, err
	}

	defer sub.Unsubscribe()

	go func() {
		select {
		case <-ch:
			// contract is ready!!
			close(done)
		case <-b.ctx.Done():
			return
		}
	}()

	return done, nil
}

func (b *bob) WatchForRefund() (<-chan *judecoin.PrivateKeyPair, error) {
	watchOpts := &bind.WatchOpts{
		Context: b.ctx,
	}

	out := make(chan *judecoin.PrivateKeyPair)
	ch := make(chan *swap.SwapRefunded)
	defer close(out)

	// watch for Refund() event on chain, calculate unlock key as result
	sub, err := b.contract.WatchRefunded(watchOpts, ch)
	if err != nil {
		return nil, err
	}

	defer sub.Unsubscribe()

	go func() {
		select {
		case refund := <-ch:
			// got Alice's secret
			saBytes := refund.S.Bytes()
			var sa [32]byte
			copy(sa[:], saBytes)

			skA, err := judecoin.NewPrivateSpendKey(sa[:])
			if err != nil {
				fmt.Printf("failed to convert Alice's secret into a key: %w", err)
				return
			}

			vkA, err := skA.View()
			if err != nil {
				fmt.Printf("failed to get view key from Alice's secret spend key: %w", err)
				return
			}

			skAB := judecoin.SumPrivateSpendKeys(skA, b.privkeys.SpendKey())
			vkAB := judecoin.SumPrivateViewKeys(vkA, b.privkeys.ViewKey())
			kpAB := judecoin.NewPrivateKeyPair(skAB, vkAB)
			out <- kpAB
		case <-b.ctx.Done():
			return
		}
	}()

	return out, nil
}

func (b *bob) LockFunds(amount uint) (judecoin.Address, error) {
	kp := judecoin.SumSpendAndViewKeys(b.alicePublicKeys, b.pubkeys)

	fmt.Println("Bob: going to lock funds...")

	balance, err := b.client.GetBalance(0)
	if err != nil {
		return "", err
	}

	fmt.Println("balance: ", balance.Balance)
	fmt.Println("unlocked balance: ", balance.UnlockedBalance)
	fmt.Println("blocks to unlock: ", balance.BlocksToUnlock)

	address := kp.Address()
	if err := b.client.Transfer(address, 0, amount); err != nil {
		return "", err
	}

	bobAddr, err := b.client.GetAddress(0)
	if err != nil {
		return "", err
	}

	if err := b.daemonClient.GenerateBlocks(bobAddr.Address, 1); err != nil {
		return "", err
	}

	fmt.Println("Bob: successfully locked funds")
	fmt.Println("address: ", address)
	return address, nil
}

func (b *bob) ClaimFunds() error {
	txOpts := &bind.TransactOpts{
		From:   b.auth.From,
		Signer: b.auth.Signer,
	}
	// call swap.Swap.Claim() w/ b.privkeys.sk, revealing Bob's secret spend key
	secret := b.privkeys.Bytes()
	s := big.NewInt(0).SetBytes(secret)
	_, err := b.contract.Claim(txOpts, s)
	return err
}
