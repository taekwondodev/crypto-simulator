package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"

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

func (bc *Blockchain) FindCommonBlock(blockHashes [][]byte) (*block.Block, error) {
	hashesMap := make(map[string]bool)
	for _, hash := range blockHashes {
		hashesMap[hex.EncodeToString(hash)] = true
	}

	lastBlock, err := bc.LastBlock()
	if err != nil {
		return nil, err
	}

	return bc.findFirstMatchingBlock(lastBlock, hashesMap)
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

func (bc *Blockchain) CleanupForkDB(forkChain *Blockchain) error {
	return forkChain.Db.Update(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bbolt.Bucket) error {
			return tx.DeleteBucket(name)
		})
	})
}

func (bc *Blockchain) ReorganizeChain(oldTip *block.Block, newChain *Blockchain) ([]*transaction.Transaction, error) {
	commonAncestor, err := bc.findForkPoint(newChain)
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
		blocksBucket := tx.Bucket([]byte(blocksBucket))
		heightBucket := tx.Bucket([]byte(heightIndex))

		restoredTxs, err := rollbackBlocks(tx, bc, commonAncestor, oldBlocks, newBlocks, heightBucket)
		if err != nil {
			return err
		}
		txToRestore = restoredTxs

		if err := applyNewBlocksReverseOrder(tx, commonAncestor, blocksBucket, newBlocks, heightBucket); err != nil {
			return err
		}

		// Update chain tip
		blocksBucket.Put([]byte("l"), newTip.Hash)
		bc.tip = newTip.Hash

		return nil
	})

	if err != nil {
		return nil, err
	}

	bc.utxoCache.Purge()
	return txToRestore, nil
}

func (bc *Blockchain) findForkPoint(chainB *Blockchain) (*block.Block, error) {
	hashesB, err := bc.collectChainHashes(chainB)
	if err != nil {
		return nil, err
	}

	return bc.FindCommonBlock(hashesB)
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

func (bc *Blockchain) findFirstMatchingBlock(startBlock *block.Block, blocksMap map[string]bool) (*block.Block, error) {
	current := startBlock

	for {
		// Safety check
		if current == nil {
			return nil, fmt.Errorf("unexpected nil block during traversal")
		}

		// Check if current block exists in the other chain
		hashStr := hex.EncodeToString(current.Hash)
		if blocksMap[hashStr] {
			return current, nil
		}

		// Check if we've reached genesis without finding a match
		if current.PreviousHash == nil {
			return nil, fmt.Errorf("no common ancestor found between chains (reached genesis)")
		}

		// Move to previous block
		var err error
		current, err = bc.GetBlock(current.PreviousHash)
		if err != nil {
			return nil, err
		}
	}
}
