package judecoin

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func reverse(s []byte) []byte {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func TestPrivateKeyPairToAddress(t *testing.T) {
	skBytes := "a6e51afb9662bf2173d807ceaf88938d09a82d1ab2cea3eeb1706eeeb8b6ba03"
	pskBytes := "57edf916a28c2a0a172565468564ab1c5c217d33ea63436f7604a96aa28ec471"
	vkBytes := "42090ad9b1e3f7cecb6ff4189aa209e7d1e739bad25d9026d807380b883ed30b"
	pvkBytes := "03a57793b8fb5f87cdabcc26996393e1f700d2cb62e95e3943fdad76ff349bb6"

	sk, err := hex.DecodeString(skBytes)
	require.NoError(t, err)

	psk, err := hex.DecodeString(pskBytes)
	require.NoError(t, err)

	vk, err := hex.DecodeString(vkBytes)
	require.NoError(t, err)

	pvk, err := hex.DecodeString(pvkBytes)
	require.NoError(t, err)

	// test DecodeJudecoinBase58
	address := "J8dgPV5ktqtfDP89YYD4EwYWD19655TEP7ikbouDZX9L3Wd73CBVF4U2YL2nQnK8me8sSFdhGnqhjYVR9zP28edR3ou3cnA"
	addressBytes := DecodeJudecoinBase58(address)
	require.Equal(t, psk, addressBytes[1:33])
	require.Equal(t, pvk, addressBytes[33:65])

	// test that Address() gives the correct address bytes
	// implicitly tests that the *PrivateSpendKey.Public() and *PrivateViewKey.Public()
	// give the correct public keys
	kp, err := NewPrivateKeyPairFromBytes(sk, vk)
	require.NoError(t, err)
	require.Equal(t, addressBytes, kp.AddressBytes())
	require.Equal(t, Address(address), kp.Address())

	// check public key derivation
	require.Equal(t, pskBytes, kp.sk.Public().Hex())
	require.Equal(t, pvkBytes, kp.vk.Public().Hex())
}

func TestGeneratePrivateKeyPair(t *testing.T) {
	_, err := GenerateKeys()
	require.NoError(t, err)
}
