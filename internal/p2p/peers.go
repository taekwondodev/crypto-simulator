package p2p

import (
	"fmt"
	"log"

	"go.etcd.io/bbolt"
)

const peerBucket = "peers"

type StoredPeers struct {
	Addresses []string `json:"addresses"`
}

func (n *Node) SavePeers() {
	err := n.blockchain.Db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(peerBucket))
		if err != nil {
			return err
		}

		// Delete existing entries
		err = b.ForEach(func(k, v []byte) error {
			return b.Delete(k)
		})
		if err != nil {
			return err
		}

		// Add current peers
		n.mu.Lock()
		defer n.mu.Unlock()

		i := 0
		for addr := range n.Peers {
			// Store each peer with an index as key
			key := []byte(fmt.Sprintf("peer_%d", i))
			if err := b.Put(key, []byte(addr)); err != nil {
				return err
			}
			i++
		}

		return nil
	})
	if err != nil {
		log.Printf("Error saving peers: %v", err)
	}
}

func (n *Node) LoadPeers() {
	err := n.blockchain.Db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(peerBucket))
		if b == nil {
			return nil // No peers bucket yet
		}

		return b.ForEach(func(k, v []byte) error {
			addr := string(v)
			if addr != n.Address { // Don't connect to ourselves
				go n.Connect(addr)
			}
			return nil
		})
	})

	if err != nil {
		log.Printf("Error loading peers: %v", err)
	}
}
