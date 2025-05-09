package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/taekwondodev/crypto-simulator/internal/blockchain"
	"github.com/taekwondodev/crypto-simulator/internal/cli"
	"github.com/taekwondodev/crypto-simulator/internal/mempool"
	"github.com/taekwondodev/crypto-simulator/internal/p2p"
)

func main() {
	// Parse command line flags
	interactive := len(os.Args) > 1 && os.Args[1] == "interactive"

	// Initialize blockchain
	bc := blockchain.New()
	defer bc.Close()

	if err := bc.InitHeightIndex(); err != nil {
		panic(err)
	}

	// Initialize mempool
	mp := mempool.New(bc)

	// Define bootstrap nodes
	bootstrapNodes := []string{"localhost:3000", "localhost:3001"}

	// Create P2P node
	portStr := ":3000" // Default port
	if len(os.Args) > 2 {
		portStr = os.Args[2] // Allow custom port
	}

	node := p2p.NewNode(portStr, bootstrapNodes, bc, mp)

	// Set up signal handling for graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Start P2P node
	go node.Start()

	fmt.Printf("Node started on %s\n", portStr)

	if interactive {
		// Start interactive CLI
		cli := cli.NewCLI(bc, mp, node)
		go cli.Run()

		// Wait for shutdown signal
		<-signals
		fmt.Println("\nShutting down...")
	} else {
		fmt.Println("Running in non-interactive mode")
		fmt.Println("Use 'go run main.go interactive [port]' for CLI mode")

		// Start background tasks like mining
		// In non-interactive mode, you could automatically mine blocks periodically

		// Wait for shutdown signal
		<-signals
		fmt.Println("\nShutting down...")
	}

	node.Stop()
}
