package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"math/big"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	Address    []byte
}

type WalletData struct {
	PrivateKeyD []byte
	PrivateKeyX []byte
	PrivateKeyY []byte
	Address     []byte
}

func NewWallet() *Wallet {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	publicKey := append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...)
	return &Wallet{privateKey, publicKey}
}

func NewWalletFromKeys(privateKeyD, publicKeyX, publicKeyY, address []byte) *Wallet {
	privateKey := new(ecdsa.PrivateKey)
	privateKey.PublicKey.Curve = elliptic.P256()

	privateKey.D = new(big.Int).SetBytes(privateKeyD)

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

func (w *Wallet) Serialize() ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	walletData := WalletData{
		PrivateKeyD: w.PrivateKey.D.Bytes(),
		PrivateKeyX: w.PrivateKey.PublicKey.X.Bytes(),
		PrivateKeyY: w.PrivateKey.PublicKey.Y.Bytes(),
		Address:     w.Address,
	}

	if err := encoder.Encode(walletData); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func DeserializeWallet(data []byte) (*Wallet, error) {
	var walletData WalletData

	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&walletData); err != nil {
		return nil, err
	}

	return NewWalletFromKeys(
		walletData.PrivateKeyD,
		walletData.PrivateKeyX,
		walletData.PrivateKeyY,
		walletData.Address,
	), nil
}
