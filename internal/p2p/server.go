package p2p

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
)

func (n *Node) handleMessage(msg *Message, conn net.Conn) error {
	handlers := map[uint8]func(*Message, net.Conn) error{
		MsgVersion:   func(m *Message, c net.Conn) error { return n.handleVersion(m) },
		MsgVerAck:    func(m *Message, c net.Conn) error { return n.handleVerAck() },
		MsgPing:      func(m *Message, c net.Conn) error { return n.handlePing(c) },
		MsgPong:      func(m *Message, c net.Conn) error { return n.handlePong() },
		MsgTx:        func(m *Message, c net.Conn) error { return n.handleTransaction(m) },
		MsgBlock:     func(m *Message, c net.Conn) error { return n.handleBlock(m) },
		MsgGetBlocks: func(m *Message, c net.Conn) error { return n.handleGetBlocks(m, c) },
		MsgInv:       func(m *Message, c net.Conn) error { return n.handleInventory(m, c) },
		MsgGetData:   func(m *Message, c net.Conn) error { return n.handleGetData(m, c) },
		MsgAddr:      func(m *Message, c net.Conn) error { return n.handleAddress(m) },
		MsgGetAddr:   func(m *Message, c net.Conn) error { return n.handleGetAddr(c) },
	}

	handler, exists := handlers[msg.Type]
	if !exists {
		return errors.New("unknown message type")
	}

	return handler(msg, conn)
}

func (n *Node) handleVersion(msg *Message) error {
	peerAddr := string(msg.Payload)
	if peerAddr != "" {
		logMessageReceived(msg.Type, peerAddr)
	}

	// We've already responded with verack in the handshake
	return nil
}

func (n *Node) handleVerAck() error {
	// Handshake complete, nothing to do
	return nil
}

func (n *Node) handlePing(conn net.Conn) error {
	return sendPongMessage(conn)
}

func (n *Node) handlePong() error {
	// Ping/pong is just for keepalive, nothing to do
	return nil
}

func (n *Node) handleBlock(msg *Message) error {
	newBlock, err := block.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	b, err := n.blockchain.GetBlock(newBlock.Hash)
	if err != nil {
		return err
	}
	if b != nil {
		return nil
	}

	previousBlock, err := n.blockchain.GetBlock(newBlock.PreviousHash)
	if err != nil {
		return err
	}
	if previousBlock == nil {
		getBlockMsg := NewGetBlocksMessage(newBlock.PreviousHash)
		n.Broadcast(getBlockMsg)
		return fmt.Errorf("missing previous block, requesting chain")
	}

	if _, err := n.blockchain.AddBlock(newBlock.Transactions); err != nil {
		return err
	}

	return nil
}

func (n *Node) handleTransaction(msg *Message) error {
	tx, err := transaction.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	if n.mempool.ValidateTransaction(tx) {
		n.mempool.Add(tx)
		n.Broadcast(msg)
	}

	return nil
}

func (n *Node) handleGetBlocks(msg *Message, conn net.Conn) error {
	var lastKnownHash []byte
	if len(msg.Payload) > 0 {
		lastKnownHash = msg.Payload
	}

	blocks, locator, err := n.blockchain.GetBlockLocator(lastKnownHash)
	if err != nil {
		return err
	}

	if err := sendInvMessage(conn, locator); err != nil {
		return err
	}

	for _, block := range blocks {
		serializedBlock, err := block.Serialize()
		if err != nil {
			return err
		}
		if err := sendBlockMessage(conn, serializedBlock); err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) handleInventory(msg *Message, conn net.Conn) error {
	hashes, err := deserializeHashes(msg.Payload)
	if err != nil {
		return err
	}

	if len(hashes) > 0 {
		if ch, exists := n.pendingInventoryReqs[hex.EncodeToString(hashes[0])]; exists {
			ch <- hashes
			return nil
		}
	}

	unknownHashes, err := n.collectUnknownBlockHashes(hashes)
	if err != nil {
		return err
	}

	if len(unknownHashes) > 0 {
		return sendGetDataMessage(conn, unknownHashes)
	}

	return nil
}

func (n *Node) handleGetData(msg *Message, conn net.Conn) error {
	hashes, err := deserializeHashes(msg.Payload)
	if err != nil {
		return err
	}

	for _, hash := range hashes {
		if err := n.sendRequestedData(hash, conn); err != nil {
			log.Printf("Error sending requested data for hash %x: %v", hash, err)
			// Continue with other hashes even if one fails
		}
	}

	return nil
}

func (n *Node) handleAddress(msg *Message) error {
	var addresses []string
	decoder := gob.NewDecoder(bytes.NewReader(msg.Payload))
	if err := decoder.Decode(&addresses); err != nil {
		return err
	}

	for _, addr := range addresses {
		if addr != n.Address {
			go n.Connect(addr)
		}
	}

	return nil
}

func (n *Node) handleGetAddr(conn net.Conn) error {
	addresses := n.collectPeerAddresses()
	buffer := serializeAddresses(addresses)
	return sendAddrMessage(conn, buffer)
}

/*********************************************************************************************/

func (n *Node) collectUnknownBlockHashes(hashes [][]byte) ([][]byte, error) {
	var unknownHashes [][]byte

	for _, hash := range hashes {
		block, err := n.blockchain.GetBlock(hash)
		if err != nil {
			return nil, err
		}
		if block == nil {
			unknownHashes = append(unknownHashes, hash)
		}
	}

	return unknownHashes, nil
}

func (n *Node) sendRequestedData(hash []byte, conn net.Conn) error {
	// Check if it's a block hash
	blk, err := n.blockchain.GetBlock(hash)
	if err != nil {
		return err
	}
	if blk != nil {
		serialize, err := blk.Serialize()
		if err != nil {
			return err
		}
		return sendBlockMessage(conn, serialize)
	}

	txID := hex.EncodeToString(hash)
	tx := n.mempool.Get(txID)
	if tx != nil {
		log.Printf("Sending requested transaction from mempool: %s", txID)
		serialize, err := tx.Serialize()
		if err != nil {
			return err
		}
		return sendTxMessage(conn, serialize)
	}

	// If not in mempool, check if it's a transaction in the blockchain
	tx = n.blockchain.FindTransaction(hash)
	if tx != nil {
		log.Printf("Sending requested transaction from blockchain: %s", txID)
		serialize, err := tx.Serialize()
		if err != nil {
			return err
		}
		return sendTxMessage(conn, serialize)
	}

	log.Printf("Requested data not found for hash: %x", hash)
	return nil
}

func (n *Node) collectPeerAddresses() []string {
	n.mu.Lock()
	defer n.mu.Unlock()

	addresses := make([]string, 0, len(n.Peers))
	for addr := range n.Peers {
		addresses = append(addresses, addr)
	}

	return addresses
}

func serializeAddresses(addresses []string) []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	encoder.Encode(addresses)
	return buffer.Bytes()
}
