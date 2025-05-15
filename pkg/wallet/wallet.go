package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
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

	privKeyD := w.PrivateKey.D.Bytes()
	binary.Write(&buffer, binary.LittleEndian, int32(len(privKeyD)))
	buffer.Write(privKeyD)

	pubKeyX := w.PrivateKey.PublicKey.X.Bytes()
	binary.Write(&buffer, binary.LittleEndian, int32(len(pubKeyX)))
	buffer.Write(pubKeyX)

	pubKeyY := w.PrivateKey.PublicKey.Y.Bytes()
	binary.Write(&buffer, binary.LittleEndian, int32(len(pubKeyY)))
	buffer.Write(pubKeyY)

	binary.Write(&buffer, binary.LittleEndian, int32(len(w.Address)))
	buffer.Write(w.Address)

	return buffer.Bytes(), nil
}

func DeserializeWallet(data []byte) (*Wallet, error) {
	buffer := bytes.NewReader(data)

	var privKeyDLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &privKeyDLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize private key D length: %w", err)
	}
	privKeyD := make([]byte, privKeyDLen)
	if _, err := io.ReadFull(buffer, privKeyD); err != nil {
		return nil, fmt.Errorf("failed to deserialize private key D: %w", err)
	}

	var pubKeyXLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &pubKeyXLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize public key X length: %w", err)
	}
	pubKeyX := make([]byte, pubKeyXLen)
	if _, err := io.ReadFull(buffer, pubKeyX); err != nil {
		return nil, fmt.Errorf("failed to deserialize public key X: %w", err)
	}

	var pubKeyYLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &pubKeyYLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize public key Y length: %w", err)
	}
	pubKeyY := make([]byte, pubKeyYLen)
	if _, err := io.ReadFull(buffer, pubKeyY); err != nil {
		return nil, fmt.Errorf("failed to deserialize public key Y: %w", err)
	}

	var addressLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &addressLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize address length: %w", err)
	}
	address := make([]byte, addressLen)
	if _, err := io.ReadFull(buffer, address); err != nil {
		return nil, fmt.Errorf("failed to deserialize address: %w", err)
	}

	return NewWalletFromKeys(privKeyD, pubKeyX, pubKeyY, address), nil
}
