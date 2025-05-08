package main

import (
	"fmt"

	"github.com/taekwondodev/crypto-simulator/pkg/blockchain"
)

func main() {
	difficult := 2

	bc := blockchain.NewBlockchain()
	bc.AddBlock([]string{"Alice paga Bob 1 BTC", "Bob paga Charlie 0.5 BTC"}, difficult)
	bc.AddBlock([]string{"Charlie paga Dave 0.3 BTC"}, difficult)

	bc.Print()

	fmt.Println("-------------------")
	fmt.Println("Verifying blockchain...", bc.VerifyChain(difficult))
	fmt.Println("-------------------")

	// Test
	fmt.Println("Modifying transaction...")
	bc.TestModifyTransaction(1, 0, "Alice paga Bob 100 BTC")

	fmt.Println("-------------------")
	fmt.Println("Verifying blockchain after modification...", bc.VerifyChain(difficult))
	fmt.Println("-------------------")

	fmt.Println("Blockchain after modification:")
	fmt.Println("-------------------")
	bc.Print()
}
