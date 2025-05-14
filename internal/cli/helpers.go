package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
	"github.com/taekwondodev/crypto-simulator/pkg/wallet"
)

type Wallet = wallet.Wallet

func askFor(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
}

func getRequiredParam(parts []string, position int, usage string) (string, bool) {
	if len(parts) <= position {
		fmt.Println(usage)
		return "", false
	}
	return parts[position], true
}

func parseAmount(amountStr string) (int, error) {
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %v", err)
	}
	if amount <= 0 {
		return 0, fmt.Errorf("amount must be greater than 0")
	}
	return amount, nil
}

func createTransaction(fromWallet *Wallet, toAddress string, amount int, utxos []*utxo.UTXO) (*transaction.Transaction, error) {
	inputs, collected, err := selectInputs(fromWallet, utxos, amount)
	if err != nil {
		return nil, err
	}

	outputs := createOutputs(fromWallet.GetAddress(), toAddress, amount, collected)

	return transaction.New(inputs, outputs)
}

func selectInputs(wallet *Wallet, availableUTXOs []*utxo.UTXO, amount int) ([]utxo.TxInput, int, error) {
	var inputs []utxo.TxInput
	var collected int

	for _, u := range availableUTXOs {
		input := utxo.TxInput{
			TxID:     u.TxID,
			OutIndex: u.Index,
			PubKey:   wallet.Address,
		}
		input.Sign(wallet.PrivateKey)

		inputs = append(inputs, input)
		collected += u.Output.Value

		if collected >= amount {
			break
		}
	}

	if collected < amount {
		return nil, 0, fmt.Errorf("not enough funds: required %d, available %d", amount, collected)
	}

	return inputs, collected, nil
}

func createOutputs(fromAddress, toAddress string, amount, collected int) []utxo.TxOutput {
	var outputs []utxo.TxOutput

	outputs = append(outputs, utxo.TxOutput{
		Value:      amount,
		PubKeyHash: []byte(toAddress),
	})

	if collected > amount {
		outputs = append(outputs, utxo.TxOutput{
			Value:      collected - amount,
			PubKeyHash: []byte(fromAddress),
		})
	}

	return outputs
}

func handleWalletLookupError(wallets map[string]*Wallet, name string) (*Wallet, error) {
	wallet, exists := wallets[name]
	if !exists {
		return nil, fmt.Errorf("wallet not found: %s\n", name)
	}
	return wallet, nil
}

func checkBalance(available, required int) bool {
	if available < required {
		return false
	}
	return true
}

func calculateBalance(utxos []*utxo.UTXO) int {
	balance := 0
	for _, u := range utxos {
		balance += u.Output.Value
	}
	return balance
}

func executeCommand(f func() error) {
	if err := f(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
