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
	"github.com/taekwondodev/crypto-simulator/pkg/wallet"
)

type App struct {
	config     *config.Config
	blockchain *blockchain.Blockchain
	wallets    map[string]*wallet.Wallet
	mempool    *mempool.Mempool
	node       *p2p.Node
	signals    chan os.Signal
	stopMining chan struct{}
}

func New(config *config.Config) *App {
	bc := blockchain.New(config.DatabasePath)
	bc.InitHeightIndex()

	mp := mempool.New(bc)
	node := p2p.NewNode(config.Port, config.BootstrapNodes, bc, mp)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	a := &App{
		config:     config,
		blockchain: bc,
		wallets:    make(map[string]*wallet.Wallet),
		mempool:    mp,
		node:       node,
		signals:    signals,
		stopMining: make(chan struct{}),
	}

	if err := a.LoadWallets(); err != nil {
		fmt.Printf("Error loading wallets: %v\n", err)
	}
	return a
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

	select {
	case <-cliHandler.Done:
	case <-a.signals:
		close(cliHandler.Done)
	}
	fmt.Println("\nShutting down...")
}

func (a *App) startAutomaticMode() {
	fmt.Println("Running in non-interactive mode")
	fmt.Printf("Mining blocks automatically every %d seconds\n",
		int(a.config.MiningInterval.Seconds()))
	fmt.Println("Use './crypto-simulator -interactive for CLI mode")

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
			if err := a.mineNewBlock(); err != nil {
				fmt.Printf("Error mining block: %v\n", err)
			}
		case <-a.stopMining:
			return
		}
	}
}

func (a *App) mineNewBlock() error {
	minerAddress, err := a.handleMiningReward()
	if err != nil {
		return err
	}
	txs, err := a.mempool.HandleCoinbaseTxs(minerAddress, 50)
	if err != nil {
		return err
	}

	newBlock, err := a.blockchain.CreateBlock(txs)
	if err != nil {
		return err
	}

	if err := a.blockchain.AddBlock(newBlock); err != nil {
		return err
	}

	fmt.Printf("Block mined! Height: %d\n", newBlock.Height)

	invMsg := p2p.NewInvMessage([][]byte{newBlock.Hash})
	a.node.Broadcast(invMsg)

	return nil
}

func (a *App) handleMiningReward() (string, error) {
	if len(a.wallets) == 0 {
		return "", fmt.Errorf("No wallets available. Create a wallet first with 'createwallet'")
	}
	var firstWallet *wallet.Wallet

	for _, wallet := range a.wallets {
		firstWallet = wallet
		break
	}

	return firstWallet.GetAddress(), nil
}
