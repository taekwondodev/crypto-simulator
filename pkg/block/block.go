package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
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
		b.Print()
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

	binary.Write(&buffer, binary.LittleEndian, int32(len(txs)))

	for _, tx := range txs {
		txBytes, err := tx.Serialize()
		if err != nil {
			return nil, err
		}

		binary.Write(&buffer, binary.LittleEndian, int32(len(txBytes)))
		buffer.Write(txBytes)
	}

	return buffer.Bytes(), nil
}

func (b *Block) Serialize() ([]byte, error) {
	var buffer bytes.Buffer

	binary.Write(&buffer, binary.LittleEndian, int32(b.Height))
	binary.Write(&buffer, binary.LittleEndian, b.Timestamp)

	binary.Write(&buffer, binary.LittleEndian, int32(len(b.PreviousHash)))
	if len(b.PreviousHash) > 0 {
		buffer.Write(b.PreviousHash)
	}

	txBytes, err := serializeTransactions(b.Transactions)
	if err != nil {
		return nil, err
	}
	binary.Write(&buffer, binary.LittleEndian, int32(len(txBytes)))
	buffer.Write(txBytes)

	binary.Write(&buffer, binary.LittleEndian, int32(b.Nonce))
	binary.Write(&buffer, binary.LittleEndian, int32(b.Difficulty))

	binary.Write(&buffer, binary.LittleEndian, int32(len(b.Hash)))
	buffer.Write(b.Hash)

	return buffer.Bytes(), nil
}

func Deserialize(data []byte) (*Block, error) {
	buffer := bytes.NewReader(data)
	block := &Block{}

	var height int32
	if err := binary.Read(buffer, binary.LittleEndian, &height); err != nil {
		return nil, fmt.Errorf("failed to deserialize height: %w", err)
	}
	block.Height = int(height)

	if err := binary.Read(buffer, binary.LittleEndian, &block.Timestamp); err != nil {
		return nil, fmt.Errorf("failed to deserialize timestamp: %w", err)
	}

	var prevHashLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &prevHashLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize prevHash length: %w", err)
	}
	if prevHashLen > 0 {
		block.PreviousHash = make([]byte, prevHashLen)
		if _, err := io.ReadFull(buffer, block.PreviousHash); err != nil {
			return nil, fmt.Errorf("failed to deserialize prevHash: %w", err)
		}
	}
	var txBytesLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &txBytesLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize tx bytes length: %w", err)
	}
	txBytes := make([]byte, txBytesLen)
	if _, err := io.ReadFull(buffer, txBytes); err != nil {
		return nil, fmt.Errorf("failed to deserialize tx bytes: %w", err)
	}

	txs, err := deserializeTransactions(txBytes)
	if err != nil {
		return nil, err
	}
	block.Transactions = txs

	var nonce, difficulty int32
	if err := binary.Read(buffer, binary.LittleEndian, &nonce); err != nil {
		return nil, fmt.Errorf("failed to deserialize nonce: %w", err)
	}
	block.Nonce = int(nonce)

	if err := binary.Read(buffer, binary.LittleEndian, &difficulty); err != nil {
		return nil, fmt.Errorf("failed to deserialize difficulty: %w", err)
	}
	block.Difficulty = int(difficulty)

	var hashLen int32
	if err := binary.Read(buffer, binary.LittleEndian, &hashLen); err != nil {
		return nil, fmt.Errorf("failed to deserialize hash length: %w", err)
	}
	block.Hash = make([]byte, hashLen)
	if _, err := io.ReadFull(buffer, block.Hash); err != nil {
		return nil, fmt.Errorf("failed to deserialize hash: %w", err)
	}

	return block, nil
}

func deserializeTransactions(data []byte) ([]*transaction.Transaction, error) {
	buffer := bytes.NewReader(data)

	var count int32
	if err := binary.Read(buffer, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("failed to deserialize tx count: %w", err)
	}

	txs := make([]*transaction.Transaction, count)
	for i := int32(0); i < count; i++ {
		var txBytesLen int32
		if err := binary.Read(buffer, binary.LittleEndian, &txBytesLen); err != nil {
			return nil, fmt.Errorf("failed to deserialize tx bytes length: %w", err)
		}

		txBytes := make([]byte, txBytesLen)
		if _, err := io.ReadFull(buffer, txBytes); err != nil {
			return nil, fmt.Errorf("failed to deserialize tx bytes: %w", err)
		}

		tx, err := transaction.Deserialize(txBytes)
		if err != nil {
			return nil, err
		}
		txs[i] = tx
	}

	return txs, nil
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
