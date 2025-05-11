package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/taekwondodev/crypto-simulator/internal/blockchain"
	"github.com/taekwondodev/crypto-simulator/internal/cli"
	"github.com/taekwondodev/crypto-simulator/internal/mempool"
	"github.com/taekwondodev/crypto-simulator/internal/p2p"
)

func main() {
	var interactive bool
	var port string

	flag.BoolVar(&interactive, "interactive", false, "Run in interactive mode")
	flag.StringVar(&port, "port", ":3000", "Port to listen on (default :3000)")
	flag.Parse()

	portStr := port
	if !strings.HasPrefix(portStr, ":") {
		portStr = ":" + portStr
	}

	bc := blockchain.New()
	defer bc.Close()

	if err := bc.InitHeightIndex(); err != nil {
		panic(err)
	}

	mp := mempool.New(bc)
	bootstrapNodes := []string{"localhost:3000", "localhost:3001"}
	node := p2p.NewNode(portStr, bootstrapNodes, bc, mp)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go node.Start()

	fmt.Printf("Node started on %s\n", portStr)

	if interactive {
		cli := cli.NewCLI(bc, mp, node)
		go cli.Run()

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
