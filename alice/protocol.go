package alice

import (
	"crypto/ecdsa"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/judecoin/jude-eth-swap/judecoin"
	"github.com/judecoin/jude-eth-swap/swap-contract"
)

var _ Alice = &alice{}

// Alice contains the functions that will be called by a user who owns ETH
// and wishes to swap for JUDE.
type Alice interface {
	// GenerateKeys generates Alice's judecoin spend and view keys (S_b, V_b)
	// It returns Alice's public spend key
	GenerateKeys() (*judecoin.PublicKey, error)

	// DeployAndLockETH deploys an instance of the Swap contract and locks `amount` ether in it.
	DeployAndLockETH(amount uint) (*swap.Swap, error)

	// Ready calls the Ready() method on the Swap contract, indicating to Bob he has until time t_1 to
	// call Claim(). Ready() should only be called once Alice sees Bob lock his JUDE.
	// If time t_0 has passed, there is no point of calling Ready().
	Ready() error

	// WatchForClaim watches for Bob to call Claim() on the Swap contract.
	// When Claim() is called, revealing Bob's secret s_b, the secret key corresponding
	// to (s_a + s_b) will be sent over this channel, allowing Alice to claim the JUDE it contains.
	WatchForClaim() (<-chan *judecoin.PrivateKeyPair, error)

	// Refund calls the Refund() method in the Swap contract, revealing Alice's secret
	// and returns to her the ether in the contract.
	// If time t_1 passes and Claim() has not been called, Alice should call Refund().
	Refund() error
}

type alice struct {
	t0, t1 time.Time

	privkeys *judecoin.PrivateKeyPair
	pubkeys  *judecoin.PublicKeyPair
	client   judecoin.Client

	contract    *swap.Swap
	ethPrivKey  *ecdsa.PrivateKey
	ethEndpoint string
}

// NewAlice returns a new instance of Alice.
// It accepts an endpoint to a judecoin-wallet-rpc instance where Alice will generate
// the account in which the JUDE will be deposited.
func NewAlice(judecoinEndpoint, ethEndpoint, ethPrivKey string) (*alice, error) {
	pk, err := crypto.HexToECDSA(ethPrivKey)
	if err != nil {
		return nil, err
	}

	return &alice{
		ethPrivKey:  pk,
		ethEndpoint: ethEndpoint,
		client:      judecoin.NewClient(judecoinEndpoint),
	}, nil
}

func (a *alice) GenerateKeys() (*judecoin.PublicKey, error) {
	var err error
	a.privkeys, err = judecoin.GenerateKeys()
	if err != nil {
		return nil, err
	}

	a.pubkeys = a.privkeys.PublicKeyPair()
	return a.pubkeys.SpendKey(), nil
}

func (a *alice) DeployAndLockETH(amount uint) (*swap.Swap, error) {
	return nil, nil
}

func (a *alice) Ready() error {
	return nil
}

func (a *alice) WatchForClaim() (<-chan *judecoin.PrivateKeyPair, error) {
	return nil, nil
}

func (a *alice) Refund() error {
	return nil
}
