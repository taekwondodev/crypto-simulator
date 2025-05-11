package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/taekwondodev/crypto-simulator/internal/blockchain"
	"github.com/taekwondodev/crypto-simulator/internal/cli"
	"github.com/taekwondodev/crypto-simulator/internal/config"
	"github.com/taekwondodev/crypto-simulator/internal/mempool"
	"github.com/taekwondodev/crypto-simulator/internal/p2p"
	"github.com/taekwondodev/crypto-simulator/pkg/transaction"
)

type App struct {
	config     *config.Config
	blockchain *blockchain.Blockchain
	mempool    *mempool.Mempool
	node       *p2p.Node
	signals    chan os.Signal
	stopMining chan struct{}
}

func New(config *config.Config) *App {
	bc := blockchain.New(config.DatabasePath)
	if err := bc.InitHeightIndex(); err != nil {
		panic(err)
	}

	mp := mempool.New(bc)
	node := p2p.NewNode(config.Port, config.BootstrapNodes, bc, mp)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	return &App{
		config:     config,
		blockchain: bc,
		mempool:    mp,
		node:       node,
		signals:    signals,
		stopMining: make(chan struct{}),
	}
}

func (a *App) Start() {
	go a.node.Start()
	fmt.Printf("Node started on %s\n", a.config.Port)

	if a.config.Interactive {
		a.startInteractiveMode()
	} else {
		a.startAutomaticMode()
	}
}

func (a *App) Shutdown() {
	a.node.Stop()
	a.blockchain.Close()
}

func (a *App) startInteractiveMode() {
	cliHandler := cli.NewCLI(a.blockchain, a.mempool, a.node)
	go cliHandler.Run()

	<-a.signals
	fmt.Println("\nShutting down...")
}

func (a *App) startAutomaticMode() {
	fmt.Println("Running in non-interactive mode")
	fmt.Printf("Mining blocks automatically every %d seconds\n",
		int(a.config.MiningInterval.Seconds()))
	fmt.Println("Use 'go run cmd/main.go -interactive -port=3000' for CLI mode")

	go a.automaticMining()

	<-a.signals
	fmt.Println("\nShutting down...")
	close(a.stopMining)
}

func (a *App) automaticMining() {
	ticker := time.NewTicker(a.config.MiningInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.mineNewBlock()
		case <-a.stopMining:
			return
		}
	}
}

func (a *App) mineNewBlock() {
	txs := a.mempool.Flush()

	// If mempool is empty, create a coinbase transaction
	if len(txs) == 0 {
		coinbase := transaction.NewCoinBaseTx("MiningReward", 50)
		txs = []*transaction.Transaction{coinbase}
		fmt.Println("Mining empty block with coinbase transaction only")
	} else {
		fmt.Printf("Mining new block with %d transactions from mempool\n", len(txs))
	}

	newBlock := a.blockchain.AddBlock(txs)
	blockMsg := p2p.NewBlockMessage(newBlock.Serialize())
	a.node.Broadcast(blockMsg)

	fmt.Printf("Block mined! Hash: %x, Height: %d, Transactions: %d\n",
		newBlock.Hash, newBlock.Height, len(newBlock.Transactions))
}
