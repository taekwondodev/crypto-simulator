package mempool

import (
	"encoding/hex"
	"hash/fnv"
	"sync"

	"github.com/taekwondodev/crypto-simulator/internal/blockchain"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
)

const shards = 16

type Mempool struct {
	shards []*mempoolShard
	txChan chan *transaction.Transaction
	bc     *blockchain.Blockchain
}

type mempoolShard struct {
	txs map[string]*transaction.Transaction
	mu  sync.RWMutex
}

func New(bc *blockchain.Blockchain) *Mempool {
	m := &Mempool{
		shards: make([]*mempoolShard, shards),
		txChan: make(chan *transaction.Transaction, 10000),
		bc:     bc,
	}
	for i := range m.shards {
		m.shards[i] = &mempoolShard{txs: make(map[string]*transaction.Transaction)}
	}
	go m.processTransactions()
	return m
}

func (m *Mempool) getShard(txID string) *mempoolShard {
	h := fnv.New32a()
	h.Write([]byte(txID))
	return m.shards[h.Sum32()%shards]
}

func (m *Mempool) Add(tx *transaction.Transaction) {
	m.txChan <- tx
}

func (m *Mempool) processTransactions() {
	for tx := range m.txChan {
		if m.ValidateTransaction(tx) {
			shard := m.getShard(hex.EncodeToString(tx.ID))
			shard.mu.Lock()
			shard.txs[hex.EncodeToString(tx.ID)] = tx
			shard.mu.Unlock()
		}
	}
}

func (m *Mempool) ValidateTransaction(tx *transaction.Transaction) bool {
	// Don't need to validate coinbase transactions
	if tx.IsCoinBase() {
		return true
	}

	return m.bc.VerifyTransaction(tx)
}

func (m *Mempool) Flush() []*transaction.Transaction {
	result := make([]*transaction.Transaction, 0, 100) // Pre-allocate for reasonable batch size

	for i := range m.shards {
		shard := m.shards[i]
		shard.mu.Lock()

		for _, tx := range shard.txs {
			result = append(result, tx)
		}

		// Clear the shard
		shard.txs = make(map[string]*transaction.Transaction)
		shard.mu.Unlock()
	}

	return result
}

func (m *Mempool) Get(txID string) *transaction.Transaction {
	shard := m.getShard(txID)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	tx, exists := shard.txs[txID]
	if !exists {
		return nil
	}
	return tx
}

func (m *Mempool) Count() int {
	count := 0
	for i := range m.shards {
		shard := m.shards[i]
		shard.mu.RLock()
		count += len(shard.txs)
		shard.mu.RUnlock()
	}
	return count
}
