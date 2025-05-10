package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/taekwondodev/crypto-simulator/internal/blockchain"
	"github.com/taekwondodev/crypto-simulator/internal/mempool"
)

const (
	readBufferSize  = 1024 * 1024
	writeBufferSize = 1024 * 1024
	maxConnections  = 1000
	pingInterval    = 2 * time.Minute
	readTimeout     = 5 * time.Minute
	connectTimeout  = 5 * time.Second
)

type Node struct {
	Address    string
	Peers      map[string]*Peer
	blockchain *blockchain.Blockchain
	mempool    *mempool.Mempool
	mu         sync.Mutex
	done       chan struct{}
}

type Peer struct {
	Address    string
	Connection net.Conn
	LastSeen   time.Time
}

func NewNode(address string, bootstrapNodes []string, bc *blockchain.Blockchain, mp *mempool.Mempool) *Node {
	n := &Node{
		Address:    address,
		Peers:      make(map[string]*Peer),
		blockchain: bc,
		mempool:    mp,
		done:       make(chan struct{}),
	}
	n.LoadPeers()

	if len(n.Peers) == 0 {
		for _, addr := range bootstrapNodes {
			if addr != address {
				n.Connect(addr)
			}
		}
	}
	return n
}

type tcpKeepAliveListener struct{ *net.TCPListener }

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, _ := ln.AcceptTCP()
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	tc.SetReadBuffer(readBufferSize)
	tc.SetWriteBuffer(writeBufferSize)
	return tc, nil
}

func (n *Node) Start() {
	listener, err := net.Listen("tcp", n.Address)
	if err != nil {
		log.Fatal("Listen error:", err)
	}
	defer listener.Close()
	listener = tcpKeepAliveListener{listener.(*net.TCPListener)}

	log.Printf("Node listening on %s", n.Address)

	go n.runPeerSaver()
	go n.acceptConnections(listener)

	<-n.done
	n.SavePeers() // Final save before shutdown
}

func (n *Node) Stop() {
	close(n.done)
}

func (n *Node) Broadcast(msg *Message) {
	n.mu.Lock()
	defer n.mu.Unlock()

	for addr, peer := range n.Peers {
		if err := writeMessage(peer.Connection, msg); err != nil {
			logError(fmt.Sprintf("Failed to broadcast to %s", addr), err)
			n.removePeerLocked(addr)
		} else {
			logMessageSent(msg.Type, addr)
		}
	}
}

func (n *Node) Connect(addr string) {
	// Don't connect if already connected
	n.mu.Lock()
	if _, exists := n.Peers[addr]; exists {
		n.mu.Unlock()
		return
	}
	n.mu.Unlock()

	conn, err := net.DialTimeout("tcp", addr, connectTimeout)
	if err != nil {
		log.Printf("Connection to %s failed: %v", addr, err)
		return
	}

	go n.handleConnection(conn)
}

func (n *Node) GetPeers() []Peer {
	n.mu.Lock()
	defer n.mu.Unlock()

	peers := make([]Peer, 0, len(n.Peers))
	for _, p := range n.Peers {
		peers = append(peers, *p)
	}
	return peers
}

/*********************************************************************************************/

func (n *Node) runPeerSaver() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			n.SavePeers()
		case <-n.done:
			return
		}
	}
}

func (n *Node) acceptConnections(listener net.Listener) {
	sem := make(chan struct{}, maxConnections)

	for {
		sem <- struct{}{}

		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			<-sem

			select {
			case <-n.done:
				return
			default:
				// Continue accepting if not shutting down
			}
			continue
		}

		go func() {
			defer func() { <-sem }()
			n.handleConnection(conn)
		}()
	}
}

func (n *Node) handleConnection(conn net.Conn) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(readTimeout))

	peerAddr := conn.RemoteAddr().String()

	pingDone := n.setupPingKeepAlive(conn)
	defer close(pingDone)

	if err := n.performHandshake(conn); err != nil {
		log.Printf("Handshake failed: %v", err)
		return
	}

	n.registerPeer(conn, peerAddr)

	n.processMessages(conn, peerAddr)

	n.removePeer(peerAddr)
}

func (n *Node) setupPingKeepAlive(conn net.Conn) chan struct{} {
	done := make(chan struct{})
	pingTicker := time.NewTicker(pingInterval)

	go func() {
		defer pingTicker.Stop()

		for {
			select {
			case <-pingTicker.C:
				if err := sendPingMessage(conn); err != nil {
					log.Printf("Ping failed: %v", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	return done
}

func (n *Node) registerPeer(conn net.Conn, addr string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.Peers[addr] = &Peer{
		Address:    addr,
		Connection: conn,
		LastSeen:   time.Now(),
	}

	log.Printf("Peer registered: %s", addr)
}

func (n *Node) performHandshake(conn net.Conn) error {
	if err := sendVersionMessage(conn, n.Address); err != nil {
		return err
	}

	msg, err := readMessage(conn, 10*time.Second)
	if err != nil {
		return err
	}

	if msg.Type != MsgVersion {
		return errors.New("expected version message during handshake")
	}

	return sendVerAckMessage(conn)
}

func (n *Node) updatePeerLastSeen(addr string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if peer, ok := n.Peers[addr]; ok {
		peer.LastSeen = time.Now()
	}
}

func (n *Node) processMessages(conn net.Conn, peerAddr string) {
	for {
		msg, err := readMessage(conn, readTimeout)
		if err != nil {
			log.Printf("Read error from %s: %v", peerAddr, err)
			break
		}

		n.updatePeerLastSeen(peerAddr)

		logMessageReceived(msg.Type, peerAddr)

		if err := n.handleMessage(msg, conn); err != nil {
			log.Printf("Error handling message from %s: %v", peerAddr, err)
		}
	}
}

func (n *Node) removePeer(addr string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if peer, exists := n.Peers[addr]; exists {
		peer.Connection.Close()
		delete(n.Peers, addr)
	}
}

func (n *Node) removePeerLocked(addr string) {
	if peer, exists := n.Peers[addr]; exists {
		peer.Connection.Close()
		delete(n.Peers, addr)
	}
}
