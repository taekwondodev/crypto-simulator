package main

import (
	"fmt"

	"github.com/taekwondodev/crypto-simulator/pkg/blockchain"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/wallet"
)

func main() {
	difficult := 2

	alice := wallet.NewWallet()
	bob := wallet.NewWallet()

	inputs := []transaction.TxInput{
		{TxID: []byte("tx1"), OutIndex: 0, PubKey: alice.Address},
	}
	outputs := []transaction.TxOutput{
		{Value: 1, PubKeyHash: bob.Address},
	}

	tx := transaction.NewTransaction(inputs, outputs, alice.PrivateKey)

	fmt.Println("Verifying transaction...", tx.Verify())

	bc := blockchain.NewBlockchain()
	bc.AddBlock([]*transaction.Transaction{tx}, difficult)

	bc.Print()
}
