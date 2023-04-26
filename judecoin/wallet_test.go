package judecoin

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCallGenerateFromKeys(t *testing.T) {
	kp, err := GenerateKeys()
	require.NoError(t, err)

	r, err := rand.Int(rand.Reader, big.NewInt(999))
	require.NoError(t, err)

	err = callGenerateFromKeys(kp, fmt.Sprintf("test-wallet-%d", r), "")
	require.NoError(t, err)
}
