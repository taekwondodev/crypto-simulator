package blockchain

import (
	"fmt"
	"log"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
	"go.etcd.io/bbolt"
)

type Blockchain struct {
	Tip       []byte
	Db        *bbolt.DB
	utxoCache *lru.Cache[string, []*utxo.UTXO]
}

func New(dbPath string) *Blockchain {
	Db, err := bbolt.Open(dbPath, 0600, nil)
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

func (bc *Blockchain) InitHeightIndex() {
	err := bc.Db.Update(func(tx *bbolt.Tx) error {
		return createHeightIndex(tx)
	})

	if err != nil {
		log.Panic(err)
	}
}

func (bc *Blockchain) AddBlock(txs []*transaction.Transaction) (*block.Block, error) {
	difficulty, err := adjustDifficulty(bc)
	if err != nil {
		return nil, err
	}
	height, err := bc.CurrentHeight()
	if err != nil {
		return nil, err
	}

	newBlock, err := block.New(height+1, txs, bc.Tip, difficulty)
	if err != nil {
		return nil, err
	}
	err = newBlock.Mine()
	if err != nil {
		return nil, err
	}

	err = bc.Db.Update(func(tx *bbolt.Tx) error {
		return addBlockToDb(tx, bc, newBlock)
	})

	if err != nil {
		return nil, fmt.Errorf("Failed to persist block: %w", err)
	}

	return newBlock, nil
}

func (bc *Blockchain) GetUTXOs(address string) ([]*utxo.UTXO, error) {
	if cached, ok := bc.utxoCache.Get(address); ok {
		return cached, nil
	}

	var utxos []*utxo.UTXO
	err := bc.Db.View(func(tx *bbolt.Tx) error {
		return getUTXOs(tx, address, &utxos)
	})

	if err != nil {
		return nil, err
	}

	bc.utxoCache.Add(address, utxos)
	return utxos, nil
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

func (bc *Blockchain) GetBalance(address string) (int, error) {
	utxos, err := bc.GetUTXOs(address)
	if err != nil {
		return -1, err
	}
	balance := 0
	for _, utxo := range utxos {
		balance += utxo.Output.Value
	}
	return balance, nil
}

func (bc *Blockchain) CurrentDifficulty() (int, error) {
	lastBlock, err := bc.LastBlock()
	if err != nil {
		return -1, err
	}
	if lastBlock == nil {
		return 1, nil // Default difficulty
	}
	return lastBlock.Difficulty, nil
}

func (bc *Blockchain) CurrentHeight() (int, error) {
	lastBlock, err := bc.LastBlock()
	if err != nil {
		return -1, err
	}
	if lastBlock == nil {
		return 0, nil
	}
	return lastBlock.Height, nil
}

func (bc *Blockchain) LastBlock() (*block.Block, error) {
	return bc.GetBlock(bc.Tip)
}

func (bc *Blockchain) GetBlock(hash []byte) (*block.Block, error) {
	var blockData []byte

	err := bc.Db.View(func(tx *bbolt.Tx) error {
		return getBlockByHash(tx, hash, &blockData)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve block: %w", err)
	}

	if blockData == nil {
		return nil, nil
	}

	return block.Deserialize(blockData)
}

func (bc *Blockchain) GetBlockAtHeight(height int) (*block.Block, error) {
	var blockData []byte

	bc.Db.View(func(tx *bbolt.Tx) error {
		return getHashBlockByHeight(tx, height, &blockData)
	})

	if blockData == nil {
		return nil, nil
	}

	return block.Deserialize(blockData)
}
