package config

import "time"

type Config struct {
	Interactive    bool
	Port           string
	BootstrapNodes []string
	MiningInterval time.Duration
	MiningReward   int
	DatabasePath   string
}

func DefaultConfig() *Config {
	return &Config{
		Interactive:    false,
		Port:           ":3000",
		BootstrapNodes: []string{"localhost:3000", "localhost:3001"},
		MiningInterval: 60 * time.Second,
		MiningReward:   50,
		DatabasePath:   "blockchain.db",
	}
}
