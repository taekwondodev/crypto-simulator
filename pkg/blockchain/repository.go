package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
	"go.etcd.io/bbolt"
)

func getLastHash(tx *bbolt.Tx, lastHash *[]byte) error {
	b := tx.Bucket([]byte(blocksBucket))
	*lastHash = b.Get([]byte("l"))
	return nil
}

func createGenesisBlock(tx *bbolt.Tx, tip *[]byte) error {
	b := tx.Bucket([]byte(blocksBucket))
	if b == nil {
		genesis := block.New(nil, []byte("0"), 0)
		b, _ := tx.CreateBucket([]byte(blocksBucket))
		b.Put(genesis.Hash, genesis.Serialize())
		b.Put([]byte("l"), genesis.Hash)
		*tip = genesis.Hash
	} else {
		*tip = b.Get([]byte("l"))
	}
	return nil
}

func addBlockToDb(tx *bbolt.Tx, bc *Blockchain, newBlock *block.Block) error {
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
