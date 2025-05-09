package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
)

type Block struct {
	Timestamp    time.Time
	PreviousHash []byte
	Transactions []*transaction.Transaction
	Nonce        int
	Difficulty   int
	Hash         []byte
}

func New(transactions []*transaction.Transaction, prevHash []byte, difficulty int) *Block {
	block := &Block{
		Timestamp:    time.Now(),
		PreviousHash: prevHash,
		Transactions: transactions,
		Nonce:        0,
		Difficulty:   difficulty,
	}
	block.Hash = block.calculateHash()
	return block
}

func (b *Block) Mine() {
	target := strings.Repeat("0", b.Difficulty)
	for {
		hashStr := hex.EncodeToString(b.Hash)
		if strings.HasPrefix(hashStr, target) {
			break
		}
		b.Nonce++
		b.Hash = b.calculateHash()
	}
}

func (b *Block) calculateHash() []byte {
	data := bytes.Join(
		[][]byte{
			[]byte(b.Timestamp.String()),
			b.PreviousHash,
			serializeTransactions(b.Transactions),
			[]byte(strconv.Itoa(b.Nonce)),
			[]byte(strconv.Itoa(b.Difficulty)),
		},
		[]byte{},
	)
	hash := sha256.Sum256(data)
	return hash[:]
}

func serializeTransactions(txs []*transaction.Transaction) []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(txs)
	if err != nil {
		log.Panic("Failed to serialize transactions:", err)
	}
	return buffer.Bytes()
}

func (b *Block) Serialize() []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(b)
	if err != nil {
		log.Panic("Failed to serialize block:", err)
	}
	return buffer.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic("Failed to deserialize block:", err)
	}
	return &block
}
