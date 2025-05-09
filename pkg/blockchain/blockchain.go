package blockchain

import (
	"log"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
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
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			genesis := block.New(nil, []byte("0"), 0)
			b, _ := tx.CreateBucket([]byte(blocksBucket))
			b.Put(genesis.Hash, genesis.Serialize())
			b.Put([]byte("l"), genesis.Hash)
			tip = genesis.Hash
		} else {
			tip = b.Get([]byte("l"))
		}
		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{tip, db}
}

func (bc *Blockchain) Close() {
	bc.db.Close()
}
