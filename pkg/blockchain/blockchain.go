package blockchain

import (
	"github.com/taekwondodev/crypto-simulator/pkg/block"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
)

type Blockchain struct {
	Blocks []*block.Block
}

func NewBlockchain() *Blockchain {
	return &Blockchain{
		[]*block.Block{block.CreateGenesisBlock()},
	}
}

func (bc *Blockchain) AddBlock(transactions []*transaction.Transaction, difficulty int) {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := block.NewBlock(prevBlock.Index+1, prevBlock.Hash, transactions)
	newBlock.MineBlock(difficulty)
	bc.Blocks = append(bc.Blocks, newBlock)
}

func (bc *Blockchain) VerifyChain(difficulty int) bool {
	for i := 1; i < len(bc.Blocks); i++ {
		prevBlock := bc.Blocks[i-1]
		currentBlock := bc.Blocks[i]

		if currentBlock.PreviousHash != prevBlock.Hash {
			return false
		}

		if !currentBlock.CheckHashBlock(difficulty) {
			return false
		}
	}
	return true
}

func (bc *Blockchain) Print() {
	for _, block := range bc.Blocks {
		block.Print()
	}
}
