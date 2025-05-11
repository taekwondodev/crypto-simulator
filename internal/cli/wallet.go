package cli

import (
	"log"

	"github.com/taekwondodev/crypto-simulator/pkg/wallet"
	"go.etcd.io/bbolt"
)

const walletBucket = "wallets"

func (cli *CLI) SaveWallets() error {
	return cli.bc.Db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(walletBucket))
		if err != nil {
			return err
		}

		storeWallets(b, cli.wallets)

		return nil
	})
}

func (cli *CLI) LoadWallets() error {
	return cli.bc.Db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(walletBucket))
		if b == nil {
			// No wallets saved yet
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			return getWallet(k, v, cli.wallets)
		})
	})
}

func storeWallets(b *bbolt.Bucket, wallets map[string]*wallet.Wallet) error {
	for name, w := range wallets {
		data, err := w.Serialize()
		if err != nil {
			return err
		}

		err = b.Put([]byte(name), data)
		if err != nil {
			return err
		}
	}

	return nil
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
