package bob

import (
	"github.com/judecoin/atomic-swap-eth/judecoin"
)

// Bob contains the functions that will be called by a user who owns JUDE
// and wishes to swap for ETH.
type Bob interface {
	GenerateKeys() *judecoin.PublicKeyPair
}
