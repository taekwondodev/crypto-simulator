<div align="center">

# Crypto-simulator CLI

<img alt="Go" src="https://img.shields.io/badge/Go-1.24.3+-00ADD8?logo=go">
<img alt="MIT License" src="https://img.shields.io/badge/License-MIT-yellow.svg">

A blockchain simulator that implements the core concepts of a cryptocurrency network including P2P networking, mining, transactions, and wallet management.

</div>

## Features

- **Blockchain Implementation**: Proof-of-work consensus with dynamic difficulty adjustment

- **Full P2P Network**: Decentralized node discovery and synchronization

- **UTXO Model**: Transaction model similar to Bitcoin

- **Wallet Management**: Create and manage wallets with public/private key pairs

- **Transaction Pool**: Memory pool for pending transactions

- **CLI Interface**: Interactive command-line interface for node management

- **Automatic Mining**: Background mining in non-interactive mode

- **Deterministic Serialization**: Binary encoding ensures consistent blockchain state across nodes

- **Advanced Fork Detection**: Robust mechanisms for identifying and resolving chain forks

- **Performance Optimization**: UTXO caching for faster transaction validation and balance queries

## Technical Details

### Deterministic Binary Serialization
All blockchain data (blocks, transactions, UTXOs) use deterministic binary encoding rather than generic serialization formats like GOB. This ensures that identical data produces identical byte representations across different nodes, preventing hash mismatches during block validation.

### Fork Management
The system includes sophisticated fork detection and resolution:
- Hash-based block locators for efficiently finding common ancestors
- Chain reorganization to handle longer competing chains
- Transaction restoration from orphaned blocks back to mempool
- Fork DB isolation to safely evaluate alternative chains

### UTXO Management
- In-memory LRU cache for frequently accessed UTXOs to reduce database reads
- Efficient UTXO indexing for fast balance calculations and transaction verification
- Proper handling of spent outputs during chain reorganization

### Address Handling
- SHA-256 hashing of public keys for address generation
- Hex-encoding for human-readable address display
- Binary representation for internal storage efficiency
- ECDSA-based transaction signing and verification

## Installation

### Prerequisites

- [Go](https://go.dev/doc/install) (1.24.3 or higher)

### Building

```bash
git clone https://github.com/taekwondodev/crypto-simulator.git

cd crypto-simulator
touch blockchain.db
touch fork.db
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


## Architecture

![Project Structure](https://github.com/user-attachments/assets/78ec8b6d-2d24-4f03-8ca4-4a992e1143cd)

- **Blockchain**: Core data structure that stores blocks of transactions
- **Mempool**: Temporary storage for unconfirmed transactions
- **P2P Network**: Communication layer for node discovery and data exchange
- **UTXO Set**: Tracks unspent transaction outputs for efficient balance calculation
- **Wallets**: Manages cryptographic keys and addresses

## CLI Commands

| Command                                  | Description                                      |
|------------------------------------------|--------------------------------------------------|
| `help`                                   | Show available commands                          |
| `createwallet`                           | Create a new wallet                              |
| `listwallet`                             | List all wallets                                 |
| `balance <name>`                         | Get balance for wallet                           |
| `send <from> <to-address> <amount>`      | Send coins from one wallet to another            |
| `mine`                                   | Mine a new block with transactions from mempool  |
| `blockchain`                             | Print the blockchain                             |
| `mempool`                                | Show transactions in mempool                     |
| `peers`                                  | List connected peers                             |
| `connect <address>`                      | Connect to a peer                                |
| `tx <txid>`                              | View transaction details                         |

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
