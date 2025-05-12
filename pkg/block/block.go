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
	Timestamp    time.Time
	PreviousHash []byte
	Transactions []*transaction.Transaction
	Nonce        int
	Difficulty   int
	Hash         []byte
}

func New(height int, transactions []*transaction.Transaction, prevHash []byte, difficulty int) (*Block, error) {
	var err error
	block := &Block{
		Height:       height,
		Timestamp:    time.Now(),
		PreviousHash: prevHash,
		Transactions: transactions,
		Nonce:        0,
		Difficulty:   difficulty,
	}
	block.Hash, err = block.calculateHash()
	return block, err
}

func (b *Block) Mine() error {
	target := strings.Repeat("0", b.Difficulty)
	var err error
	for {
		hashStr := hex.EncodeToString(b.Hash)
		if strings.HasPrefix(hashStr, target) {
			break
		}
		b.Nonce++
		b.Hash, err = b.calculateHash()
	}
	return err
}

func (b *Block) calculateHash() ([]byte, error) {
	tx, err := serializeTransactions(b.Transactions)
	if err != nil {
		return nil, err
	}
	data := bytes.Join(
		[][]byte{
			[]byte(b.Timestamp.String()),
			b.PreviousHash,
			tx,
			[]byte(strconv.Itoa(b.Nonce)),
			[]byte(strconv.Itoa(b.Difficulty)),
		},
		[]byte{},
	)
	hash := sha256.Sum256(data)
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
	fmt.Printf("Block %d: %x\n", b.Height, b.Hash)
	fmt.Printf("  Transactions: %d\n", len(b.Transactions))
	fmt.Printf("  Timestamp: %s\n", b.Timestamp.Format(time.RFC3339))
	fmt.Printf("  Difficulty: %d\n", b.Difficulty)
	fmt.Println()
}
