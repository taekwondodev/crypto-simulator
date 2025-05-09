package blockchain

import (
	"bytes"
	"log"
	"strconv"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
	"go.etcd.io/bbolt"
)

const (
	dbFile       = "blockchain.db"
	utxoBucket   = "utxo"
	blocksBucket = "blocks"
	heightIndex  = "height_index"
)

type Blockchain struct {
	tip       []byte
	db        *bbolt.DB
	utxoCache *lru.Cache[string, []*utxo.UTXO]
}

func New() *Blockchain {
	db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	var tip []byte
	err = db.Update(func(tx *bbolt.Tx) error {
		return createGenesisBlock(tx, &tip)
	})

	if err != nil {
		log.Panic(err)
	}

	cache, _ := lru.New[string, []*utxo.UTXO](10000)
	return &Blockchain{tip, db, cache}
}

func (bc *Blockchain) Close() {
	bc.db.Close()
}

func (bc *Blockchain) InitHeightIndex() error {
	return bc.db.Update(func(tx *bbolt.Tx) error {
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

				blk := block.Deserialize(v)
				// Store hash by height: height -> hash
				key := []byte(strconv.Itoa(blk.Height))
				heightBucket.Put(key, blk.Hash)
			}
		}

		return nil
	})
}

func (bc *Blockchain) AddBlock(txs []*transaction.Transaction) *block.Block {
	difficulty := bc.adjustDifficulty()
	height := bc.CurrentHeight()

	newBlock := block.New(height+1, txs, bc.tip, difficulty)
	newBlock.Mine()

	err := bc.db.Update(func(tx *bbolt.Tx) error {
		return addBlockToDb(tx, bc, newBlock)
	})

	if err != nil {
		log.Panic("Failed to persist block:", err)
	}

	return newBlock
}

func (bc *Blockchain) GetUTXOs(address string) []*utxo.UTXO {
	if cached, ok := bc.utxoCache.Get(address); ok {
		return cached
	}

	var utxos []*utxo.UTXO
	err := bc.db.View(func(tx *bbolt.Tx) error {
		return getUTXOs(tx, address, &utxos)
	})

	if err != nil {
		log.Panic(err)
	}

	bc.utxoCache.Add(address, utxos)
	return utxos
}

func (bc *Blockchain) VerifyTransaction(tx *transaction.Transaction) bool {
	if tx.IsCoinBase() {
		return true
	}

	inputSum := 0
	if !verifyInputTx(tx, bc, &inputSum) {
		return false
	}

	outputSum := 0
	verifyOutputTx(tx, &outputSum)

	return inputSum >= outputSum
}

func (bc *Blockchain) GetBalance(address string) int {
	utxos := bc.GetUTXOs(address)
	balance := 0
	for _, utxo := range utxos {
		balance += utxo.Output.Value
	}
	return balance
}

func (bc *Blockchain) CurrentDifficulty() int {
	var lastBlock *block.Block
	err := bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastBlock = block.Deserialize(b.Get(bc.tip))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return lastBlock.Difficulty
}

func (bc *Blockchain) GetBlock(hash []byte) *block.Block {
	var blockData []byte

	bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockData = b.Get(hash)
		return nil
	})

	if blockData == nil {
		return nil
	}

	return block.Deserialize(blockData)
}
