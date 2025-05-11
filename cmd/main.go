package main

import (
	"github.com/taekwondodev/crypto-simulator/internal/app"
	"github.com/taekwondodev/crypto-simulator/internal/config"
)

func main() {
	cfg := config.LoadFromFlags()

	application := app.New(cfg)
	application.Start()
	defer application.Shutdown()
}
