package alice

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

	"github.com/judecoin/jude-eth-swap/judecoin"
	"github.com/judecoin/jude-eth-swap/swap-contract"
)

var _ Alice = &alice{}

const (
	keyAlice = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"
)

// Alice contains the functions that will be called by a user who owns ETH
// and wishes to swap for JUDE.
type Alice interface {
	// GenerateKeys generates Alice's judecoin spend and view keys (S_b, V_b)
	// It returns Alice's public spend key
	GenerateKeys() (*judecoin.PublicKeyPair, error)

	// SetBobKeys sets Bob's public spend key (to be stored in the contract) and Bob's
	// private view key (used to check JUDE balance before calling Ready())
	SetBobKeys(*judecoin.PublicKey, *judecoin.PrivateViewKey)

	// DeployAndLockETH deploys an instance of the Swap contract and locks `amount` ether in it.
	DeployAndLockETH(amount uint) (ethcommon.Address, error)

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

	// CreateJudecoinWallet creates Alice's judecoin wallet after Bob calls Claim().
	CreateJudecoinWallet(*judecoin.PrivateKeyPair) (judecoin.Address, error)
}

type alice struct {
	ctx    context.Context
	t0, t1 time.Time

	privkeys    *judecoin.PrivateKeyPair
	pubkeys     *judecoin.PublicKeyPair
	bobSpendKey *judecoin.PublicKey
	bobViewKey  *judecoin.PrivateViewKey
	client      judecoin.Client

	contract   *swap.Swap
	ethPrivKey *ecdsa.PrivateKey
	ethClient  *ethclient.Client
	auth       *bind.TransactOpts
}

// NewAlice returns a new instance of Alice.
// It accepts an endpoint to a judecoin-wallet-rpc instance where Alice will generate
// the account in which the JUDE will be deposited.
func NewAlice(judecoinEndpoint, ethEndpoint, ethPrivKey string) (*alice, error) {
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

	return &alice{
		ctx:        context.Background(), // TODO: add cancel
		ethPrivKey: pk,
		ethClient:  ec,
		client:     judecoin.NewClient(judecoinEndpoint),
		auth:       auth,
	}, nil
}

func (a *alice) GenerateKeys() (*judecoin.PublicKeyPair, error) {
	var err error
	a.privkeys, err = judecoin.GenerateKeys()
	if err != nil {
		return nil, err
	}

	a.pubkeys = a.privkeys.PublicKeyPair()
	return a.pubkeys, nil
}

func (a *alice) SetBobKeys(sk *judecoin.PublicKey, vk *judecoin.PrivateViewKey) {
	a.bobSpendKey = sk
	a.bobViewKey = vk
}

func (a *alice) DeployAndLockETH(amount uint) (ethcommon.Address, error) {

	pkAlice := a.pubkeys.SpendKey().Bytes()
	pkBob := a.bobSpendKey.Bytes()

	var pka, pkb [32]byte
	copy(pka[:], pkAlice)
	copy(pkb[:], pkBob)

	address, _, swap, err := swap.DeploySwap(a.auth, a.ethClient, pka, pkb)
	if err != nil {
		// return nil, err
	}

	return address, nil
}

func (a *alice) Ready() error {
	txOpts := &bind.TransactOpts{
		From:   a.auth.From,
		Signer: a.auth.Signer,
	}
	_, err := a.contract.SetReady(txOpts)
	return err
}

func (a *alice) WatchForClaim() (<-chan *judecoin.PrivateKeyPair, error) {
	watchOpts := &bind.WatchOpts{
		Context: a.ctx,
	}

	out := make(chan *judecoin.PrivateKeyPair)
	ch := make(chan *swap.SwapClaimed)
	defer close(out)

	// watch for Refund() event on chain, calculate unlock key as result
	sub, err := a.contract.WatchClaimed(watchOpts, ch)
	if err != nil {
		return nil, err
	}

	defer sub.Unsubscribe()

	go func() {
		for {
			select {
			case claim := <-ch:
				fmt.Println("got claim", claim)
				if claim == nil || claim.S == nil {
					continue
				}

				// got Bob's secret
				sbBytes := claim.S.Bytes()
				var sb [32]byte
				copy(sb[:], sbBytes)

				skB, err := judecoin.NewPrivateSpendKey(sb[:])
				if err != nil {
					fmt.Printf("failed to convert Bob's secret into a key: %w", err)
					return
				}

				vkA, err := skB.View()
				if err != nil {
					fmt.Printf("failed to get view key from Bob's secret spend key: %w", err)
					return
				}

				skAB := judecoin.SumPrivateSpendKeys(skB, a.privkeys.SpendKey())
				vkAB := judecoin.SumPrivateViewKeys(vkA, a.privkeys.ViewKey())
				kpAB := judecoin.NewPrivateKeyPair(skAB, vkAB)

				out <- kpAB
				return
			case <-a.ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

func (a *alice) Refund() error {
	txOpts := &bind.TransactOpts{
		From:   a.auth.From,
		Signer: a.auth.Signer,
	}
	secret := a.privkeys.Bytes()
	s := big.NewInt(0).SetBytes(secret)
	_, err := a.contract.Refund(txOpts, s)
	return err
}

func (a *alice) CreateJudecoinWallet(kpB *judecoin.PrivateKeyPair) (judecoin.Address, error) {
	// got Bob's secret
	skAB := judecoin.SumPrivateSpendKeys(kpB.SpendKey(), a.privkeys.SpendKey())
	vkAB := judecoin.SumPrivateViewKeys(kpB.ViewKey(), a.privkeys.ViewKey())
	kpAB := judecoin.NewPrivateKeyPair(skAB, vkAB)

	if err := a.client.GenerateFromKeys(kpAB, "alice-swap-wallet", ""); err != nil {
		return "", err
	}

	return kpAB.Address(), nil
}
