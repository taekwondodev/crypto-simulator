package utxo

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"log"
)

type UTXO struct {
	TxID   []byte
	Index  int
	Output TxOutput
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

func (in *TxInput) Sign(privateKey *ecdsa.PrivateKey) {
	dataToSign := fmt.Sprintf("%x:%d", in.TxID, in.OutIndex)
	dataHash := sha256.Sum256([]byte(dataToSign))
	r, s, _ := ecdsa.Sign(rand.Reader, privateKey, dataHash[:])
	in.Signature = append(r.Bytes(), s.Bytes()...)
}

func (u *UTXO) Serialize() []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(u)
	if err != nil {
		log.Panic(err)
	}
	return buffer.Bytes()
}

func Deserialize(data []byte) *UTXO {
	var utxo UTXO
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&utxo)
	if err != nil {
		log.Panic(err)
	}
	return &utxo
}
