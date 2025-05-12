package p2p

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
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

func NewVersionMessage(senderAddr string) *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgVersion,
		Timestamp: time.Now().Unix(),
		Payload:   []byte(senderAddr),
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

func NewGetBlocksMessage(lastKnownHash []byte) *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgGetBlocks,
		Timestamp: time.Now().Unix(),
		Payload:   lastKnownHash,
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

func NewAddrMessage(addresses []byte) *Message {
	return &Message{
		Version:   0x01,
		Type:      MsgAddr,
		Timestamp: time.Now().Unix(),
		Payload:   addresses,
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
	encoder := gob.NewEncoder(&buffer)
	encoder.Encode(len(hashes))

	for _, hash := range hashes {
		encoder.Encode(len(hash))
		encoder.Encode(hash)
	}

	return buffer.Bytes()
}

func deserializeHashes(data []byte) ([][]byte, error) {
	buffer := bytes.NewReader(data)
	decoder := gob.NewDecoder(buffer)

	var count int
	if err := decoder.Decode(&count); err != nil {
		return nil, err
	}

	hashes := make([][]byte, count)
	for i := range count {
		if err := deserializeHash(decoder, &hashes, i); err != nil {
			return nil, err
		}
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

func deserializeHash(decoder *gob.Decoder, hashes *[][]byte, i int) error {
	var hashLen int
	if err := decoder.Decode(&hashLen); err != nil {
		return err
	}

	hash := make([]byte, hashLen)
	if err := decoder.Decode(&hash); err != nil {
		return err
	}

	(*hashes)[i] = hash
	return nil
}
