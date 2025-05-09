package main

import (
	"fmt"
	"log"

	"github.com/taekwondodev/crypto-simulator/pkg/blockchain"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
	"github.com/taekwondodev/crypto-simulator/pkg/wallet"
)

func main() {
	bc := blockchain.New()
	defer bc.Close()

	difficulty := 2

	// Miner -> Alice
	minerWallet := wallet.NewWallet()
	minerAddress := minerWallet.GetAddress()
	aliceWallet := wallet.NewWallet()
	aliceAddress := aliceWallet.GetAddress()

	coinbaseTx := transaction.NewCoinBaseTx(minerAddress, 100)
	bc.AddBlock([]*transaction.Transaction{coinbaseTx}, difficulty)

	utxos := bc.GetUTXOs(minerAddress)
	if len(utxos) == 0 {
		log.Panic("Nessun UTXO disponibile per il miner!")
	}

	input := utxo.TxInput{
		TxID:     utxos[0].TxID,
		OutIndex: utxos[0].Index,
		PubKey:   minerWallet.Address,
	}
	input.Sign(minerWallet.PrivateKey)

	outputs := []utxo.TxOutput{
		{Value: 70, PubKeyHash: []byte(aliceAddress)},
		{Value: 30, PubKeyHash: []byte(minerAddress)},
	}

	tx := transaction.New([]utxo.TxInput{input}, outputs)
	if !bc.VerifyTransaction(tx) {
		log.Panic("Transazione non valida!")
	}

	bc.AddBlock([]*transaction.Transaction{tx}, difficulty)

	fmt.Printf("Saldo Miner: %d\n", bc.GetBalance(minerAddress))
	fmt.Printf("Saldo Alice: %d\n", bc.GetBalance(aliceAddress))
}
