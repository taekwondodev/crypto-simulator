package p2p

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"time"
)

const (
	MsgVersion   = 0x01
	MsgVerAck    = 0x02
	MsgAddr      = 0x03
	MsgGetAddr   = 0x04
	MsgInv       = 0x05
	MsgGetData   = 0x06
	MsgGetBlocks = 0x07
	MsgBlock     = 0x08
	MsgTx        = 0x09
	MsgPing      = 0x0A
	MsgPong      = 0x0B
)

type Message struct {
	Version   uint32
	Type      uint8
	Timestamp int64
	Payload   []byte
}

func NewTxMessage(txData []byte) *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgTx,
		Timestamp: time.Now().Unix(),
		Payload:   txData,
	}
}

func NewBlockMessage(blockData []byte) *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgBlock,
		Timestamp: time.Now().Unix(),
		Payload:   blockData,
	}
}

func NewGetBlocksMessage(lastKnownHash []byte) *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgGetBlocks,
		Timestamp: time.Now().Unix(),
		Payload:   lastKnownHash,
	}
}

// Serialize converts a message to bytes
func (m *Message) Serialize() ([]byte, error) {
	var buffer bytes.Buffer

	// Write header fields
	if err := binary.Write(&buffer, binary.LittleEndian, m.Version); err != nil {
		return nil, err
	}

	if err := binary.Write(&buffer, binary.LittleEndian, m.Type); err != nil {
		return nil, err
	}

	if err := binary.Write(&buffer, binary.LittleEndian, m.Timestamp); err != nil {
		return nil, err
	}

	// Write payload length and payload
	payloadLen := uint32(len(m.Payload))
	if err := binary.Write(&buffer, binary.LittleEndian, payloadLen); err != nil {
		return nil, err
	}

	if payloadLen > 0 {
		if _, err := buffer.Write(m.Payload); err != nil {
			return nil, err
		}
	}

	return buffer.Bytes(), nil
}

// DeserializeMessage converts bytes to a message
func DeserializeMessage(data []byte) (*Message, error) {
	if len(data) < 13 { // Minimum size: version(4) + type(1) + timestamp(8)
		return nil, errors.New("message data too short")
	}

	buffer := bytes.NewReader(data)
	var msg Message

	if err := binary.Read(buffer, binary.LittleEndian, &msg.Version); err != nil {
		return nil, err
	}

	if err := binary.Read(buffer, binary.LittleEndian, &msg.Type); err != nil {
		return nil, err
	}

	if err := binary.Read(buffer, binary.LittleEndian, &msg.Timestamp); err != nil {
		return nil, err
	}

	// Read payload length
	var payloadLen uint32
	if err := binary.Read(buffer, binary.LittleEndian, &payloadLen); err != nil {
		return nil, err
	}

	// Read payload if exists
	if payloadLen > 0 {
		msg.Payload = make([]byte, payloadLen)
		if _, err := buffer.Read(msg.Payload); err != nil {
			return nil, err
		}
	}

	return &msg, nil
}

// SerializeHashes serializes a slice of hashes
func serializeHashes(hashes [][]byte) []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	encoder.Encode(len(hashes))

	for _, hash := range hashes {
		encoder.Encode(len(hash))
		encoder.Encode(hash)
	}

	return buffer.Bytes()
}

// DeserializeHashes deserializes a slice of hashes
func deserializeHashes(data []byte) ([][]byte, error) {
	buffer := bytes.NewReader(data)
	decoder := gob.NewDecoder(buffer)

	var count int
	if err := decoder.Decode(&count); err != nil {
		return nil, err
	}

	hashes := make([][]byte, count)
	for i := 0; i < count; i++ {
		var hashLen int
		if err := decoder.Decode(&hashLen); err != nil {
			return nil, err
		}

		hash := make([]byte, hashLen)
		if err := decoder.Decode(&hash); err != nil {
			return nil, err
		}

		hashes[i] = hash
	}

	return hashes, nil
}
