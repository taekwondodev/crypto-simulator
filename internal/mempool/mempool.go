package mempool

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"sync"

	"github.com/taekwondodev/crypto-simulator/internal/blockchain"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
)

const shards = 16

type Mempool struct {
	shards []*mempoolShard
	bc     *blockchain.Blockchain
}

type mempoolShard struct {
	txs map[string]*transaction.Transaction
	mu  sync.RWMutex
}

func New(bc *blockchain.Blockchain) *Mempool {
	m := &Mempool{
		shards: make([]*mempoolShard, shards),
		bc:     bc,
	}
	for i := range m.shards {
		m.shards[i] = &mempoolShard{txs: make(map[string]*transaction.Transaction)}
	}
	return m
}

func (m *Mempool) getShard(txID string) *mempoolShard {
	h := fnv.New32a()
	h.Write([]byte(txID))
	return m.shards[h.Sum32()%shards]
}

func (m *Mempool) Add(tx *transaction.Transaction) bool {
	if m.ValidateTransaction(tx) {
		shard := m.getShard(hex.EncodeToString(tx.ID))
		shard.mu.Lock()
		defer shard.mu.Unlock()
		shard.txs[hex.EncodeToString(tx.ID)] = tx
		return true
	}
	return false
}

func (m *Mempool) ValidateTransaction(tx *transaction.Transaction) bool {
	// Don't need to validate coinbase transactions
	if tx.IsCoinBase() {
		return true
	}

	return m.bc.VerifyTransaction(tx)
}

func (m *Mempool) Remove(txID string) {
	shard := m.getShard(txID)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	delete(shard.txs, txID)
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

func (m *Mempool) HandleCoinbaseTxs(minerAddress string, reward int) ([]*transaction.Transaction, error) {
	txs := m.Flush()

	coinbase, err := transaction.NewCoinBaseTx(minerAddress, reward)
	if err != nil {
		return txs, err
	}
	txs = append([]*transaction.Transaction{coinbase}, txs...)

	fmt.Printf("Mining new block with %d transactions from mempool\n", len(txs))
	return txs, nil
}
