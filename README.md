# Crypto-simulator

<img alt="Go" src="https://img.shields.io/badge/Go-1.24.3+-00ADD8?logo=go">

A blockchain simulator that implements the core concepts of a cryptocurrency network including P2P networking, mining, transactions, and wallet management.

## Features

- **Blockchain Implementation**: Proof-of-work consensus with dynamic difficulty adjustment

- **Full P2P Network**: Decentralized node discovery and synchronization

- **UTXO Model**: Transaction model similar to Bitcoin

- **Wallet Management**: Create and manage wallets with public/private key pairs

- **Transaction Pool**: Memory pool for pending transactions

- **CLI Interface**: Interactive command-line interface for node management

- **Automatic Mining**: Background mining in non-interactive mode

## Installation

### Prerequisites

- [Go](https://go.dev/doc/install)(1.24.3 or higher)

### Building

```bash
git clone https://github.com/taekwondodev/crypto-simulator.git

cd crypto-simulator
go mod download

go build -o crypto-simulator ./cmd
```

## Usage

### Start in Interactive Mode

```bash
./crypto-simulator -interactive -port=3000
```

### Start in Automatic Mining Mode

```bash
./crypto-simulator -port=3001 -mining-interval=30
```

### Available Flags

- `-interactive`: Enable interactive CLI mode  
- `-port`: Port to listen on (default: ":3000")  
- `-mining-interval`: Seconds between automatic mining attempts (default: 60)


## Architecture

┌───────────┐     ┌──────────┐     ┌──────────┐
│   Node    │◄───►│ Mempool  │◄───►│Blockchain│
└───────────┘     └──────────┘     └──────────┘
      ▲                                 ▲
      │                                 │
      ▼                                 │
┌───────────┐                      ┌──────────┐
│   P2P     │                      │  UTXO    │
│  Network  │                      │   Set    │
└───────────┘                      └──────────┘
      ▲
      │
      ▼
┌───────────┐
│  Wallets  │
└───────────┘

- **Blockchain**: Core data structure that stores blocks of transactions
- **Mempool**: Temporary storage for unconfirmed transactions
- **P2P Network**: Communication layer for node discovery and data exchange
- **UTXO Set**: Tracks unspent transaction outputs for efficient balance calculation
- **Wallets**: Manages cryptographic keys and addresses

## CLI Commands

| Command                          | Description                                      |
|----------------------------------|--------------------------------------------------|
| `help`                           | Show available commands                          |
| `createwallet`                   | Create a new wallet                              |
| `listwallet`                     | List all wallets                                 |
| `balance <name>`                 | Get balance for wallet                           |
| `send <from> <to> <amount>`      | Send coins from one wallet to another            |
| `mine`                           | Mine a new block with transactions from mempool  |
| `blockchain`                     | Print the blockchain                             |
| `mempool`                        | Show transactions in mempool                     |
| `peers`                          | List connected peers                             |
| `connect <address>`              | Connect to a peer                                |
| `tx <txid>`                      | View transaction details                         |

## Project Structure

```text
cmd/                   # Application entry points
internal/              # Private application code
  app/                 # Application coordinator
  blockchain/          # Core blockchain implementation
  cli/                 # Command-line interface
  config/              # Configuration handling
  mempool/             # Transaction memory pool
  p2p/                 # Peer-to-peer networking
pkg/                   # Public libraries
  block/               # Block structure and validation
  transaction/         # Transaction structure and handling
  utxo/                # Unspent transaction output model
  wallet/              # Cryptographic wallet implementation
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Inspired by blockchain technology and cryptocurrency systems
- Built with Go's powerful standard libraries and minimal dependencies
