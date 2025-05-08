package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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

func (w *Wallet) GetAddress() string {
	hash := sha256.Sum256(w.Address)
	return hex.EncodeToString(hash[:])
}
