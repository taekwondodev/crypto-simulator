package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
	"go.etcd.io/bbolt"
)

func (bc *Blockchain) AddBlock(txs []*transaction.Transaction, difficulty int) {
	var lastHash []byte

	err := bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))
		return nil
	})
	if err != nil {
		log.Panic("Failed to get last hash:", err)
	}

	newBlock := block.New(txs, lastHash, difficulty)
	newBlock.Mine()

	err = bc.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		b.Put(newBlock.Hash, newBlock.Serialize())
		b.Put([]byte("l"), newBlock.Hash)
		bc.tip = newBlock.Hash

		utxoBucket := tx.Bucket([]byte(utxoBucket))
		for _, tx := range newBlock.Transactions {
			txID := tx.ID
			for outIdx, output := range tx.Outputs {
				key := fmt.Sprintf("%x_%d", txID, outIdx)
				utxo := &utxo.UTXO{
					TxID:   txID,
					Index:  outIdx,
					Output: output,
				}
				utxoBucket.Put([]byte(key), utxo.Serialize())
			}
		}
		return nil
	})

	if err != nil {
		log.Panic("Failed to persist block:", err)
	}
}

func (bc *Blockchain) GetUTXOs(address string) []*utxo.UTXO {
	var utxos []*utxo.UTXO
	err := bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()
		prefix := []byte(address + "_")

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			utxos = append(utxos, utxo.Deserialize(v))
		}
		return nil
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
	for _, in := range tx.Inputs {
		utxos := bc.GetUTXOs(hex.EncodeToString(in.PubKey))
		found := false
		for _, utxo := range utxos {
			if bytes.Equal(utxo.TxID, in.TxID) && utxo.Index == in.OutIndex {
				inputSum += utxo.Output.Value // Verifica firma
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
	outputSum := 0
	for _, out := range tx.Outputs {
		outputSum += out.Value
	}

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
