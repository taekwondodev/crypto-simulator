package p2p

import (
	"bytes"
	"encoding/hex"
	"errors"
	"log"
	"net"

	"github.com/taekwondodev/crypto-simulator/internal/blockchain"
	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
)

func (n *Node) handleMessage(msg *Message, conn net.Conn) error {
	handlers := map[uint8]func(*Message, net.Conn) error{
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
	hashes, err := deserializeHashes(msg.Payload)
	if err != nil {
		return err
	}
	var genesisFromPeer []byte
	for _, hash := range hashes {
		genesisFromPeer = hash
		break
	}

	actualGenesis, err := n.blockchain.GetBlockAtHeight(0)
	if err != nil {
		return err
	}
	if !bytes.Equal(genesisFromPeer, actualGenesis.Hash) {
		return errors.New("Peer has a different genesis block. Cannot synchronize.")
	}
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
		log.Printf("Received block with unknown previous hash")
		return n.handleSync(conn)
	}

	if err := n.handleFork(newBlock, previousBlock); err != nil {
		return err
	}

	if err := n.blockchain.AddBlock(newBlock); err != nil {
		conn.Close()
		n.removePeer(conn.RemoteAddr().String())
		return err
	}

	log.Printf("Added new block to the blockchain at height: %d\n", newBlock.Height)
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
		if n.mempool.Add(tx) {
			n.Broadcast(msg)
		}
	}

	return nil
}

func (n *Node) handleGetBlocks(msg *Message, conn net.Conn) error {
	locator, err := deserializeHashes(msg.Payload)
	if err != nil {
		return err
	}

	startBlock, err := n.blockchain.FindCommonBlock(locator)
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

	log.Printf("Blockchain has no unknown blocks")
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

func isPotentialFork(forkChain *blockchain.Blockchain, lastBlock *block.Block) bool {
	defer func() {
		forkChain.Close()
		forkChain.CleanupForkDB(forkChain)
	}()

	forkHeight, _ := forkChain.CurrentHeight()

	if forkHeight <= lastBlock.Height {
		return false
	}

	forkDifficulty, _ := forkChain.CurrentDifficulty()
	return forkDifficulty > lastBlock.Difficulty
}

func (n *Node) handleFork(new, previous *block.Block) error {
	lastBlock, err := n.blockchain.LastBlock()
	if err != nil {
		return err
	}
	if !bytes.Equal(previous.Hash, lastBlock.Hash) {
		forkChain, err := n.blockchain.GetForkChain(new.Hash)
		if err != nil {
			return err
		}
		if isPotentialFork(forkChain, lastBlock) {
			txsToRestore, err := n.blockchain.ReorganizeChain(lastBlock, forkChain)
			if err != nil {
				return err
			}
			for _, tx := range txsToRestore {
				n.mempool.Add(tx)
			}

			log.Printf("Chain reorganized and %d transactions restored to mempool", len(txsToRestore))
		}
	}
	return nil
}

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
		serialize, err := tx.Serialize()
		if err != nil {
			return err
		}
		return sendTxMessage(conn, serialize, n.Address)
	}

	// If not in mempool, check if it's a transaction in the blockchain
	tx = n.blockchain.FindTransaction(hash)
	if tx != nil {
		serialize, err := tx.Serialize()
		if err != nil {
			return err
		}
		return sendTxMessage(conn, serialize, n.Address)
	}

	log.Printf("Requested data not found for hash: %x", hash)
	return nil
}
