package block

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Block struct {
	Index        int
	Timestamp    time.Time
	PreviousHash string
	Transactions []string
	Nonce        int
	Hash         string
}

// The first block in the blockchain
func CreateGenesisBlock() *Block {
	genesisBlock := &Block{
		Index:        0,
		Timestamp:    time.Now(),
		PreviousHash: "0", // Genesis block has no previous hash
		Transactions: []string{"Genesis Transaction"},
		Nonce:        0,
	}
	genesisBlock.Hash = calculateHash(genesisBlock)

	return genesisBlock
}

func (b *Block) MineBlock(difficulty int) {
	prefix := strings.Repeat("0", difficulty) // Difficulty based on 0s
	for {
		b.Hash = calculateHash(b)
		if strings.HasPrefix(b.Hash, prefix) {
			break
		}
		b.Nonce++
	}
}

func (b *Block) CheckHashBlock(difficulty int) bool {
	prefix := strings.Repeat("0", difficulty)
	return isValidHash(b, prefix)
}

func calculateHash(b *Block) string {
	data := fmt.Sprintf("%d%s%s%v%d",
		b.Index,
		b.Timestamp.String(),
		b.PreviousHash,
		b.Transactions,
		b.Nonce,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func isValidHash(b *Block, prefix string) bool {
	if b.Hash != calculateHash(b) {
		return false
	}

	if !strings.HasPrefix(b.Hash, prefix) {
		return false
	}

	return true
}

func (b *Block) Print() {
	fmt.Println("Index:", b.Index)
	fmt.Println("Timestamp:", b.Timestamp.String())
	fmt.Println("Previous Hash:", b.PreviousHash)
	fmt.Println("Transactions:", b.Transactions)
	fmt.Println("Nonce:", b.Nonce)
	fmt.Println("Hash:", b.Hash)
	fmt.Println("-------------------")
}
