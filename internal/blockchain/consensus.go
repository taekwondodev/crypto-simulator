package blockchain

import (
	"bytes"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"go.etcd.io/bbolt"
)

func (bc *Blockchain) FindTransaction(id []byte) *transaction.Transaction {
	var t *transaction.Transaction

	bc.Db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			if bytes.Equal(k, []byte("l")) {
				continue
			}

			block, err := block.Deserialize(v)
			if err != nil {
				return err
			}

			for _, transaction := range block.Transactions {
				if bytes.Equal(transaction.ID, id) {
					t = transaction
					return nil
				}
			}
		}

		return nil
	})

	return t
}

func (bc *Blockchain) GetChainFrom(startHash []byte) []*block.Block {
	var chain []*block.Block
	currentHash := startHash

	bc.Db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		for currentHash != nil {
			blk, err := block.Deserialize(b.Get(currentHash))
			if err != nil {
				return err
			}
			chain = append(chain, blk)
			currentHash = blk.PreviousHash
		}
		return nil
	})
	return reverseChain(chain)
}

func (bc *Blockchain) GetBlockLocator(lastKnownHash []byte) ([]*block.Block, [][]byte, error) {
	var blocks []*block.Block
	var hashes [][]byte
	found := false

	err := bc.Db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			if found && len(blocks) < 10 {
				blk, err := block.Deserialize(v)
				if err != nil {
					return err
				}
				blocks = append(blocks, blk)
			}

			if bytes.Equal(k, lastKnownHash) {
				found = true
			}
		}
		return nil
	})

	for _, blk := range blocks {
		hashes = append(hashes, blk.Hash)
	}

	return blocks, hashes, err
}
