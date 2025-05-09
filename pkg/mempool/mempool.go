package mempool

import (
	"encoding/hex"
	"sync"

	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
)

type Mempool struct {
	transactions map[string]*transaction.Transaction
	mu           sync.RWMutex
}

func New() *Mempool {
	return &Mempool{
		transactions: make(map[string]*transaction.Transaction),
	}
}

func (m *Mempool) Add(tx *transaction.Transaction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions[hex.EncodeToString(tx.ID)] = tx
}

func (m *Mempool) Get(txID string) *transaction.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.transactions[txID]
}

func (m *Mempool) Flush() []*transaction.Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	txs := make([]*transaction.Transaction, 0, len(m.transactions))
	for _, tx := range m.transactions {
		txs = append(txs, tx)
	}
	m.transactions = make(map[string]*transaction.Transaction)
	return txs
}
