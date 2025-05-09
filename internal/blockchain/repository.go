package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
	"go.etcd.io/bbolt"
)

const (
	blocksToAdjust  = 10
	targetBlockTime = 2 * time.Minute
)

func createGenesisBlock(tx *bbolt.Tx, tip *[]byte) error {
	blocksBucket, err := tx.CreateBucketIfNotExists([]byte(blocksBucket))
	if err != nil {
		return err
	}

	utxoBucket, err := tx.CreateBucketIfNotExists([]byte(utxoBucket))
	if err != nil {
		return err
	}

	// Check if blockchain already exists
	if blocksBucket.Get([]byte("l")) != nil {
		*tip = blocksBucket.Get([]byte("l"))
		return nil
	}

	// Create coinbase transaction for genesis block
	coinbase := transaction.NewCoinBaseTx("Genesis", 50)
	genesis := block.New(0, []*transaction.Transaction{coinbase}, []byte{}, 1)

	// Store genesis block
	blocksBucket.Put(genesis.Hash, genesis.Serialize())
	blocksBucket.Put([]byte("l"), genesis.Hash)
	*tip = genesis.Hash // Initialize UTXO set with genesis block
	// Store UTXOs from coinbase transaction
	utxo := &utxo.UTXO{
		TxID:   coinbase.ID,
		Index:  0,
		Output: coinbase.Outputs[0],
	}
	key := append(coinbase.ID, byte(0))
	utxoBucket.Put(key, utxo.Serialize())

	return nil
}

func addBlockToDb(tx *bbolt.Tx, bc *Blockchain, newBlock *block.Block) error {
	b := tx.Bucket([]byte(blocksBucket))
	heightBucket := tx.Bucket([]byte(heightIndex))
	b.Put(newBlock.Hash, newBlock.Serialize())
	b.Put([]byte("l"), newBlock.Hash)
	bc.tip = newBlock.Hash

	key := []byte(strconv.Itoa(newBlock.Height))
	heightBucket.Put(key, newBlock.Hash)

	if err := updateUTXOSet(tx, newBlock); err != nil {
		return err
	}

	bc.utxoCache.Purge()
	return nil
}

func updateUTXOSet(tx *bbolt.Tx, b *block.Block) error {
	bucket := tx.Bucket([]byte(utxoBucket))

	for _, tx := range b.Transactions {
		// Remove spent outputs
		if !tx.IsCoinBase() {
			for _, input := range tx.Inputs {
				// Construct key for UTXO lookup
				key := append(input.TxID, byte(input.OutIndex))
				bucket.Delete(key)
			}
		}

		// Add new outputs as UTXOs
		for i, output := range tx.Outputs {
			utxo := &utxo.UTXO{
				TxID:   tx.ID,
				Index:  i,
				Output: output,
			}

			key := append(tx.ID, byte(i))
			bucket.Put(key, utxo.Serialize())
		}
	}

	return nil
}

func getUTXOs(tx *bbolt.Tx, address string, utxos *[]*utxo.UTXO) error {
	b := tx.Bucket([]byte(utxoBucket))
	c := b.Cursor()
	prefix := []byte(address + "_")

	for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
		*utxos = append(*utxos, utxo.Deserialize(v))
	}
	return nil
}

func verifyInputTx(tx *transaction.Transaction, bc *Blockchain, inputSum *int) bool {
	for _, in := range tx.Inputs {
		utxos := bc.GetUTXOs(hex.EncodeToString(in.PubKey))
		found := false
		for _, utxo := range utxos {
			if bytes.Equal(utxo.TxID, in.TxID) && utxo.Index == in.OutIndex {
				*inputSum += utxo.Output.Value // Verifica firma
				data := fmt.Sprintf("%x:%d", in.TxID, in.OutIndex)
				hash := sha256.Sum256([]byte(data))
				r := new(big.Int).SetBytes(in.Signature[:len(in.Signature)/2])
				s := new(big.Int).SetBytes(in.Signature[len(in.Signature)/2:])

				x := new(big.Int).SetBytes(in.PubKey[:len(in.PubKey)/2])
				y := new(big.Int).SetBytes(in.PubKey[len(in.PubKey)/2:])
				pubKey := ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}

				if !ecdsa.Verify(&pubKey, hash[:], r, s) {
					return false
				}
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func verifyOutputTx(tx *transaction.Transaction, outputSum *int) {
	for _, out := range tx.Outputs {
		*outputSum += out.Value
	}
}

func (bc *Blockchain) adjustDifficulty() int {
	// Simple difficulty adjustment - every 10 blocks
	// In a real blockchain, this would look at timestamps of previous blocks
	const targetBlockTime = 10 * 60 // 10 minutes in seconds
	const adjustmentInterval = 10   // blocks

	currentHeight := bc.CurrentHeight()
	if currentHeight > 0 && currentHeight%adjustmentInterval == 0 {
		// Get current difficulty
		currentDifficulty := bc.CurrentDifficulty()

		// Get timestamps for blocks at start and end of adjustment interval
		lastBlock := bc.LastBlock()
		tenBlocksAgo := bc.GetBlockAtHeight(currentHeight - adjustmentInterval)

		// Calculate time taken for last 10 blocks
		timeSpan := lastBlock.Timestamp.Sub(tenBlocksAgo.Timestamp).Seconds()
		expectedTimeSpan := float64(targetBlockTime * adjustmentInterval)

		// Adjust difficulty based on time taken
		if timeSpan < expectedTimeSpan/2 {
			return currentDifficulty + 1
		} else if timeSpan > expectedTimeSpan*2 {
			return max(1, currentDifficulty-1) // Don't go below 1
		}

		return currentDifficulty
	}
	// Otherwise, keep current difficulty
	return bc.CurrentDifficulty()
}

// Add method to get current height and last block
func (bc *Blockchain) CurrentHeight() int {
	lastBlock := bc.LastBlock()
	if lastBlock == nil {
		return 0
	}
	return lastBlock.Height
}

func (bc *Blockchain) LastBlock() *block.Block {
	return bc.GetBlock(bc.tip)
}

func (bc *Blockchain) GetBlockAtHeight(height int) *block.Block {
	var blockData []byte

	bc.db.View(func(tx *bbolt.Tx) error {
		heightBucket := tx.Bucket([]byte(heightIndex))
		blockBucket := tx.Bucket([]byte(blocksBucket))

		// Get block hash by height
		key := []byte(strconv.Itoa(height))
		hash := heightBucket.Get(key)
		if hash == nil {
			return nil
		}

		// Get block data using hash
		blockData = blockBucket.Get(hash)
		return nil
	})

	if blockData == nil {
		return nil
	}

	return block.Deserialize(blockData)
}
