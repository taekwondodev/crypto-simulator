package blockchain

import (
	"time"

	"github.com/taekwondodev/crypto-simulator/pkg/block"
)

type Blockchain struct {
	Blocks []*block.Block
}

func NewBlockchain() *Blockchain {
	return &Blockchain{
		[]*block.Block{block.CreateGenesisBlock()},
	}
}

func (bc *Blockchain) AddBlock(transactions []string, difficulty int) {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := &block.Block{
		Index:        prevBlock.Index + 1,
		Timestamp:    time.Now(),
		PreviousHash: prevBlock.Hash,
		Transactions: transactions,
		Nonce:        0,
	}
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

func (bc *Blockchain) TestModifyTransaction(blockIndex int, txIndex int, newTx string) {
	block := bc.Blocks[blockIndex]
	block.Transactions[txIndex] = newTx
}

func (bc *Blockchain) Print() {
	for _, block := range bc.Blocks {
		block.Print()
	}
}
