package blockchain

import (
	"bytes"
	"fmt"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
	"go.etcd.io/bbolt"
)

func (bc *Blockchain) IsValidChain(newChain []*block.Block) bool {
	// Verifica l'intera catena
	for i := 1; i < len(newChain); i++ {
		if !newChain[i].Validate(newChain[i-1]) {
			return false
		}
	}
	return true
}

func (bc *Blockchain) ReorganizeChain(newChain []*block.Block) error {
	return bc.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		utxoBucket := tx.Bucket([]byte(utxoBucket))

		// Find common ancestor
		currentChain := bc.getChainFrom(bc.tip)
		var forkPoint *block.Block

		for _, current := range currentChain {
			if containsBlock(newChain, current) {
				forkPoint = current
				break
			}
		}

		if forkPoint == nil {
			return fmt.Errorf("no common ancestor found for chain reorganization")
		}
		// Rollback transactions from current chain back to fork point
		for _, blk := range currentChain {
			if bytes.Equal(blk.Hash, forkPoint.Hash) {
				break
			}

			// Revert UTXO changes
			for _, tx := range blk.Transactions {
				// Restore inputs (they were previously spent)
				for _, input := range tx.Inputs {
					origTx := bc.FindTransaction(input.TxID)
					if origTx != nil {
						utxo := &utxo.UTXO{
							TxID:   input.TxID,
							Index:  input.OutIndex,
							Output: origTx.Outputs[input.OutIndex],
						}
						key := append(input.TxID, byte(input.OutIndex))
						utxoBucket.Put(key, utxo.Serialize())
					}
				}

				// Remove outputs (they are no longer valid)
				for i := range tx.Outputs {
					key := append(tx.ID, byte(i))
					utxoBucket.Delete(key)
				}
			}
		}
		// Apply new chain blocks from fork point
		for _, blk := range newChain {
			if bytes.Equal(blk.Hash, forkPoint.Hash) {
				continue // Skip common ancestor
			}

			// Apply UTXO changes for this block
			for _, tx := range blk.Transactions {
				// Remove spent outputs
				if !tx.IsCoinBase() {
					for _, input := range tx.Inputs {
						key := append(input.TxID, byte(input.OutIndex))
						utxoBucket.Delete(key)
					}
				}
				// Add new outputs
				for i, output := range tx.Outputs {
					utxo := &utxo.UTXO{
						TxID:   tx.ID,
						Index:  i,
						Output: output,
					}
					key := append(tx.ID, byte(i))
					utxoBucket.Put(key, utxo.Serialize())
				}
			}

			// Store the block
			bucket.Put(blk.Hash, blk.Serialize())
		}

		// Update blockchain tip to point to the tip of the new chain
		newTip := newChain[len(newChain)-1].Hash
		bucket.Put([]byte("l"), newTip)
		bc.tip = newTip

		// Clear the UTXO cache since we've made significant changes
		bc.utxoCache.Purge()

		return nil
	})
}

func (bc *Blockchain) FindTransaction(id []byte) *transaction.Transaction {
	var t *transaction.Transaction

	bc.db.View(func(tx *bbolt.Tx) error {
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

	bc.db.View(func(tx *bbolt.Tx) error {
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

		// Salta indietro esponenzialmente
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

func containsBlock(chain []*block.Block, blk *block.Block) bool {
	for _, b := range chain {
		if bytes.Equal(b.Hash, blk.Hash) {
			return true
		}
	}
	return false
}

func revertTransactions(txs []*transaction.Transaction) {
	// Revert transactions by updating the UTXO set
	// For each transaction, restore inputs and remove outputs
	for _, tx := range txs {
		if tx.IsCoinBase() {
			continue
		}
		// Implementation would restore UTXOs consumed by this transaction
	}
}

func reverseChain(chain []*block.Block) []*block.Block {
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}
