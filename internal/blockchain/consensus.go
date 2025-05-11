package blockchain

import (
	"bytes"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"go.etcd.io/bbolt"
)

func (bc *Blockchain) ReorganizeChain(newChain []*block.Block) error {
	return bc.Db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		utxoBucket := tx.Bucket([]byte(utxoBucket))

		currentChain := bc.getChainFrom(bc.tip)
		forkPoint, err := findForkPoint(currentChain, newChain)
		if err != nil {
			return err
		}

		if err := rollbackTransactions(bc, currentChain, forkPoint, utxoBucket); err != nil {
			return err
		}

		if err := applyNewChainBlocks(newChain, forkPoint, bucket, utxoBucket); err != nil {
			return err
		}

		newTip := newChain[len(newChain)-1].Hash
		bucket.Put([]byte("l"), newTip)
		bc.tip = newTip

		bc.utxoCache.Purge()
		return nil
	})
}

func (bc *Blockchain) FindTransaction(id []byte) *transaction.Transaction {
	var t *transaction.Transaction

	bc.Db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			if bytes.Equal(k, []byte("l")) {
				continue
			}

			block := block.Deserialize(v)

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

func (bc *Blockchain) getChainFrom(startHash []byte) []*block.Block {
	var chain []*block.Block
	currentHash := startHash

	bc.Db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		for currentHash != nil {
			blk := block.Deserialize(b.Get(currentHash))
			chain = append(chain, blk)
			currentHash = blk.PreviousHash
		}
		return nil
	})
	return reverseChain(chain)
}

func (bc *Blockchain) GetBlockLocator(lastKnownHash []byte) [][]byte {
	var locator [][]byte
	step := 1
	currentHash := bc.tip

	for range 10 { // Limita a 10 hop
		locator = append(locator, currentHash)
		block := bc.GetBlock(currentHash)
		if block == nil {
			break
		}

		for range step {
			currentHash = block.PreviousHash
			block = bc.GetBlock(currentHash)
			if block == nil {
				break
			}
		}
		step *= 2
	}
	return locator
}
