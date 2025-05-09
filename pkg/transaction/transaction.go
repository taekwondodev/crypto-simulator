package transaction

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"

	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
)

type Transaction struct {
	ID      []byte
	Inputs  []utxo.TxInput
	Outputs []utxo.TxOutput
}

// First transaction in the blockchain, usually called the coinbase transaction
func NewCoinBaseTx(to string, value int) *Transaction {
	tx := &Transaction{
		Outputs: []utxo.TxOutput{
			{Value: value, PubKeyHash: []byte(to)},
		},
	}

	tx.ID = tx.hash()
	return tx
}

func New(inputs []utxo.TxInput, outputs []utxo.TxOutput) *Transaction {
	tx := &Transaction{
		Inputs:  inputs,
		Outputs: outputs,
	}

	tx.ID = tx.hash()
	return tx
}

func (tx *Transaction) hash() []byte {
	var hash [32]byte
	txCopy := *tx
	txCopy.ID = []byte{}
	hash = sha256.Sum256(txCopy.Serialize())
	return hash[:]
}

func (tx *Transaction) IsCoinBase() bool {
	return len(tx.Inputs) == 0
}

func (tx *Transaction) Serialize() []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	return buffer.Bytes()
}

func Deserialize(data []byte) *Transaction {
	var tx Transaction
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&tx)
	if err != nil {
		log.Panic("Failed to deserialize transaction:", err)
	}
	return &tx
}
