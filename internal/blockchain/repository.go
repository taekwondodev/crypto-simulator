package blockchain

import (
	"bytes"
	"strconv"
	"time"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
	"go.etcd.io/bbolt"
)

const (
	utxoBucket      = "utxo"
	blocksBucket    = "blocks"
	heightIndex     = "height_index"
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
	coinbase, err := transaction.NewCoinBaseTx("Genesis", 50)
	if err != nil {
		return err
	}
	genesis, err := block.New(0, []*transaction.Transaction{coinbase}, nil, 1)
	if err != nil {
		return err
	}

	// Store genesis block
	serialize, err := genesis.Serialize()
	if err != nil {
		return err
	}
	blocksBucket.Put(genesis.Hash, serialize)
	blocksBucket.Put([]byte("l"), genesis.Hash)
	*tip = genesis.Hash // Initialize UTXO set with genesis block
	// Store UTXOs from coinbase transaction
	utxo := &utxo.UTXO{
		TxID:   coinbase.ID,
		Index:  0,
		Output: coinbase.Outputs[0],
	}
	key := buildUTXOKey(coinbase.ID, 0)
	serialized, err := utxo.Serialize()
	if err != nil {
		return err
	}
	utxoBucket.Put(key, serialized)

	return nil
}

func createHeightIndex(tx *bbolt.Tx) error {
	// Create height index bucket if it doesn't exist
	_, err := tx.CreateBucketIfNotExists([]byte(heightIndex))
	if err != nil {
		return err
	}

	// Populate index if empty
	heightBucket := tx.Bucket([]byte(heightIndex))
	if heightBucket.Get([]byte{0}) == nil { // Check if index is empty
		blockBucket := tx.Bucket([]byte(blocksBucket))
		cursor := blockBucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			if bytes.Equal(k, []byte("l")) {
				continue // Skip tip
			}

			blk, err := block.Deserialize(v)
			if err != nil {
				return err
			}
			// Store hash by height: height -> hash
			key := []byte(strconv.Itoa(blk.Height))
			heightBucket.Put(key, blk.Hash)
		}
	}

	return nil
}

func addBlockToDb(tx *bbolt.Tx, bc *Blockchain, newBlock *block.Block) error {
	b := tx.Bucket([]byte(blocksBucket))
	heightBucket := tx.Bucket([]byte(heightIndex))
	serialize, err := newBlock.Serialize()
	if err != nil {
		return err
	}
	b.Put(newBlock.Hash, serialize)
	b.Put([]byte("l"), newBlock.Hash)
	bc.Tip = newBlock.Hash

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
				key := buildUTXOKey(input.TxID, input.OutIndex)
				bucket.Delete(key)
			}
		}

		for i, output := range tx.Outputs {
			utxo := &utxo.UTXO{
				TxID:   tx.ID,
				Index:  i,
				Output: output,
			}

			key := buildUTXOKey(tx.ID, i)
			serialize, err := utxo.Serialize()
			if err != nil {
				return err
			}
			bucket.Put(key, serialize)
		}
	}

	return nil
}

func getUTXOs(tx *bbolt.Tx, address string, utxos *[]*utxo.UTXO) error {
	b := tx.Bucket([]byte(utxoBucket))
	c := b.Cursor()

	for k, v := c.First(); k != nil; k, v = c.Next() {
		utxo, err := utxo.Deserialize(v)
		if err != nil {
			return err
		}
		if bytes.Equal(utxo.Output.PubKeyHash, []byte(address)) {
			*utxos = append(*utxos, utxo)
		}
	}
	return nil
}

func getBlockByHash(tx *bbolt.Tx, hash []byte, blockData *[]byte) error {
	b := tx.Bucket([]byte(blocksBucket))
	*blockData = b.Get(hash)
	return nil
}

func getHashBlockByHeight(tx *bbolt.Tx, height int, blockData *[]byte) error {
	heightBucket := tx.Bucket([]byte(heightIndex))
	blockBucket := tx.Bucket([]byte(blocksBucket))

	key := []byte(strconv.Itoa(height))
	hash := heightBucket.Get(key)
	if hash == nil {
		return nil
	}

	*blockData = blockBucket.Get(hash)
	return nil
}
