package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	Address    []byte
}

func NewWallet() *Wallet {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	publicKey := append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...)
	return &Wallet{privateKey, publicKey}
}

func NewWalletFromKeys(privateKeyD, publicKeyX, publicKeyY, address []byte) *Wallet {
	// Reconstruct private key
	privateKey := new(ecdsa.PrivateKey)
	privateKey.PublicKey.Curve = elliptic.P256()

	// Set D value (private component)
	privateKey.D = new(big.Int).SetBytes(privateKeyD)

	// Set X and Y values (public components)
	privateKey.PublicKey.X = new(big.Int).SetBytes(publicKeyX)
	privateKey.PublicKey.Y = new(big.Int).SetBytes(publicKeyY)

	return &Wallet{
		PrivateKey: privateKey,
		Address:    address,
	}
}

func (w *Wallet) GetAddress() string {
	hash := sha256.Sum256(w.Address)
	return hex.EncodeToString(hash[:])
}
