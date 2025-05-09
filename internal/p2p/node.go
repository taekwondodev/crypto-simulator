package p2p

import (
	"errors"
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

	// Start a goroutine to save peer list periodically
	go func() {
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
	}()

	// rate limit connections
	sem := make(chan struct{}, maxConnections)

	// Accept connections until we're shutting down
	go func() {
		for {
			sem <- struct{}{}

			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Accept error: %v", err)
				<-sem
				continue
			}

			go func() {
				defer func() { <-sem }()
				n.handleConnection(conn)
			}()
		}
	}()

	// Wait for shutdown signal
	<-n.done
	n.SavePeers() // Final save before shutdown
}

// Stop gracefully shuts down the node
func (n *Node) Stop() {
	close(n.done)
}

func (n *Node) handleConnection(conn net.Conn) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(readTimeout))

	// Aggiungi timer per ping
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	// Run pinger in background
	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case <-pingTicker.C:
				if err := n.sendPing(conn); err != nil {
					log.Printf("Error sending ping: %v", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// 1. Handshake
	if err := n.performHandshake(conn); err != nil {
		log.Printf("Handshake failed: %v", err)
		return
	}

	// 2. Add the peer to our peer list
	peerAddr := conn.RemoteAddr().String()
	n.mu.Lock()
	n.Peers[peerAddr] = &Peer{
		Address:    peerAddr,
		Connection: conn,
		LastSeen:   time.Now(),
	}
	n.mu.Unlock()
	// 3. Gestione messaggi in loop
	buf := make([]byte, 4096)
	for {
		nBytes, err := conn.Read(buf)
		if err != nil {
			log.Printf("Read error from %s: %v", peerAddr, err)
			break
		}

		msg, err := DeserializeMessage(buf[:nBytes])
		if err != nil {
			log.Printf("Error deserializing message: %v", err)
			continue
		}

		// Reset read deadline on successful message
		conn.SetReadDeadline(time.Now().Add(readTimeout))

		// Update last seen time
		n.mu.Lock()
		if peer, ok := n.Peers[peerAddr]; ok {
			peer.LastSeen = time.Now()
		}
		n.mu.Unlock()

		// Process the message
		if err := n.handleMessage(msg, conn); err != nil {
			log.Printf("Error handling message: %v", err)
			continue
		}
	}

	// Clean up peer on disconnect
	n.removePeer(peerAddr)
}

func (n *Node) performHandshake(conn net.Conn) error {
	// Invia nostro messaggio di version
	versionMsg := &Message{
		Version:   0x01,
		Type:      MsgVersion,
		Timestamp: time.Now().Unix(),
		Payload:   []byte(n.Address),
	}
	data, err := versionMsg.Serialize()
	if err != nil {
		return err
	}

	if _, err := conn.Write(data); err != nil {
		return err
	}

	// Wait for their version message in response
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 4096)

	nBytes, err := conn.Read(buf)
	if err != nil {
		return err
	}

	msg, err := DeserializeMessage(buf[:nBytes])
	if err != nil {
		return err
	}

	// Verify it's a version message
	if msg.Type != MsgVersion {
		return errors.New("expected version message during handshake")
	}

	// Send verack
	verackMsg := &Message{
		Version:   0x01,
		Type:      MsgVerAck,
		Timestamp: time.Now().Unix(),
	}

	data, err = verackMsg.Serialize()
	if err != nil {
		return err
	}

	if _, err := conn.Write(data); err != nil {
		return err
	}

	return nil
}

func (n *Node) handleMessage(msg *Message, conn net.Conn) error {
	switch msg.Type {
	case MsgVersion:
		return n.handleVersion(msg, conn)
	case MsgVerAck:
		return n.handleVerAck(msg)
	case MsgPing:
		return n.handlePing(msg, conn)
	case MsgPong:
		return n.handlePong(msg)
	case MsgTx:
		return n.handleTransaction(msg)
	case MsgBlock:
		return n.handleBlock(msg)
	case MsgGetBlocks:
		return n.handleGetBlocks(msg, conn)
	case MsgInv:
		return n.handleInventory(msg, conn)
	case MsgGetData:
		return n.handleGetData(msg, conn)
	case MsgAddr:
		return n.handleAddress(msg)
	case MsgGetAddr:
		return n.handleGetAddr(msg, conn)
	default:
		return errors.New("unknown message type")
	}
}

func (n *Node) Broadcast(msg *Message) {
	data, err := msg.Serialize()
	if err != nil {
		log.Printf("Error serializing message: %v", err)
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	for addr, peer := range n.Peers {
		peer.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_, err := peer.Connection.Write(data)
		if err != nil {
			log.Printf("Error sending to %s: %v", addr, err)
			n.removePeer(addr)
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

func (n *Node) Connect(addr string) {
	// Don't connect if already connected
	n.mu.Lock()
	if _, exists := n.Peers[addr]; exists {
		n.mu.Unlock()
		return
	}
	n.mu.Unlock()

	// Connect with timeout
	conn, err := net.DialTimeout("tcp", addr, connectTimeout)
	if err != nil {
		log.Printf("Connection to %s failed: %v", addr, err)
		return
	}

	// Handle new connection in a separate goroutine
	go n.handleConnection(conn)
}

// non c'è
func (n *Node) sendPing(conn net.Conn) error {
	pingMsg := &Message{
		Version:   0x01,
		Type:      MsgPing,
		Timestamp: time.Now().Unix(),
	}

	data, err := pingMsg.Serialize()
	if err != nil {
		return err
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = conn.Write(data)
	return err
}
