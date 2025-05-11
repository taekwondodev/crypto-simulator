package transaction

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"

	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
)

type Transaction struct {
	ID      []byte
	Inputs  []utxo.TxInput
	Outputs []utxo.TxOutput
}

// First transaction in the blockchain, called the coinbase transaction
func NewCoinBaseTx(to string, value int) (*Transaction, error) {
	tx := &Transaction{
		Outputs: []utxo.TxOutput{
			{Value: value, PubKeyHash: []byte(to)},
		},
	}

	hash, err := tx.hash()
	if err != nil {
		return nil, err
	}
	tx.ID = hash
	return tx, nil
}

func New(inputs []utxo.TxInput, outputs []utxo.TxOutput) (*Transaction, error) {
	tx := &Transaction{
		Inputs:  inputs,
		Outputs: outputs,
	}

	hash, err := tx.hash()
	if err != nil {
		return nil, err
	}
	tx.ID = hash
	return tx, nil
}

func (tx *Transaction) hash() ([]byte, error) {
	var hash [32]byte
	txCopy := *tx
	txCopy.ID = []byte{}
	data, err := txCopy.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}
	hash = sha256.Sum256(data)
	return hash[:], nil
}

func (tx *Transaction) IsCoinBase() bool {
	return len(tx.Inputs) == 0
}

func (tx *Transaction) Serialize() ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}
	return buffer.Bytes(), nil
}

func Deserialize(data []byte) (*Transaction, error) {
	var tx Transaction
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&tx)
	if err != nil {
		return nil, fmt.Errorf("Failed to deserialize transaction: %w", err)
	}
	return &tx, nil
}

func (tx *Transaction) Print(indent string) {
	fmt.Printf("%sTransaction: %x\n", indent, tx.ID)
	fmt.Printf("%s  Inputs: %d\n", indent, len(tx.Inputs))
	for i, input := range tx.Inputs {
		fmt.Printf("%s  Input %d: %x:%d\n", indent, i, input.TxID, input.OutIndex)
	}
	fmt.Printf("%s  Outputs: %d\n", indent, len(tx.Outputs))
	for i, output := range tx.Outputs {
		fmt.Printf("%s    Output %d: %d coins\n", indent, i, output.Value)
	}
}
