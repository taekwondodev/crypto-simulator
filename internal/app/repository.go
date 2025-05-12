package app

import (
	"log"

	"github.com/taekwondodev/crypto-simulator/pkg/wallet"
	"go.etcd.io/bbolt"
)

const walletBucket = "wallets"

func (a *App) LoadWallets() error {
	return a.blockchain.Db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(walletBucket))
		if b == nil {
			// No wallets saved yet
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			return getWallet(k, v, a.wallets)
		})
	})
}

func getWallet(k, v []byte, wallets map[string]*wallet.Wallet) error {
	name := string(k)

	w, err := wallet.DeserializeWallet(v)
	if err != nil {
		log.Printf("Error deserializing wallet %s: %v", name, err)
		return nil
	}

	wallets[name] = w
	return nil
}
