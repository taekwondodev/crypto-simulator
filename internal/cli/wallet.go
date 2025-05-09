package cli

import (
	"encoding/json"
	"log"

	"github.com/taekwondodev/crypto-simulator/pkg/wallet"
	"go.etcd.io/bbolt"
)

const walletBucket = "wallets"

// SaveWallets saves all wallets to the database
func (cli *CLI) SaveWallets() error {
	return cli.bc.Db.Update(func(tx *bbolt.Tx) error {
		// Create wallets bucket if it doesn't exist
		b, err := tx.CreateBucketIfNotExists([]byte(walletBucket))
		if err != nil {
			return err
		}

		// Store each wallet
		for name, w := range cli.wallets {
			// Serialize wallet data
			walletData := struct {
				PrivateKeyD []byte `json:"private_key_d"`
				PrivateKeyX []byte `json:"private_key_x"`
				PrivateKeyY []byte `json:"private_key_y"`
				Address     []byte `json:"address"`
			}{
				PrivateKeyD: w.PrivateKey.D.Bytes(),
				PrivateKeyX: w.PrivateKey.PublicKey.X.Bytes(),
				PrivateKeyY: w.PrivateKey.PublicKey.Y.Bytes(),
				Address:     w.Address,
			}

			// Convert to JSON (still needs encryption in production)
			data, err := json.Marshal(walletData)
			if err != nil {
				return err
			}

			// Store in database
			err = b.Put([]byte(name), data)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// LoadWallets loads wallets from the database
func (cli *CLI) LoadWallets() error {
	return cli.bc.Db.View(func(tx *bbolt.Tx) error {
		// Check if wallets bucket exists
		b := tx.Bucket([]byte(walletBucket))
		if b == nil {
			// No wallets saved yet
			return nil
		}

		// Iterate over all wallets
		return b.ForEach(func(k, v []byte) error {
			name := string(k)

			// Parse wallet data from JSON
			var walletData struct {
				PrivateKeyD []byte `json:"private_key_d"`
				PrivateKeyX []byte `json:"private_key_x"`
				PrivateKeyY []byte `json:"private_key_y"`
				Address     []byte `json:"address"`
			}

			if err := json.Unmarshal(v, &walletData); err != nil {
				log.Printf("Error loading wallet %s: %v", name, err)
				return nil // Skip this wallet but continue loading others
			}

			// Reconstruct wallet (simplified, would need actual key reconstruction)
			w := wallet.NewWalletFromKeys(walletData.PrivateKeyD,
				walletData.PrivateKeyX,
				walletData.PrivateKeyY,
				walletData.Address)

			cli.wallets[name] = w
			return nil
		})
	})
}
