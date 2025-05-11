package config

import (
	"flag"
	"strings"
	"time"
)

func LoadFromFlags() *Config {
	cfg := DefaultConfig()

	var miningInterval int
	flag.BoolVar(&cfg.Interactive, "interactive", cfg.Interactive, "Run in interactive mode")
	flag.StringVar(&cfg.Port, "port", cfg.Port, "Port to listen on")
	flag.IntVar(&miningInterval, "mining-interval", int(cfg.MiningInterval.Seconds()),
		"Seconds between mining attempts in non-interactive mode")

	flag.Parse()

	if !strings.HasPrefix(cfg.Port, ":") {
		cfg.Port = ":" + cfg.Port
	}

	cfg.MiningInterval = time.Duration(miningInterval) * time.Second

	return cfg
}
