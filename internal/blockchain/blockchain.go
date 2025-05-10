package blockchain

import (
	"log"

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
	Db        *bbolt.DB
	utxoCache *lru.Cache[string, []*utxo.UTXO]
}

func New() *Blockchain {
	Db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	var tip []byte
	err = Db.Update(func(tx *bbolt.Tx) error {
		return createGenesisBlock(tx, &tip)
	})

	if err != nil {
		log.Panic(err)
	}

	cache, _ := lru.New[string, []*utxo.UTXO](10000)
	return &Blockchain{tip, Db, cache}
}

func (bc *Blockchain) Close() {
	bc.Db.Close()
}

func (bc *Blockchain) InitHeightIndex() error {
	return bc.Db.Update(func(tx *bbolt.Tx) error {
		return createHeightIndex(tx)
	})
}

func (bc *Blockchain) AddBlock(txs []*transaction.Transaction) *block.Block {
	difficulty := adjustDifficulty(bc)
	height := bc.CurrentHeight()

	newBlock := block.New(height+1, txs, bc.tip, difficulty)
	newBlock.Mine()

	err := bc.Db.Update(func(tx *bbolt.Tx) error {
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
	err := bc.Db.View(func(tx *bbolt.Tx) error {
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

func (bc *Blockchain) IsValidChain(newChain []*block.Block) bool {
	for i := 1; i < len(newChain); i++ {
		if !newChain[i].Validate(newChain[i-1]) {
			return false
		}
	}
	return true
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
	lastBlock := bc.LastBlock()
	if lastBlock == nil {
		return 1 // Default difficulty
	}
	return lastBlock.Difficulty
}

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

func (bc *Blockchain) GetBlock(hash []byte) *block.Block {
	var blockData []byte

	bc.Db.View(func(tx *bbolt.Tx) error {
		return getBlockByHash(tx, hash, &blockData)
	})

	if blockData == nil {
		return nil
	}

	return block.Deserialize(blockData)
}

func (bc *Blockchain) GetBlockAtHeight(height int) *block.Block {
	var blockData []byte

	bc.Db.View(func(tx *bbolt.Tx) error {
		return getHashBlockByHeight(tx, height, &blockData)
	})

	if blockData == nil {
		return nil
	}

	return block.Deserialize(blockData)
}
