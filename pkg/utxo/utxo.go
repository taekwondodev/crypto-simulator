package utxo

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
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

func (in *TxInput) Serialize() ([]byte, error) {
	var buffer bytes.Buffer

	binary.Write(&buffer, binary.LittleEndian, int32(len(in.TxID)))
	buffer.Write(in.TxID)

	binary.Write(&buffer, binary.LittleEndian, int32(in.OutIndex))

	binary.Write(&buffer, binary.LittleEndian, int32(len(in.Signature)))
	buffer.Write(in.Signature)

	binary.Write(&buffer, binary.LittleEndian, int32(len(in.PubKey)))
	buffer.Write(in.PubKey)

	return buffer.Bytes(), nil
}

func (u *UTXO) Serialize() ([]byte, error) {
	var buffer bytes.Buffer

	binary.Write(&buffer, binary.LittleEndian, int32(len(u.TxID)))
	buffer.Write(u.TxID)

	binary.Write(&buffer, binary.LittleEndian, int32(u.Index))

	binary.Write(&buffer, binary.LittleEndian, int32(u.Output.Value))
	binary.Write(&buffer, binary.LittleEndian, int32(len(u.Output.PubKeyHash)))
	buffer.Write(u.Output.PubKeyHash)

	return buffer.Bytes(), nil
}

func (out *TxOutput) Serialize() ([]byte, error) {
	var buffer bytes.Buffer

	binary.Write(&buffer, binary.LittleEndian, int32(out.Value))

	binary.Write(&buffer, binary.LittleEndian, int32(len(out.PubKeyHash)))
	buffer.Write(out.PubKeyHash)

	return buffer.Bytes(), nil
}

func Deserialize(data []byte) (*UTXO, error) {
	buffer := bytes.NewReader(data)
	utxo := &UTXO{}

	var txIDLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &txIDLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize TxID length: %w", err)
	}
	utxo.TxID = make([]byte, txIDLen)
	if _, err := io.ReadFull(buffer, utxo.TxID); err != nil {
		return nil, fmt.Errorf("failed to deserialize TxID: %w", err)
	}

	var index int32
	if err := binary.Read(buffer, binary.LittleEndian, &index); err != nil {
		return nil, fmt.Errorf("failed to deserialize Index: %w", err)
	}
	utxo.Index = int(index)

	var value int32
	if err := binary.Read(buffer, binary.LittleEndian, &value); err != nil {
		return nil, fmt.Errorf("failed to deserialize Output value: %w", err)
	}
	utxo.Output.Value = int(value)

	var pubKeyHashLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &pubKeyHashLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize PubKeyHash length: %w", err)
	}
	utxo.Output.PubKeyHash = make([]byte, pubKeyHashLen)
	if _, err := io.ReadFull(buffer, utxo.Output.PubKeyHash); err != nil {
		return nil, fmt.Errorf("failed to deserialize PubKeyHash: %w", err)
	}

	return utxo, nil
}

func DeserializeTxInput(data []byte) (*TxInput, error) {
	buffer := bytes.NewReader(data)
	in := &TxInput{}

	var txIDLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &txIDLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize TxID length: %w", err)
	}
	in.TxID = make([]byte, txIDLen)
	if _, err := io.ReadFull(buffer, in.TxID); err != nil {
		return nil, fmt.Errorf("failed to deserialize TxID: %w", err)
	}

	var outIndex int32
	if err := binary.Read(buffer, binary.LittleEndian, &outIndex); err != nil {
		return nil, fmt.Errorf("failed to deserialize OutIndex: %w", err)
	}
	in.OutIndex = int(outIndex)

	var sigLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &sigLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize signature length: %w", err)
	}
	in.Signature = make([]byte, sigLen)
	if _, err := io.ReadFull(buffer, in.Signature); err != nil {
		return nil, fmt.Errorf("failed to deserialize signature: %w", err)
	}
	var pubKeyLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &pubKeyLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize pubKey length: %w", err)
	}
	in.PubKey = make([]byte, pubKeyLen)
	if _, err := io.ReadFull(buffer, in.PubKey); err != nil {
		return nil, fmt.Errorf("failed to deserialize pubKey: %w", err)
	}

	return in, nil
}

func DeserializeTxOutput(data []byte) (*TxOutput, error) {
	buffer := bytes.NewReader(data)
	out := &TxOutput{}

	var value int32
	if err := binary.Read(buffer, binary.LittleEndian, &value); err != nil {
		return nil, fmt.Errorf("failed to deserialize Value: %w", err)
	}
	out.Value = int(value)

	var pubKeyHashLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &pubKeyHashLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize PubKeyHash length: %w", err)
	}
	out.PubKeyHash = make([]byte, pubKeyHashLen)
	if _, err := io.ReadFull(buffer, out.PubKeyHash); err != nil {
		return nil, fmt.Errorf("failed to deserialize PubKeyHash: %w", err)
	}

	return out, nil
}
