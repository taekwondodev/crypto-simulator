package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
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

func (bc *Blockchain) GetForkChain(hash []byte) (*Blockchain, error) {
	forkChain := New("fork.db")
	for {
		block, err := bc.GetBlock(hash)
		if err != nil {
			return nil, err
		}
		if block == nil {
			break
		}

		if err := forkChain.AddBlock(block); err != nil {
			return nil, err
		}

		if block.PreviousHash == nil {
			break
		}

		hash = block.PreviousHash
	}
	return forkChain, nil
}

func (bc *Blockchain) ReorganizeChain(oldTip *block.Block, newChain *Blockchain) ([]*transaction.Transaction, error) {
	commonAncestor, err := bc.findForkPoint(oldTip, newChain)
	if err != nil {
		return nil, err
	}

	oldBlocks, err := bc.collectBlocksFrom(oldTip, commonAncestor)
	if err != nil {
		return nil, err
	}

	newTip, _ := newChain.LastBlock()

	newBlocks, err := newChain.collectBlocksFrom(newTip, commonAncestor)
	if err != nil {
		return nil, err
	}

	var txToRestore []*transaction.Transaction
	err = bc.Db.Update(func(tx *bbolt.Tx) error {
		// Get necessary buckets
		blocksBucket := tx.Bucket([]byte(blocksBucket))
		utxoBucket := tx.Bucket([]byte(utxoBucket))
		heightBucket := tx.Bucket([]byte(heightIndex))

		// Rollback UTXO state - restore spent outputs and remove created outputs
		// For each block being undone (except the common ancestor)
		for _, blk := range oldBlocks {
			if bytes.Equal(blk.Hash, commonAncestor.Hash) {
				continue // Skip common ancestor
			}

			// For each transaction in the block
			for _, tx := range blk.Transactions {
				// Restore inputs (previously spent outputs)
				if !tx.IsCoinBase() {
					for _, input := range tx.Inputs {
						// Find the referenced transaction
						refTx := bc.FindTransaction(input.TxID)
						if refTx == nil {
							continue // Skip if not found
						}

						// Restore the UTXO
						utxo := &utxo.UTXO{
							TxID:   refTx.ID,
							Index:  input.OutIndex,
							Output: refTx.Outputs[input.OutIndex],
						}

						serialized, err := utxo.Serialize()
						if err != nil {
							return err
						}

						key := buildUTXOKey(refTx.ID, input.OutIndex)
						utxoBucket.Put(key, serialized)
					}
				}
				// Remove outputs created in this block
				for outIdx := range tx.Outputs {
					key := buildUTXOKey(tx.ID, outIdx)
					utxoBucket.Delete(key)
				}

				// Add transaction to mempool if not in new chain
				if !tx.IsCoinBase() && !txExistsInBlocks(tx.ID, newBlocks) {
					txToRestore = append(txToRestore, tx)
				}
			}

			// Update height index
			heightBucket.Delete([]byte(strconv.Itoa(blk.Height)))
		}

		// Apply new blocks in reverse order (from oldest to newest)
		// We reverse newBlocks since it was collected from tip to ancestor
		for i := len(newBlocks) - 1; i >= 0; i-- {
			blk := newBlocks[i]
			if bytes.Equal(blk.Hash, commonAncestor.Hash) {
				continue // Skip common ancestor
			}

			// Update UTXO set with the new block
			if err := updateUTXOSet(tx, blk); err != nil {
				return err
			}

			// Update height index
			heightBucket.Put([]byte(strconv.Itoa(blk.Height)), blk.Hash)

			// Update block storage if needed
			blockData, err := blk.Serialize()
			if err != nil {
				return err
			}
			blocksBucket.Put(blk.Hash, blockData)
		}

		// Update chain tip
		blocksBucket.Put([]byte("l"), newTip.Hash)
		bc.tip = newTip.Hash

		return nil
	})

	if err != nil {
		return nil, err
	}
	return txToRestore, nil
}

func (bc *Blockchain) findForkPoint(blockA *block.Block, chainB *Blockchain) (*block.Block, error) {
	seen := make(map[string]bool)

	currentB := chainB.tip
	depthLimit := 100 // Limit how far back we'll scan to avoid huge maps

	err := chainB.Db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		for range depthLimit {
			blockData := b.Get(currentB)
			if blockData == nil {
				break
			}

			blockObj, err := block.Deserialize(blockData)
			if err != nil {
				return err
			}

			// Add to map
			seen[hex.EncodeToString(currentB)] = true

			// Move to previous block
			if blockObj.PreviousHash == nil {
				break // Genesis block
			}
			currentB = blockObj.PreviousHash
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	current := blockA
	for {
		if current == nil {
			return nil, fmt.Errorf("no common ancestor found between chains")
		}

		// Check if this block exists in chain B
		hashStr := hex.EncodeToString(current.Hash)
		if seen[hashStr] {
			return current, nil
		}

		// Move to previous block
		if current.PreviousHash == nil {
			return nil, fmt.Errorf("no common ancestor found between chains")
		}

		var err error
		current, err = bc.GetBlock(current.PreviousHash)
		if err != nil {
			return nil, err
		}
	}
}

func (bc *Blockchain) collectBlocksFrom(startBlock, endBlock *block.Block) ([]*block.Block, error) {
	var blocks []*block.Block
	current := startBlock

	if bytes.Equal(startBlock.Hash, endBlock.Hash) {
		return []*block.Block{startBlock}, nil
	}

	for {
		blocks = append(blocks, current)

		// Check if we've reached the end block
		if bytes.Equal(current.Hash, endBlock.Hash) {
			break
		}

		// Get previous block
		prev, err := bc.GetBlock(current.PreviousHash)
		if err != nil {
			return nil, err
		}

		if prev == nil {
			return nil, fmt.Errorf("block chain broken, previous block not found")
		}

		current = prev

		// Safety check to avoid infinite loops
		if current.Height < endBlock.Height {
			return nil, fmt.Errorf("went past end block without finding it")
		}
	}

	return blocks, nil
}

func CleanupForkDB(forkChain *Blockchain) error {
	return forkChain.Db.Update(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bbolt.Bucket) error {
			return tx.DeleteBucket(name)
		})
	})
}

func txExistsInBlocks(txID []byte, blocks []*block.Block) bool {
	for _, blk := range blocks {
		for _, tx := range blk.Transactions {
			if bytes.Equal(tx.ID, txID) {
				return true
			}
		}
	}
	return false
}
