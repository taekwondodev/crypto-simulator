package p2p

import (
	"encoding/json"
	"log"
	"os"
)

const peersFile = "peers.json"

type StoredPeers struct {
	Addresses []string `json:"addresses"`
}

func (n *Node) SavePeers() {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Extract addresses from peers
	peerAddresses := make([]string, 0, len(n.Peers))
	for addr := range n.Peers {
		peerAddresses = append(peerAddresses, addr)
	}

	// Create data structure for storage
	data := StoredPeers{
		Addresses: peerAddresses,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Error marshaling peer data: %v", err)
		return
	}

	// Write to file
	err = os.WriteFile(peersFile, jsonData, 0644)
	if err != nil {
		log.Printf("Error writing peer file: %v", err)
	}
}

func (n *Node) LoadPeers() {
	// Check if peers file exists
	if _, err := os.Stat(peersFile); os.IsNotExist(err) {
		log.Println("No peers file found, starting with empty peer list")
		return
	}

	// Read file
	jsonData, err := os.ReadFile(peersFile)
	if err != nil {
		log.Printf("Error reading peers file: %v", err)
		return
	}

	// Parse JSON
	var data StoredPeers
	if err := json.Unmarshal(jsonData, &data); err != nil {
		log.Printf("Error parsing peers file: %v", err)
		return
	}

	// Connect to known peers
	for _, addr := range data.Addresses {
		if addr != n.Address { // Don't connect to ourselves
			go n.Connect(addr)
		}
	}
}
