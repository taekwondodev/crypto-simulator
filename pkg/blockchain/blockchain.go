package blockchain

import (
	"log"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
	"go.etcd.io/bbolt"
)

const (
	dbFile       = "blockchain.db"
	utxoBucket   = "utxo"
	blocksBucket = "blocks"
)

type Blockchain struct {
	tip []byte
	db  *bbolt.DB
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

	return &Blockchain{tip, db}
}

func (bc *Blockchain) Close() {
	bc.db.Close()
}

func (bc *Blockchain) AddBlock(txs []*transaction.Transaction, difficulty int) {
	var lastHash []byte

	err := bc.db.View(func(tx *bbolt.Tx) error {
		return getLastHash(tx, &lastHash)
	})
	if err != nil {
		log.Panic("Failed to get last hash:", err)
	}

	newBlock := block.New(txs, lastHash, difficulty)
	newBlock.Mine()

	err = bc.db.Update(func(tx *bbolt.Tx) error {
		return addBlockToDb(tx, bc, newBlock)
	})

	if err != nil {
		log.Panic("Failed to persist block:", err)
	}
}

func (bc *Blockchain) GetUTXOs(address string) []*utxo.UTXO {
	var utxos []*utxo.UTXO
	err := bc.db.View(func(tx *bbolt.Tx) error {
		return getUTXOs(tx, address, &utxos)
	})

	if err != nil {
		log.Panic(err)
	}
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
