package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/taekwondodev/crypto-simulator/internal/blockchain"
	"github.com/taekwondodev/crypto-simulator/internal/mempool"
	"github.com/taekwondodev/crypto-simulator/internal/p2p"
)

func main() {
	bc := blockchain.New()
	defer bc.Close()
	if err := bc.InitHeightIndex(); err != nil {
		panic(err)
	}

	mp := mempool.New(bc)
	bootstrapNodes := []string{"localhost:3000", "localhost:3001"}

	node := p2p.NewNode(":3000", bootstrapNodes, bc, mp)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go node.Start()

	// Wait for shutdown signal
	<-signals
	log.Println("Shutting down...")
	node.Stop()
}
