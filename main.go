package main

import (
	"time"

	"github.com/taekwondodev/crypto-simulator/pkg/blockchain"
	"github.com/taekwondodev/crypto-simulator/pkg/mempool"
	"github.com/taekwondodev/crypto-simulator/pkg/p2p"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/wallet"
)

func main() {
	node := p2p.NewNode(":3000")
	go node.Start()

	bc := blockchain.New()
	defer bc.Close()
	mp := mempool.New()

	node.Connect(":3001")

	go minerRoutine(node, bc, mp)

	go transactionGenerator(node, mp)

	select {}
}

func minerRoutine(node *p2p.Node, bc *blockchain.Blockchain, mp *mempool.Mempool) {
	for {
		// Prendi transazioni dalla mempool
		txs := mp.Flush()
		if len(txs) == 0 {
			time.Sleep(10 * time.Second)
			continue
		}

		// Calcola difficoltà
		difficulty := bc.CurrentDifficulty()

		// Aggiungi alla blockchain
		newBlock := bc.AddBlock(txs, difficulty)

		// Diffondi il blocco alla rete
		blockData := newBlock.Serialize()
		node.Broadcast(append([]byte{byte(p2p.MsgBlock)}, blockData...))
	}
}

func transactionGenerator(node *p2p.Node, mp *mempool.Mempool) {
	for {
		// Crea transazione fittizia
		wallet := wallet.NewWallet()
		tx := transaction.NewCoinBaseTx(wallet.GetAddress(), 10)
		mp.Add(tx)

		// Diffondi alla rete
		txData := tx.Serialize()
		node.Broadcast(append([]byte{byte(p2p.MsgTx)}, txData...))

		time.Sleep(30 * time.Second)
	}
}
