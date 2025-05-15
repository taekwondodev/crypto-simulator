package cli

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"
	"github.com/taekwondodev/crypto-simulator/internal/blockchain"
	"github.com/taekwondodev/crypto-simulator/internal/mempool"
	"github.com/taekwondodev/crypto-simulator/internal/p2p"
	"github.com/taekwondodev/crypto-simulator/pkg/wallet"
)

type CLI struct {
	bc              *blockchain.Blockchain
	mp              *mempool.Mempool
	node            *p2p.Node
	wallets         map[string]*Wallet
	commandHandlers map[string]func([]string) error
	Done            chan struct{}
	line            *liner.State
	historyFile     string
}

func NewCLI(bc *blockchain.Blockchain, mp *mempool.Mempool, node *p2p.Node) *CLI {
	line := liner.NewLiner()
	historyFile := filepath.Join(os.TempDir(), ".crypto_simulator_history")

	cli := &CLI{
		bc:          bc,
		mp:          mp,
		node:        node,
		wallets:     make(map[string]*Wallet),
		Done:        make(chan struct{}),
		line:        line,
		historyFile: historyFile,
	}

	if f, err := os.Open(historyFile); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	cli.commandHandlers = map[string]func([]string) error{
		"help":         func(parts []string) error { return cli.printHelp() },
		"createwallet": func(parts []string) error { return cli.createWallet() },
		"listwallet":   func(parts []string) error { return cli.listWallets() },
		"balance":      func(parts []string) error { return cli.handleBalance(parts) },
		"send":         func(parts []string) error { return cli.handleSend(parts) },
		"mine":         func(parts []string) error { return cli.mineBlock() },
		"blockchain":   func(parts []string) error { return cli.printBlockchain() },
		"mempool":      func(parts []string) error { return cli.showMempool() },
		"peers":        func(parts []string) error { return cli.listPeers() },
		"connect":      func(parts []string) error { return cli.handleConnect(parts) },
		"tx":           func(parts []string) error { return cli.handleTransactionDetails(parts) },
	}

	if err := cli.LoadWallets(); err != nil {
		log.Println("Error loading wallets:", err)
	}
	return cli
}

func (cli *CLI) Run() {
	fmt.Println("Blockchain CLI")
	fmt.Println("Type 'help' for commands")
	defer cli.cleanupLiner()

	for {
		input := cli.askFor("> ")
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		if command == "exit" || command == "quit" {
			close(cli.Done)
			break
		}

		handler, exists := cli.commandHandlers[command]
		if !exists {
			fmt.Println("Unknown command:", command)
			cli.printHelp()
			continue
		}

		executeCommand(func() error {
			return handler(parts)
		})
	}
}

func (cli *CLI) printHelp() error {
	fmt.Println("Commands:")
	commands := []struct {
		cmd  string
		desc string
	}{
		{"createwallet", "Create a new wallet"},
		{"listwallet", "List all wallets"},
		{"balance <name>", "Get balance for wallet"},
		{"send <from> <to-address> <amount>", "Send coins from one wallet to another"},
		{"mine", "Mine a new block with transactions from mempool"},
		{"blockchain", "Print the blockchain"},
		{"mempool", "Show transactions in mempool"},
		{"peers", "List connected peers"},
		{"connect <address>", "Connect to a peer"},
		{"exit", "Exit the program"},
		{"tx <txid>", "View transaction details"},
	}

	// Find the longest command for proper padding
	maxLen := 0
	for _, cmd := range commands {
		if len(cmd.cmd) > maxLen {
			maxLen = len(cmd.cmd)
		}
	}

	// Print each command with consistent formatting
	format := "  %-" + fmt.Sprintf("%d", maxLen+2) + "s- %s\n"
	for _, cmd := range commands {
		fmt.Printf(format, cmd.cmd, cmd.desc)
	}

	return nil
}

func (cli *CLI) createWallet() error {
	name := cli.askFor("Enter wallet name: ")
	if name == "" {
		return fmt.Errorf("wallet name cannot be empty")
	}

	if _, exists := cli.wallets[name]; exists {
		return fmt.Errorf("Wallet with that name already exists")
	}

	cli.wallets[name] = wallet.NewWallet()

	if err := cli.SaveWallets(); err != nil {
		return err
	}

	fmt.Println("Created new wallet:", name)
	fmt.Println("Address:", cli.wallets[name].GetAddress())
	return nil
}

func (cli *CLI) listWallets() error {
	if len(cli.wallets) == 0 {
		return fmt.Errorf("No wallets created yet. Use 'createwallet' to create one.")
	}

	fmt.Println("Wallets:")
	for name, w := range cli.wallets {
		fmt.Printf("  %s: %s\n", name, w.GetAddress())
	}
	return nil
}

func (cli *CLI) handleBalance(parts []string) error {
	walletName, ok := getRequiredParam(parts, 1, "Usage: balance <wallet-name>")
	if !ok {
		return fmt.Errorf("missing wallet name")
	}

	w, err := handleWalletLookupError(cli.wallets, walletName)
	if err != nil {
		return err
	}

	address := w.GetAddress()
	balance, err := cli.bc.GetBalance(address)
	if err != nil {
		return err
	}
	fmt.Printf("Balance of %s: %d coins\n", walletName, balance)
	return nil
}

func (cli *CLI) handleSend(parts []string) error {
	if len(parts) < 4 {
		return fmt.Errorf("Usage: send <from-wallet> <to-address> <amount>")
	}

	fromName := parts[1]
	toAddress := parts[2]

	amount, err := parseAmount(parts[3])
	if err != nil {
		return err
	}

	from, err := handleWalletLookupError(cli.wallets, fromName)
	if err != nil {
		return err
	}

	// Get UTXOs for the sender
	fromAddress := from.GetAddress()
	utxos, err := cli.bc.GetUTXOs(fromAddress)
	if err != nil {
		return err
	}

	balance := calculateBalance(utxos)
	if !checkBalance(balance, amount) {
		return fmt.Errorf("Not enough balance. Available: %d, Required: %d", balance, amount)
	}

	tx, err := createTransaction(from, toAddress, amount, utxos)
	if err != nil {
		return err
	}
	if cli.mp.Add(tx) {
		fmt.Printf("Transaction created: %x\n", tx.ID)
		fmt.Printf("Sent %d coins from %s to %s\n", amount, fromName, toAddress)
		fmt.Println("Transaction is in mempool, waiting to be mined")

		invMsg := p2p.NewInvMessage([][]byte{tx.ID})
		cli.node.Broadcast(invMsg)
	}
	return nil
}

func (cli *CLI) mineBlock() error {
	minerAddress, err := cli.handleMiningReward()
	if err != nil {
		return err
	}

	txs, err := cli.mp.HandleCoinbaseTxs(minerAddress, 50)
	if err != nil {
		return err
	}

	newBlock, err := cli.bc.CreateBlock(txs)
	if err != nil {
		return err
	}

	if err := cli.bc.AddBlock(newBlock); err != nil {
		return err
	}
	fmt.Printf("Block mined! Height: %d\n", newBlock.Height)

	invMsg := p2p.NewInvMessage([][]byte{newBlock.Hash})
	cli.node.Broadcast(invMsg)

	return nil
}

func (cli *CLI) printBlockchain() error {
	height, err := cli.bc.CurrentHeight()
	if err != nil {
		return err
	}
	fmt.Printf("Blockchain height: %d\n", height)

	fmt.Println("Latest blocks:")
	for i := height; i >= 0 && i > height-10; i-- {
		block, err := cli.bc.GetBlockAtHeight(i)
		if err != nil {
			return err
		}
		if block == nil {
			continue
		}
		block.Print()
	}
	return nil
}

func (cli *CLI) showMempool() error {
	count := cli.mp.Count()

	if count == 0 {
		fmt.Printf("Mempool is empty\n")
	} else {
		fmt.Printf("Mempool has %d transaction(s)\n", count)
	}
	return nil
}

func (cli *CLI) listPeers() error {
	peers := cli.node.GetPeers()

	if len(peers) == 0 {
		fmt.Printf("No peers connected")
	}

	fmt.Println("Connected peers:")
	for _, peer := range peers {
		fmt.Printf("  %s (last seen: %s)\n", peer.Address, peer.LastSeen.Format("15:04:05"))
	}
	return nil
}

func (cli *CLI) handleConnect(parts []string) error {
	address, ok := getRequiredParam(parts, 1, "Usage: connect <peer-address>")
	if !ok {
		return fmt.Errorf("missing peer address")
	}

	fmt.Printf("Connecting to peer %s...\n", address)
	cli.node.Connect(address)
	fmt.Println("Connection initiated")
	return nil
}

func (cli *CLI) handleTransactionDetails(parts []string) error {
	txID, ok := getRequiredParam(parts, 1, "Usage: tx <transaction-id>")
	if !ok {
		return fmt.Errorf("missing transaction ID")
	}

	// Try to find transaction in mempool first
	tx := cli.mp.Get(txID)
	source := "mempool"

	// If not in mempool, try to find in blockchain
	if tx == nil {
		txHash, err := hex.DecodeString(txID)
		if err != nil {
			return fmt.Errorf("Invalid transaction ID format")
		}

		tx = cli.bc.FindTransaction(txHash)
		source = "blockchain"
	}

	if tx == nil {
		return fmt.Errorf("Transaction not found")
	}

	fmt.Printf("Transaction Details (from %s):\n", source)
	tx.Print("  ")
	return nil
}

func (cli *CLI) handleMiningReward() (string, error) {
	if len(cli.wallets) == 0 {
		return "", fmt.Errorf("No wallets available. Create a wallet first with 'createwallet'")
	}

	if len(cli.wallets) == 1 {
		for _, wallet := range cli.wallets {
			return wallet.GetAddress(), nil
		}
	}

	fmt.Println("Available wallets:")
	for name := range cli.wallets {
		fmt.Println(" -", name)
	}
	walletName := cli.askFor("Enter wallet name to receive mining reward: ")

	wallet, exists := cli.wallets[walletName]
	if !exists {
		return "", fmt.Errorf("Wallet %s not found", walletName)
	}
	return wallet.GetAddress(), nil
}
