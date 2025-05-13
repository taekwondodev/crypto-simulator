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

func (bc *Blockchain) GetBlockLocator(lastKnownHash []byte) ([][]byte, error) {
	var locator [][]byte
	step := 1

	currentHash := bc.tip
	if lastKnownHash != nil {
		currentHash = lastKnownHash
	}

	for range 10 { // Limita a 10 hop
		locator = append(locator, currentHash)
		block, err := bc.GetBlock(currentHash)
		if err != nil {
			return nil, err
		}
		if block == nil {
			break
		}

		for range step {
			currentHash = block.PreviousHash
			block, err = bc.GetBlock(currentHash)
			if err != nil {
				return nil, err
			}
			if block == nil {
				break
			}
		}
		step *= 2
	}
	return locator, nil
}

func (bc *Blockchain) GetFirstMatchingBlock(hashes [][]byte) (*block.Block, error) {
	for _, hash := range hashes {
		block, err := bc.GetBlock(hash)
		if err != nil {
			return nil, err
		}
		if block != nil {
			return block, nil
		}
	}

	return bc.GetBlockAtHeight(0)
}

func (bc *Blockchain) GetNextBlockHashes(block *block.Block, limit int) ([][]byte, error) {
	var hashes [][]byte
	currentHash := block.Hash

	for range limit {
		next, err := bc.GetBlockByPreviousHash(currentHash)
		if err != nil {
			return nil, err
		}
		if next == nil {
			break
		}
		hashes = append(hashes, next.Hash)
		currentHash = next.Hash
	}

	return hashes, nil
}
