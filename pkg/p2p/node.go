package p2p

import (
	"net"
	"sync"
)

type Node struct {
	Address string
	Peers   map[string]net.Conn
	mu      sync.Mutex
}

func NewNode(address string) *Node {
	return &Node{
		Address: address,
		Peers:   make(map[string]net.Conn),
	}
}

func (n *Node) Start() {
	listener, _ := net.Listen("tcp", n.Address)
	defer listener.Close()

	for {
		conn, _ := listener.Accept()
		go n.handleConnection(conn)
	}
}

func (n *Node) Connect(peerAddress string) {
	conn, _ := net.Dial("tcp", peerAddress)
	n.mu.Lock()
	n.Peers[peerAddress] = conn
	n.mu.Unlock()
	go n.listenToPeer(conn)
}

func (n *Node) handleConnection(conn net.Conn) {
	// Gestione messaggi in entrata
}

func (n *Node) Broadcast(data []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, peer := range n.Peers {
		peer.Write(data)
	}
}
