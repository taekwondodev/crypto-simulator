package p2p

import (
	"bytes"
	"encoding/gob"
	"log"
	"net"
	"time"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
)

func (n *Node) handleVersion(msg *Message, conn net.Conn) error {
	// Extract peer address from version message if present
	peerAddr := string(msg.Payload)
	if peerAddr != "" {
		log.Printf("Received version message from %s", peerAddr)
	}

	// We've already responded with verack in the handshake
	return nil
}

func (n *Node) handleVerAck(msg *Message) error {
	// Handshake complete, nothing to do
	return nil
}

func (n *Node) handlePing(msg *Message, conn net.Conn) error {
	// Send pong in response to ping
	pongMsg := &Message{
		Version:   0x01,
		Type:      MsgPong,
		Timestamp: time.Now().Unix(),
	}

	data, err := pongMsg.Serialize()
	if err != nil {
		return err
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = conn.Write(data)
	return err
}

func (n *Node) handlePong(msg *Message) error {
	// Ping/pong is just for keepalive, nothing to do
	return nil
}

func (n *Node) handleBlock(msg *Message) error {
	// Deserialize the block
	newBlock := block.Deserialize(msg.Payload)

	// Get current blockchain tip
	lastBlock := n.blockchain.LastBlock()

	// Scenario 1: This block extends our current chain
	if bytes.Equal(newBlock.PreviousHash, lastBlock.Hash) {
		if newBlock.Validate(lastBlock) {
			n.blockchain.AddBlock([]*transaction.Transaction{}) // This isn't right but we need to fix elsewhere
			log.Printf("Added new block: %x at height %d", newBlock.Hash, newBlock.Height)
		} else {
			log.Printf("Received invalid block: %x", newBlock.Hash)
		}
		return nil
	}
	// Scenario 2: This could be a fork, check if it's longer
	currentHeight := n.blockchain.CurrentHeight()
	if newBlock.Height > currentHeight {
		// Fetch the full candidate chain
		candidateChain, err := n.fetchCandidateChain(newBlock)
		if err != nil {
			log.Printf("Error fetching candidate chain: %v", err)
			return err
		}

		if n.blockchain.IsValidChain(candidateChain) {
			err := n.blockchain.ReorganizeChain(candidateChain)
			if err != nil {
				log.Printf("Chain reorganization failed: %v", err)
				return err
			}
			log.Printf("Chain reorganized to new tip: %x at height %d",
				candidateChain[len(candidateChain)-1].Hash, candidateChain[len(candidateChain)-1].Height)
		}
	}

	return nil
}

func (n *Node) handleTransaction(msg *Message) error {
	tx := transaction.Deserialize(msg.Payload)

	if n.mempool.ValidateTransaction(tx) {
		n.mempool.Add(tx)
		n.Broadcast(msg) // Inoltra ad altri peer
	}

	return nil
}

func (n *Node) handleGetBlocks(msg *Message, conn net.Conn) error {
	// 1. Ottieni l'ultimo hash conosciuto dal peer
	var lastKnownHash []byte
	if len(msg.Payload) > 0 {
		lastKnownHash = msg.Payload
	}

	// Get block locator (list of hashes to help peer sync)
	locator := n.blockchain.GetBlockLocator(lastKnownHash)

	// Send inventory message with block hashes
	invMsg := &Message{
		Version:   0x01,
		Type:      MsgInv,
		Timestamp: time.Now().Unix(),
		Payload:   serializeHashes(locator),
	}

	data, err := invMsg.Serialize()
	if err != nil {
		return err
	}
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = conn.Write(data)
	return err
}

func (n *Node) handleInventory(msg *Message, conn net.Conn) error {
	// Deserialize list of hashes
	hashes, err := deserializeHashes(msg.Payload)
	if err != nil {
		return err
	}

	// For each hash that we don't have, request the data
	var unknownHashes [][]byte

	for _, hash := range hashes {
		if n.blockchain.GetBlock(hash) == nil {
			unknownHashes = append(unknownHashes, hash)
		}
	}

	if len(unknownHashes) > 0 {
		// Request unknown blocks
		getDataMsg := &Message{
			Version:   0x01,
			Type:      MsgGetData,
			Timestamp: time.Now().Unix(),
			Payload:   serializeHashes(unknownHashes),
		}

		data, err := getDataMsg.Serialize()
		if err != nil {
			return err
		}

		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_, err = conn.Write(data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) handleGetData(msg *Message, conn net.Conn) error {
	// Deserialize list of requested hashes
	hashes, err := deserializeHashes(msg.Payload)
	if err != nil {
		return err
	}

	// Send each requested block or transaction
	for _, hash := range hashes {
		// Check if it's a block hash
		block := n.blockchain.GetBlock(hash)
		if block != nil {
			blockMsg := &Message{
				Version:   0x01,
				Type:      MsgBlock,
				Timestamp: time.Now().Unix(),
				Payload:   block.Serialize(),
			}

			data, err := blockMsg.Serialize()
			if err != nil {
				continue
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			conn.Write(data)
			continue
		}

		// Could implement transaction lookup here too if needed
	}
	return nil
}

func (n *Node) handleAddress(msg *Message) error {
	// Add addresses to our peer list
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

func (n *Node) handleGetAddr(msg *Message, conn net.Conn) error {
	// Send list of known peer addresses
	n.mu.Lock()
	addresses := make([]string, 0, len(n.Peers))
	for addr := range n.Peers {
		addresses = append(addresses, addr)
	}
	n.mu.Unlock()

	// Serialize the addresses
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(addresses); err != nil {
		return err
	}

	// Create and send address message
	addrMsg := &Message{
		Version:   0x01,
		Type:      MsgAddr,
		Timestamp: time.Now().Unix(),
		Payload:   buffer.Bytes(),
	}

	data, err := addrMsg.Serialize()
	if err != nil {
		return err
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = conn.Write(data)
	return err
}

func (n *Node) fetchCandidateChain(startBlock *block.Block) ([]*block.Block, error) {
	// Create a message to request blocks
	getBlocksMsg := &Message{
		Version:   0x01,
		Type:      MsgGetBlocks,
		Timestamp: time.Now().Unix(),
		Payload:   startBlock.Hash,
	}

	// Broadcast the request
	n.Broadcast(getBlocksMsg)

	// In a real implementation, this would wait for responses and build the chain
	// For now, we'll just return the starting block as a placeholder
	return []*block.Block{startBlock}, nil
}
