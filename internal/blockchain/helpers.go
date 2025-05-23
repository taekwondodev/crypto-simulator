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
)

func adjustDifficulty(bc *Blockchain) (int, error) {
	// Simple difficulty adjustment - every 10 blocks
	currentHeight, err := bc.CurrentHeight()
	if err != nil {
		return 0, err
	}

	if currentHeight > 0 && currentHeight%blocksToAdjust == 0 {
		currentDifficulty, err := bc.CurrentDifficulty()
		if err != nil {
			return 0, err
		}

		lastBlock, err := bc.LastBlock()
		if err != nil {
			return 0, err
		}
		tenBlocksAgo, err := bc.GetBlockAtHeight(currentHeight - blocksToAdjust)
		if err != nil {
			return 0, err
		}

		timeSpan := float64(lastBlock.Timestamp - tenBlocksAgo.Timestamp)
		expectedTimeSpan := float64(targetBlockTime * blocksToAdjust)

		// Adjust difficulty based on time taken
		if timeSpan < expectedTimeSpan/2 {
			return currentDifficulty + 1, nil
		} else if timeSpan > expectedTimeSpan*2 {
			return max(1, currentDifficulty-1), nil // Don't go below 1
		}

		return currentDifficulty, nil
	}
	// Otherwise, keep current difficulty
	return bc.CurrentDifficulty()
}

func verifyInputTx(tx *transaction.Transaction, bc *Blockchain, inputSum *int) bool {
	for _, in := range tx.Inputs {
		hash := sha256.Sum256(in.PubKey)
		pubKeyString := hex.EncodeToString(hash[:])
		utxos, err := bc.GetUTXOs(pubKeyString)
		if err != nil {
			log.Printf("Error getting UTXOs for pubKey %x: %v", in.PubKey, err)
			return false
		}
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
					log.Printf("Signature verification failed for txID: %x, index: %d", in.TxID, in.OutIndex)
					return false
				}
				found = true
				break
			}
		}
		if !found {
			log.Printf("UTXO not found for txID: %x, index: %d", in.TxID, in.OutIndex)
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

func buildUTXOKey(txID []byte, index int) []byte {
	if index < 0 || index > 255 {
		index = 0
	}
	return append([]byte("utxo:"), append(txID, byte(index))...)
}

func txExistsInBlocks(txID []byte, blocks []*block.Block) bool {
	for _, blk := range blocks {
		for _, tx := range blk.Transactions {
			if bytes.Equal(tx.ID, txID) {
				return true
			}
		}
	}
	return false
}
