package p2p

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

func writeMessage(conn net.Conn, msg *Message) error {
	data, err := msg.Serialize()
	if err != nil {
		return err
	}
	return writeData(conn, data)
}

func writeData(conn net.Conn, data []byte) error {
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := conn.Write(data)
	return err
}

func readMessage(conn net.Conn, timeout time.Duration) (*Message, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))

	header := make([]byte, 13)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	msg, err := readHeader(bytes.NewReader(header))
	if err != nil {
		return nil, err
	}

	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}

	payloadLen := binary.LittleEndian.Uint32(lenBuf)
	if payloadLen > 0 {
		msg.Payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(conn, msg.Payload); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

func sendVersionMessage(conn net.Conn, address string) error {
	msg := NewVersionMessage(address)
	return writeMessage(conn, msg)
}

func sendVerAckMessage(conn net.Conn) error {
	msg := NewVerAckMessage()
	return writeMessage(conn, msg)
}

func sendPingMessage(conn net.Conn) error {
	msg := NewPingMessage()
	return writeMessage(conn, msg)
}

func sendPongMessage(conn net.Conn) error {
	msg := NewPongMessage()
	return writeMessage(conn, msg)
}

func sendTxMessage(conn net.Conn, txData []byte) error {
	msg := NewTxMessage(txData)
	return writeMessage(conn, msg)
}

func sendBlockMessage(conn net.Conn, blockData []byte) error {
	msg := NewBlockMessage(blockData)
	return writeMessage(conn, msg)
}

func sendInvMessage(conn net.Conn, hashes [][]byte) error {
	msg := NewInvMessage(hashes)
	return writeMessage(conn, msg)
}

func sendGetBlocksMessage(conn net.Conn, locator [][]byte) error {
	msg := NewGetBlocksMessage(locator)
	return writeMessage(conn, msg)
}

func sendGetDataMessage(conn net.Conn, hashes [][]byte) error {
	msg := NewGetDataMessage(hashes)
	return writeMessage(conn, msg)
}

func logMessageSent(msgType uint8, addr string) {
	log.Printf("Sent %s message to %s", messageTypeName(msgType), addr)
}

func logMessageReceived(msgType uint8, addr string) {
	log.Printf("Received %s message from %s", messageTypeName(msgType), addr)
}

func logError(context string, err error) {
	log.Printf("%s error: %v", context, err)
}

func messageTypeName(msgType uint8) string {
	switch msgType {
	case MsgVersion:
		return "VERSION"
	case MsgVerAck:
		return "VERACK"
	case MsgInv:
		return "INV"
	case MsgGetData:
		return "GETDATA"
	case MsgGetBlocks:
		return "GETBLOCKS"
	case MsgBlock:
		return "BLOCK"
	case MsgTx:
		return "TX"
	case MsgPing:
		return "PING"
	case MsgPong:
		return "PONG"
	default:
		return "UNKNOWN"
	}
}
