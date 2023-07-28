package bob

import (
	"fmt"

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
	// It accepts an instance of the Swap contract (as deployed by Alice)
	RedeemFunds() error
}

type bob struct {
	privkeys *judecoin.PrivateKeyPair
	pubkeys  *judecoin.PublicKeyPair
	client judecoin.Client
	contract *swap.Swap
}

func NewBob() *bob {
	return &bob{}
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

func (b *bob) WatchForRefund() (<-chan *judecoin.PrivateKeyPair, error) {
	return nil, nil
}

func (b *bob) LockFunds(akp *judecoin.PublicKeyPair, amount uint) error {
	sk := judecoin.Sum(akp.SpendKey(), b.pubkeys.SpendKey())
	vk := judecoin.Sum(akp.ViewKey(), b.pubkeys.ViewKey())

	address := judecoin.NewPublicKeyPair(sk, vk).Address()
	if err := b.client.Transfer(address, amount); err != nil {
		return err
	}

	fmt.Println("Bob: successfully locked funds")
	fmt.Println("address: ", address)
	return nil
}

func (b *bob) RedeemFunds() error {
	return nil
}