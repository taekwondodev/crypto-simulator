package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/taekwondodev/crypto-simulator/internal/blockchain"
	"github.com/taekwondodev/crypto-simulator/internal/mempool"
	"github.com/taekwondodev/crypto-simulator/internal/p2p"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
	"github.com/taekwondodev/crypto-simulator/pkg/utxo"
	"github.com/taekwondodev/crypto-simulator/pkg/wallet"
)

type CLI struct {
	bc      *blockchain.Blockchain
	mp      *mempool.Mempool
	node    *p2p.Node
	wallets map[string]*wallet.Wallet
}

func NewCLI(bc *blockchain.Blockchain, mp *mempool.Mempool, node *p2p.Node) *CLI {
	cli := &CLI{
		bc:      bc,
		mp:      mp,
		node:    node,
		wallets: make(map[string]*wallet.Wallet),
	}

	if err := cli.LoadWallets(); err != nil {
		log.Println("Error loading wallets:", err)
	}
	return cli
}

func (cli *CLI) Run() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Blockchain CLI")
	fmt.Println("Type 'help' for commands")

	for {
		fmt.Print("> ")
		scanner.Scan()
		text := scanner.Text()

		if text == "" {
			continue
		}

		parts := strings.Fields(text)
		command := parts[0]

		switch command {
		case "exit", "quit":
			return
		case "help":
			cli.printHelp()
		case "createwallet":
			cli.createWallet()
		case "listwallet":
			cli.listWallets()
		case "balance":
			if len(parts) < 2 {
				fmt.Println("Usage: balance <wallet-name>")
				continue
			}
			cli.getBalance(parts[1])
		case "send":
			if len(parts) < 4 {
				fmt.Println("Usage: send <from-wallet> <to-wallet> <amount>")
				continue
			}
			amount, err := strconv.Atoi(parts[3])
			if err != nil {
				fmt.Println("Invalid amount:", err)
				continue
			}
			cli.sendCoins(parts[1], parts[2], amount)
		case "mine":
			cli.mineBlock()
		case "blockchain":
			cli.printBlockchain()
		case "mempool":
			cli.showMempool()
		case "peers":
			cli.listPeers()
		case "connect":
			if len(parts) < 2 {
				fmt.Println("Usage: connect <peer-address>")
				continue
			}
			cli.connectPeer(parts[1])
		default:
			fmt.Println("Unknown command. Type 'help' for commands")
		}
	}
}

func (cli *CLI) printHelp() {
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
}

func (cli *CLI) createWallet() {
	fmt.Print("Enter wallet name: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	name := scanner.Text()

	if name == "" {
		fmt.Println("Wallet name cannot be empty")
		return
	}

	if _, exists := cli.wallets[name]; exists {
		fmt.Println("Wallet with that name already exists")
		return
	}

	cli.wallets[name] = wallet.NewWallet()
	fmt.Println("Created new wallet:", name)
	fmt.Println("Address:", cli.wallets[name].GetAddress())

	if err := cli.SaveWallets(); err != nil {
		fmt.Println("Error saving wallet:", err)
	}
}

func (cli *CLI) listWallets() {
	if len(cli.wallets) == 0 {
		fmt.Println("No wallets created yet. Use 'createwallet' to create one.")
		return
	}

	fmt.Println("Wallets:")
	for name, w := range cli.wallets {
		fmt.Printf("  %s: %s\n", name, w.GetAddress())
	}
}

func (cli *CLI) getBalance(walletName string) {
	w, exists := cli.wallets[walletName]
	if !exists {
		fmt.Println("Wallet not found:", walletName)
		return
	}

	address := w.GetAddress()
	balance := cli.bc.GetBalance(address)
	fmt.Printf("Balance of %s: %d coins\n", walletName, balance)
}

func (cli *CLI) sendCoins(fromWallet, toWallet string, amount int) {
	from, fromExists := cli.wallets[fromWallet]
	if !fromExists {
		fmt.Println("Source wallet not found:", fromWallet)
		return
	}

	to, toExists := cli.wallets[toWallet]
	if !toExists {
		fmt.Println("Destination wallet not found:", toWallet)
		return
	}

	fromAddress := from.GetAddress()
	toAddress := to.GetAddress()

	// Get UTXOs for the sender
	utxos := cli.bc.GetUTXOs(fromAddress)

	// Calculate total available balance
	balance := 0
	for _, utxo := range utxos {
		balance += utxo.Output.Value
	}

	if balance < amount {
		fmt.Printf("Not enough balance. Available: %d, Required: %d\n", balance, amount)
		return
	}

	// Create transaction inputs from UTXOs
	var inputs []utxo.TxInput
	var collected int

	for _, u := range utxos {
		input := utxo.TxInput{
			TxID:     u.TxID,
			OutIndex: u.Index,
			PubKey:   from.Address,
		}
		input.Sign(from.PrivateKey)

		inputs = append(inputs, input)
		collected += u.Output.Value

		if collected >= amount {
			break
		}
	}

	// Create transaction outputs
	var outputs []utxo.TxOutput

	// Output to recipient
	outputs = append(outputs, utxo.TxOutput{
		Value:      amount,
		PubKeyHash: []byte(toAddress),
	})

	// Create change output if needed
	if collected > amount {
		outputs = append(outputs, utxo.TxOutput{
			Value:      collected - amount,
			PubKeyHash: []byte(fromAddress),
		})
	}

	// Create and sign transaction
	tx := transaction.New(inputs, outputs)

	// Add to mempool
	cli.mp.Add(tx)

	// Broadcast to network
	txMsg := p2p.NewTxMessage(tx.Serialize())
	cli.node.Broadcast(txMsg)

	fmt.Printf("Transaction created: %x\n", tx.ID)
	fmt.Printf("Sent %d coins from %s to %s\n", amount, fromWallet, toWallet)
	fmt.Println("Transaction is in mempool, waiting to be mined")
}

func (cli *CLI) mineBlock() {
	// Get transactions from mempool
	txs := cli.mp.Flush()

	if len(txs) == 0 {
		fmt.Println("No transactions in mempool to mine")
		return
	}

	fmt.Printf("Mining new block with %d transactions...\n", len(txs))

	// Add new block to blockchain
	newBlock := cli.bc.AddBlock(txs)

	// Broadcast the block to the network
	blockMsg := p2p.NewBlockMessage(newBlock.Serialize())
	cli.node.Broadcast(blockMsg)

	fmt.Printf("Block mined! Hash: %x, Height: %d\n", newBlock.Hash, newBlock.Height)
}

func (cli *CLI) printBlockchain() {
	height := cli.bc.CurrentHeight()
	fmt.Printf("Blockchain height: %d\n", height)

	fmt.Println("Latest blocks:")
	for i := height; i >= 0 && i > height-10; i-- {
		block := cli.bc.GetBlockAtHeight(i)
		if block == nil {
			continue
		}

		fmt.Printf("Block %d: %x\n", block.Height, block.Hash)
		fmt.Printf("  Transactions: %d\n", len(block.Transactions))
		fmt.Printf("  Timestamp: %s\n", block.Timestamp)
		fmt.Printf("  Difficulty: %d\n", block.Difficulty)
		fmt.Println()
	}
}

func (cli *CLI) showMempool() {
	count := cli.mp.Count()

	if count == 0 {
		fmt.Println("Mempool is empty")
		return
	}

	fmt.Printf("Mempool has %d transaction(s)\n", count)
}

func (cli *CLI) listPeers() {
	peers := cli.node.GetPeers()

	if len(peers) == 0 {
		fmt.Println("No peers connected")
		return
	}

	fmt.Println("Connected peers:")
	for _, peer := range peers {
		fmt.Printf("  %s (last seen: %s)\n", peer.Address, peer.LastSeen.Format("15:04:05"))
	}
}

func (cli *CLI) connectPeer(address string) {
	fmt.Printf("Connecting to peer %s...\n", address)
	cli.node.Connect(address)
	fmt.Println("Connection initiated")
}
