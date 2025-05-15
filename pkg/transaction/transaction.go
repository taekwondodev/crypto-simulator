package transaction

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
)

type Transaction struct {
	ID      []byte
	Inputs  []utxo.TxInput
	Outputs []utxo.TxOutput
}

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

	buffer.Write(tx.ID)

	binary.Write(&buffer, binary.LittleEndian, int32(len(tx.Inputs)))
	for _, input := range tx.Inputs {
		inputBytes, err := input.Serialize()
		if err != nil {
			return nil, err
		}
		binary.Write(&buffer, binary.LittleEndian, int32(len(inputBytes)))
		buffer.Write(inputBytes)
	}

	binary.Write(&buffer, binary.LittleEndian, int32(len(tx.Outputs)))
	for _, output := range tx.Outputs {
		outputBytes, err := output.Serialize()
		if err != nil {
			return nil, err
		}
		binary.Write(&buffer, binary.LittleEndian, int32(len(outputBytes)))
		buffer.Write(outputBytes)
	}

	return buffer.Bytes(), nil
}

func Deserialize(data []byte) (*Transaction, error) {
	buffer := bytes.NewReader(data)
	tx := &Transaction{}

	tx.ID = make([]byte, 32)
	if _, err := io.ReadFull(buffer, tx.ID); err != nil {
		return nil, fmt.Errorf("failed to deserialize ID: %w", err)
	}

	inputs, err := deserializeInputs(buffer)
	if err != nil {
		return nil, err
	}
	tx.Inputs = inputs

	outputs, err := deserializeOutputs(buffer)
	if err != nil {
		return nil, err
	}
	tx.Outputs = outputs

	return tx, nil
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

func deserializeInputs(buffer *bytes.Reader) ([]utxo.TxInput, error) {
	var inputCount int32
	if err := binary.Read(buffer, binary.LittleEndian, &inputCount); err != nil {
		return nil, fmt.Errorf("failed to deserialize input count: %w", err)
	}

	inputs := make([]utxo.TxInput, inputCount)
	for i := int32(0); i < inputCount; i++ {
		var inputBytesLen int32
		if err := binary.Read(buffer, binary.LittleEndian, &inputBytesLen); err != nil {
			return nil, fmt.Errorf("failed to deserialize input bytes length: %w", err)
		}

		inputBytes := make([]byte, inputBytesLen)
		if _, err := io.ReadFull(buffer, inputBytes); err != nil {
			return nil, fmt.Errorf("failed to deserialize input bytes: %w", err)
		}

		input, err := utxo.DeserializeTxInput(inputBytes)
		if err != nil {
			return nil, err
		}
		inputs[i] = *input
	}

	return inputs, nil
}

func deserializeOutputs(buffer *bytes.Reader) ([]utxo.TxOutput, error) {
	var outputCount int32
	if err := binary.Read(buffer, binary.LittleEndian, &outputCount); err != nil {
		return nil, fmt.Errorf("failed to deserialize output count: %w", err)
	}

	outputs := make([]utxo.TxOutput, outputCount)
	for i := int32(0); i < outputCount; i++ {
		var outputBytesLen int32
		if err := binary.Read(buffer, binary.LittleEndian, &outputBytesLen); err != nil {
			return nil, fmt.Errorf("failed to deserialize output bytes length: %w", err)
		}

		outputBytes := make([]byte, outputBytesLen)
		if _, err := io.ReadFull(buffer, outputBytes); err != nil {
			return nil, fmt.Errorf("failed to deserialize output bytes: %w", err)
		}

		output, err := utxo.DeserializeTxOutput(outputBytes)
		if err != nil {
			return nil, err
		}
		outputs[i] = *output
	}

	return outputs, nil
}
