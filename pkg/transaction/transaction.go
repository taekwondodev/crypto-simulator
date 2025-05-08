package transaction

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"
)

type Transaction struct {
	ID        []byte
	Inputs    []TxInput
	Outputs   []TxOutput
	Signature []byte
}

type TxInput struct {
	TxID      []byte
	OutIndex  int
	Signature []byte
	PubKey    []byte
}

type TxOutput struct {
	Value      int
	PubKeyHash []byte
}

func NewTransaction(inputs []TxInput, outputs []TxOutput, privateKey *ecdsa.PrivateKey) *Transaction {
	t := &Transaction{
		ID:        []byte{},
		Inputs:    inputs,
		Outputs:   outputs,
		Signature: []byte{},
	}

	t.sign(privateKey)

	return t
}

func (tx *Transaction) sign(privateKey *ecdsa.PrivateKey) {
	dataHash := sha256.Sum256(tx.serializeForSigning())
	r, s, _ := ecdsa.Sign(rand.Reader, privateKey, dataHash[:])
	tx.Signature = append(r.Bytes(), s.Bytes()...)
}

func (tx *Transaction) serializeForSigning() []byte {
	var data []byte
	for _, input := range tx.Inputs {
		data = append(data, input.TxID...)
		data = append(data, byte(input.OutIndex))
	}
	for _, output := range tx.Outputs {
		data = append(data, byte(output.Value))
		data = append(data, output.PubKeyHash...)
	}
	return data
}

func (tx *Transaction) Verify() bool {
	dataHash := sha256.Sum256(tx.serializeForSigning())
	r := new(big.Int).SetBytes(tx.Signature[:len(tx.Signature)/2])
	s := new(big.Int).SetBytes(tx.Signature[len(tx.Signature)/2:])

	pubKey := tx.Inputs[0].PubKey
	x := new(big.Int).SetBytes(pubKey[:len(pubKey)/2])
	y := new(big.Int).SetBytes(pubKey[len(pubKey)/2:])
	publicKey := ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}

	return ecdsa.Verify(&publicKey, dataHash[:], r, s)
}
