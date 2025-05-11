package cli

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"

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
}

func NewCLI(bc *blockchain.Blockchain, mp *mempool.Mempool, node *p2p.Node) *CLI {
	cli := &CLI{
		bc:      bc,
		mp:      mp,
		node:    node,
		wallets: make(map[string]*Wallet),
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

	for {
		input := askFor("> ")
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		if command == "exit" || command == "quit" {
			return
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
	fmt.Println("  createwallet         - Create a new wallet")
	fmt.Println("  listwallet           - List all wallets")
	fmt.Println("  balance <name>       - Get balance for wallet")
	fmt.Println("  send <from> <to> <amount> - Send coins from one wallet to another")
	fmt.Println("  mine                 - Mine a new block with transactions from mempool")
	fmt.Println("  blockchain           - Print the blockchain")
	fmt.Println("  mempool              - Show transactions in mempool")
	fmt.Println("  peers                - List connected peers")
	fmt.Println("  connect <address>    - Connect to a peer")
	fmt.Println("  exit                 - Exit the program")
	fmt.Println("  tx <txid>            - View transaction details")
	return nil
}

func (cli *CLI) createWallet() error {
	name := askFor("Enter wallet name: ")
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
		return fmt.Errorf("Usage: send <from-wallet> <to-wallet> <amount>")
	}

	fromName := parts[1]
	toName := parts[2]

	amount, err := parseAmount(parts[3])
	if err != nil {
		return err
	}

	from, err := handleWalletLookupError(cli.wallets, fromName)
	if err != nil {
		return err
	}

	to, err := handleWalletLookupError(cli.wallets, toName)
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

	tx, err := createTransaction(from, to, amount, utxos)
	if err != nil {
		return err
	}

	cli.mp.Add(tx)
	serialize, err := tx.Serialize()
	if err != nil {
		return err
	}
	txMsg := p2p.NewTxMessage(serialize)
	cli.node.Broadcast(txMsg)

	fmt.Printf("Transaction created: %x\n", tx.ID)
	fmt.Printf("Sent %d coins from %s to %s\n", amount, fromName, toName)
	fmt.Println("Transaction is in mempool, waiting to be mined")
	return nil
}

func (cli *CLI) mineBlock() error {
	// Get transactions from mempool
	txs := cli.mp.Flush()

	if len(txs) == 0 {
		return fmt.Errorf("No transactions in mempool to mine")
	}

	fmt.Printf("Mining new block with %d transactions...\n", len(txs))

	newBlock, err := cli.bc.AddBlock(txs)
	if err != nil {
		return err
	}
	serialize, err := newBlock.Serialize()
	if err != nil {
		return err
	}
	blockMsg := p2p.NewBlockMessage(serialize)
	cli.node.Broadcast(blockMsg)

	fmt.Printf("Block mined! Hash: %x, Height: %d\n", newBlock.Hash, newBlock.Height)
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
		fmt.Printf("Mempool is empty")
	}

	fmt.Printf("Mempool has %d transaction(s)\n", count)
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
