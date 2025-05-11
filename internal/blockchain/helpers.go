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
)

func adjustDifficulty(bc *Blockchain) int {
	// Simple difficulty adjustment - every 10 blocks
	currentHeight := bc.CurrentHeight()
	if currentHeight > 0 && currentHeight%blocksToAdjust == 0 {
		// Get current difficulty
		currentDifficulty := bc.CurrentDifficulty()

		// Get timestamps for blocks at start and end of adjustment interval
		lastBlock := bc.LastBlock()
		tenBlocksAgo := bc.GetBlockAtHeight(currentHeight - blocksToAdjust)

		// Calculate time taken for last 10 blocks
		timeSpan := lastBlock.Timestamp.Sub(tenBlocksAgo.Timestamp).Seconds()
		expectedTimeSpan := float64(targetBlockTime * blocksToAdjust)

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

func buildUTXOKey(txID []byte, index int) []byte {
	return append(txID, byte(index))
}

func reverseChain(chain []*block.Block) []*block.Block {
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}

func containsBlock(chain []*block.Block, blk *block.Block) bool {
	for _, b := range chain {
		if bytes.Equal(b.Hash, blk.Hash) {
			return true
		}
	}
	return false
}

func findForkPoint(currentChain, newChain []*block.Block) (*block.Block, error) {
	for _, current := range currentChain {
		if containsBlock(newChain, current) {
			return current, nil
		}
	}
	return nil, fmt.Errorf("no common ancestor found for chain reorganization")
}
