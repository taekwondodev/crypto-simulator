package p2p

import (
	"fmt"
	"log"

	"go.etcd.io/bbolt"
)

const peerBucket = "peers"

func (n *Node) SavePeers() {
	err := n.blockchain.Db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(peerBucket))
		if err != nil {
			return err
		}

		deleteExistingEntries(b)

		n.mu.Lock()
		defer n.mu.Unlock()

		return storePeers(b, n)
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

		return connectPeers(b, n)
	})

	if err != nil {
		log.Printf("Error loading peers: %v", err)
	}
}

func deleteExistingEntries(b *bbolt.Bucket) error {
	return b.ForEach(func(k, v []byte) error {
		return b.Delete(k)
	})
}

func storePeers(b *bbolt.Bucket, n *Node) error {
	i := 0
	for addr := range n.Peers {
		key := fmt.Appendf(nil, "peer_%d", i)
		if err := b.Put(key, []byte(addr)); err != nil {
			return err
		}
		i++
	}

	return nil
}

func connectPeers(b *bbolt.Bucket, n *Node) error {
	return b.ForEach(func(k, v []byte) error {
		addr := string(v)
		if addr != n.Address {
			go n.Connect(addr)
		}
		return nil
	})
}
