package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
)

type Block struct {
	Height       int
	Timestamp    int64
	PreviousHash []byte
	Transactions []*transaction.Transaction
	Nonce        int
	Difficulty   int
	Hash         []byte
}

func Genesis(height int, transactions []*transaction.Transaction, difficulty int, timestamp int64) (*Block, error) {
	var err error
	block := &Block{
		Height:       height,
		Timestamp:    timestamp,
		PreviousHash: nil,
		Transactions: transactions,
		Nonce:        0,
		Difficulty:   difficulty,
	}
	block.Hash, err = block.calculateHash()
	return block, err
}

func New(height int, transactions []*transaction.Transaction, prevHash []byte, difficulty int) (*Block, error) {
	var err error
	block := &Block{
		Height:       height,
		Timestamp:    time.Now().Unix(),
		PreviousHash: prevHash,
		Transactions: transactions,
		Nonce:        0,
		Difficulty:   difficulty,
	}
	return block, err
}

func (b *Block) Mine() error {
	target := strings.Repeat("0", b.Difficulty)
	var err error
	for {
		b.Hash, err = b.calculateHash()
		if err != nil {
			return err
		}
		hashStr := hex.EncodeToString(b.Hash)
		if strings.HasPrefix(hashStr, target) {
			break
		}
		b.Nonce++
	}
	return nil
}

func (b *Block) IsValid(prevBlock *Block) error {
	if !bytes.Equal(b.PreviousHash, prevBlock.Hash) {
		return fmt.Errorf("Previous hash does not match")
	}

	calculatedHash, err := b.calculateHash()
	if err != nil {
		return err
	}
	if !bytes.Equal(calculatedHash, b.Hash) {
		return fmt.Errorf("Hash does not match, block compromised")
	}

	target := strings.Repeat("0", b.Difficulty)
	hashStr := hex.EncodeToString(b.Hash)
	if !strings.HasPrefix(hashStr, target) {
		return fmt.Errorf("Hash does not meet difficulty requirement")
	}

	if b.Height != prevBlock.Height+1 {
		return fmt.Errorf("Block height is not sequential")
	}

	return nil
}

func (b *Block) calculateHash() ([]byte, error) {
	tx, err := serializeTransactions(b.Transactions)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString(strconv.Itoa(b.Height))
	buf.WriteString(strconv.FormatInt(b.Timestamp, 10))
	buf.Write(b.PreviousHash)
	buf.Write(tx)
	buf.WriteString(strconv.Itoa(b.Nonce))
	buf.WriteString(strconv.Itoa(b.Difficulty))

	hash := sha256.Sum256(buf.Bytes())
	return hash[:], nil
}

func serializeTransactions(txs []*transaction.Transaction) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(txs)
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize transactions: %w", err)
	}
	return buffer.Bytes(), nil
}

func (b *Block) Serialize() ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(b)
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize block: %w", err)
	}
	return buffer.Bytes(), nil
}

func Deserialize(data []byte) (*Block, error) {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)
	if err != nil {
		return nil, fmt.Errorf("Failed to deserialize block: %w", err)
	}
	return &block, nil
}

func (b *Block) Print() {
	fmt.Printf("Block %d\n", b.Height)
	fmt.Printf("  Hash: %x\n", b.Hash)
	fmt.Printf("  Prev: %x\n", b.PreviousHash)
	fmt.Printf("  Nonce: %d\n", b.Nonce)
	fmt.Printf("  Difficulty: %d\n", b.Difficulty)
	fmt.Printf("  Timestamp: %s (%d)\n", time.Unix(b.Timestamp, 0).Format(time.RFC3339), b.Timestamp)
	fmt.Printf("  Transactions: %d\n", len(b.Transactions))
	for _, tx := range b.Transactions {
		fmt.Printf("    - %x\n", tx.ID)
	}
	fmt.Println()
}
