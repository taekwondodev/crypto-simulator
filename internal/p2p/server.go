package p2p

import (
	"encoding/hex"
	"errors"
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
		MsgBlock:     func(m *Message, c net.Conn) error { return n.handleBlock(m, c) },
		MsgGetBlocks: func(m *Message, c net.Conn) error { return n.handleGetBlocks(m, c) },
		MsgInv:       func(m *Message, c net.Conn) error { return n.handleInventory(m, c) },
		MsgGetData:   func(m *Message, c net.Conn) error { return n.handleGetData(m, c) },
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
	return sendPongMessage(conn, n.Address)
}

func (n *Node) handlePong() error {
	// Ping/pong is just for keepalive, nothing to do
	return nil
}

func (n *Node) handleBlock(msg *Message, conn net.Conn) error {
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
		return n.handleSync(conn)
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
		txID := hex.EncodeToString(tx.ID)
		if n.mempool.Get(txID) != nil {
			return nil
		}
		n.mempool.Add(tx)
		n.Broadcast(msg)
	}

	return nil
}

func (n *Node) handleGetBlocks(msg *Message, conn net.Conn) error {
	locator, err := deserializeHashes(msg.Payload)
	if err != nil {
		return err
	}

	startBlock, err := n.blockchain.GetFirstMatchingBlock(locator)
	if err != nil {
		return err
	}
	nextBlocks, err := n.blockchain.GetNextBlockHashes(startBlock, 10)
	if err != nil {
		return err
	}

	return sendInvMessage(conn, nextBlocks, n.Address)
}

func (n *Node) handleInventory(msg *Message, conn net.Conn) error {
	hashes, err := deserializeHashes(msg.Payload)
	if err != nil {
		return err
	}

	unknownHashes, err := n.collectUnknownBlockHashes(hashes)
	if err != nil {
		return err
	}

	if len(unknownHashes) > 0 {
		return sendGetDataMessage(conn, unknownHashes, n.Address)
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

func (n *Node) handleSync(conn net.Conn) error {
	locator, err := n.blockchain.GetBlockLocator(nil)
	if err != nil {
		return err
	}
	return sendGetBlocksMessage(conn, locator, n.Address)
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
		return sendBlockMessage(conn, serialize, n.Address)
	}

	txID := hex.EncodeToString(hash)
	tx := n.mempool.Get(txID)
	if tx != nil {
		log.Printf("Sending requested transaction from mempool: %s", txID)
		serialize, err := tx.Serialize()
		if err != nil {
			return err
		}
		return sendTxMessage(conn, serialize, n.Address)
	}

	// If not in mempool, check if it's a transaction in the blockchain
	tx = n.blockchain.FindTransaction(hash)
	if tx != nil {
		log.Printf("Sending requested transaction from blockchain: %s", txID)
		serialize, err := tx.Serialize()
		if err != nil {
			return err
		}
		return sendTxMessage(conn, serialize, n.Address)
	}

	log.Printf("Requested data not found for hash: %x", hash)
	return nil
}
