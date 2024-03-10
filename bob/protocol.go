package bob

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/judecoin/jude-eth-swap/judecoin"
	"github.com/judecoin/jude-eth-swap/swap-contract"
)

var _ Bob = &bob{}

// Bob contains the functions that will be called by a user who owns JUDE
// and wishes to swap for ETH.
type Bob interface {
	// GenerateKeys generates Bob's public spend and view keys (S_b, V_b)
	GenerateKeys() error

	// SetContract sets the contract in which Alice has locked her ETH.
	SetContract(*swap.Swap)

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
	// It accepts Alice's public keys (S_a, V_a) as input, as well as the amount to lock
	// TODO: units
	LockFunds(aliceKeys *judecoin.PublicKeyPair, amount uint) error

	// RedeemFunds redeem's Bob's funds on ethereum
	RedeemFunds() error
}

type bob struct {
	t0, t1 time.Time

	privkeys   *judecoin.PrivateKeyPair
	pubkeys    *judecoin.PublicKeyPair
	client     judecoin.Client
	contract   *swap.Swap
	ethPrivKey *ecdsa.PrivateKey
}

// NewBob returns a new instance of Bob.
// It accepts an endpoint to a judecoin-wallet-rpc instance where account 0 contains Bob's JUDE.
func NewBob(endpoint string, ethPrivKey string, t0, t1 time.Time) (*bob, error) {
	pk, err := crypto.HexToECDSA(ethPrivKey)
	if err != nil {
		return nil, err
	}

	return &bob{
		t0:         t0,
		t1:         t1,
		client:     judecoin.NewClient(endpoint),
		ethPrivKey: pk,
	}, nil
}

// GenerateKeys generates Bob's public spend and view keys (S_b, V_b)
func (b *bob) GenerateKeys() error {
	var err error
	b.privkeys, err = judecoin.GenerateKeys()
	if err != nil {
		return err
	}

	b.pubkeys = b.privkeys.PublicKeyPair()
	return nil
}

func (b *bob) SetContract(contract *swap.Swap) {
	b.contract = contract
}

func (b *bob) WatchForReady() (<-chan struct{}, error) {
	return nil, nil
}

func (b *bob) WatchForRefund() (<-chan *judecoin.PrivateKeyPair, error) {
	// watch for Refund() event on chain, calculate unlock key as result
	return nil, nil
}

func (b *bob) LockFunds(akp *judecoin.PublicKeyPair, amount uint) error {
	kp := judecoin.SumSpendAndViewKeys(akp, b.pubkeys)

	address := kp.Address()
	if err := b.client.Transfer(address, 0, amount); err != nil {
		return err
	}

	fmt.Println("Bob: successfully locked funds")
	fmt.Println("address: ", address)
	return nil
}

func (b *bob) RedeemFunds() error {
	// call swap.Swap.Claim() w/ b.privkeys.sk, revealing Bob's secret spend key
	return nil
}
