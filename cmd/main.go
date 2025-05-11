package main

import (
	"flag"
	"strings"
	"time"

	"github.com/taekwondodev/crypto-simulator/internal/app"
)

func main() {
	var interactive bool
	var port string
	var miningInterval int

	flag.BoolVar(&interactive, "interactive", false, "Run in interactive mode")
	flag.StringVar(&port, "port", ":3000", "Port to listen on (default :3000)")
	flag.IntVar(&miningInterval, "mining-interval", 60, "Seconds between mining attempts in non-interactive mode")
	flag.Parse()

	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	config := app.Config{
		Interactive:    interactive,
		Port:           port,
		MiningInterval: time.Duration(miningInterval) * time.Second,
	}

	application := app.New(config)
	application.Start()
	defer application.Shutdown()
}
