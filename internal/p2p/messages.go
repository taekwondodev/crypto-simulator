package p2p

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

const (
	MsgVersion   = 0x01
	MsgVerAck    = 0x02
	MsgInv       = 0x03
	MsgGetData   = 0x04
	MsgGetBlocks = 0x05
	MsgBlock     = 0x06
	MsgTx        = 0x07
	MsgPing      = 0x0A
	MsgPong      = 0x0B
)

type Message struct {
	Version   uint32
	Type      uint8
	Timestamp int64
	Payload   []byte
}

func NewVersionMessage(genesis [][]byte) *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgVersion,
		Timestamp: time.Now().Unix(),
		Payload:   serializeHashes(genesis),
	}
}

func NewVerAckMessage() *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgVerAck,
		Timestamp: time.Now().Unix(),
	}
}

func NewPingMessage() *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgPing,
		Timestamp: time.Now().Unix(),
	}
}

func NewPongMessage() *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgPong,
		Timestamp: time.Now().Unix(),
	}
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

func NewGetBlocksMessage(locator [][]byte) *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgGetBlocks,
		Timestamp: time.Now().Unix(),
		Payload:   serializeHashes(locator),
	}
}

func NewInvMessage(hashes [][]byte) *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgInv,
		Timestamp: time.Now().Unix(),
		Payload:   serializeHashes(hashes),
	}
}

func NewGetDataMessage(hashes [][]byte) *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgGetData,
		Timestamp: time.Now().Unix(),
		Payload:   serializeHashes(hashes),
	}
}

func (m *Message) Serialize() ([]byte, error) {
	var buffer bytes.Buffer

	if err := writeHeader(&buffer, m); err != nil {
		return nil, err
	}

	if err := writePayload(&buffer, m); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func serializeHashes(hashes [][]byte) []byte {
	var buffer bytes.Buffer

	binary.Write(&buffer, binary.LittleEndian, int32(len(hashes)))

	for _, hash := range hashes {
		binary.Write(&buffer, binary.LittleEndian, int32(len(hash)))
		buffer.Write(hash)
	}

	return buffer.Bytes()
}

func deserializeHashes(data []byte) ([][]byte, error) {
	buffer := bytes.NewReader(data)

	var count int32
	if err := binary.Read(buffer, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("failed to deserialize hash count: %w", err)
	}

	hashes := make([][]byte, count)
	for i := int32(0); i < count; i++ {
		var hashLen int32
		if err := binary.Read(buffer, binary.LittleEndian, &hashLen); err != nil {
			return nil, fmt.Errorf("failed to deserialize hash length: %w", err)
		}

		hash := make([]byte, hashLen)
		if _, err := io.ReadFull(buffer, hash); err != nil {
			return nil, fmt.Errorf("failed to deserialize hash: %w", err)
		}

		hashes[i] = hash
	}

	return hashes, nil
}

func writeHeader(buffer *bytes.Buffer, m *Message) error {
	if err := binary.Write(buffer, binary.LittleEndian, m.Version); err != nil {
		return err
	}

	if err := binary.Write(buffer, binary.LittleEndian, m.Type); err != nil {
		return err
	}

	return binary.Write(buffer, binary.LittleEndian, m.Timestamp)
}

func readHeader(buffer *bytes.Reader) (*Message, error) {
	msg := &Message{}

	if err := binary.Read(buffer, binary.LittleEndian, &msg.Version); err != nil {
		return nil, err
	}

	if err := binary.Read(buffer, binary.LittleEndian, &msg.Type); err != nil {
		return nil, err
	}

	if err := binary.Read(buffer, binary.LittleEndian, &msg.Timestamp); err != nil {
		return nil, err
	}

	return msg, nil
}

func writePayload(buffer *bytes.Buffer, m *Message) error {
	payloadLen := uint32(len(m.Payload))
	if err := binary.Write(buffer, binary.LittleEndian, payloadLen); err != nil {
		return err
	}

	if payloadLen > 0 {
		if _, err := buffer.Write(m.Payload); err != nil {
			return err
		}
	}

	return nil
}
