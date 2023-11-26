package main

import (
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
)

type transaction interface {
	get([]byte) []byte
	put([]byte, []byte) error
}

type store interface {
	update(func(transaction) error) error
	view(func(transaction) error) error
}

type notifyWebhooksFunc func(mode string, id uuid.UUID, urls []string, state gameState)

// Initialization

var db store
var notifyWebhooksImpl notifyWebhooksFunc

func init() {
	testing := os.Getenv("CHECKERS_TESTING") == "1"
	if testing {
		fmt.Println("initializing test database")
		db = &memStore{}
		notifyWebhooksImpl = notifyWebhooksDummy
	} else {
		path := os.Getenv("DB_PATH")
		if path == "" {
			path = "./checkers.db"
		}
		log.Printf("opening database at %v", path)

		boltObj := must(bolt.Open(path, 0600, nil))
		db = storeFromBolt(boltObj)

		err := boltObj.Update(func(tx *bolt.Tx) error {
			if _, err := tx.CreateBucketIfNotExists([]byte("checkers")); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Fatalf("failed to initialize the database: %v", err)
		} else {
			log.Println("database initialized successfully")
		}

		notifyWebhooksImpl = notifyWebhooksReal
	}
}

// BoltDB implementation

var _ store = boltStore{}
var _ transaction = boltTransaction{}

type boltStore struct {
	db *bolt.DB
}

type boltTransaction struct {
	tx *bolt.Tx
}

func storeFromBolt(db *bolt.DB) store {
	return boltStore{db}
}

func (bs boltStore) update(fn func(transaction) error) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		return fn(txFromBolt(tx))
	})
}

func (bs boltStore) view(fn func(transaction) error) error {
	return bs.db.View(func(tx *bolt.Tx) error {
		return fn(txFromBolt(tx))
	})
}

func (bt boltTransaction) bucket() *bolt.Bucket {
	return bt.tx.Bucket([]byte("checkers"))
}

func txFromBolt(tx *bolt.Tx) boltTransaction {
	return boltTransaction{tx}
}

func (bt boltTransaction) get(key []byte) []byte {
	return bt.bucket().Get(key)
}

func (bt boltTransaction) put(key []byte, val []byte) error {
	return bt.bucket().Put(key, val)
}

// In-memory implementation

type memStore struct {
	ks [][]byte
	vs [][]byte
}

var _ store = &memStore{}
var _ transaction = &memStore{}

func (ms *memStore) get(key []byte) []byte {
	for i, k := range ms.ks {
		if slices.Equal(k, key) {
			return ms.vs[i]
		}
	}
	return nil
}

func (ms *memStore) put(key []byte, val []byte) error {
	idx := slices.IndexFunc(ms.ks, func(b []byte) bool {
		return slices.Equal(b, key)
	})
	if idx == -1 {
		ms.ks = append(ms.ks, key)
		ms.vs = append(ms.vs, val)
	} else {
		ms.vs[idx] = val
	}
	return nil
}

func (ms *memStore) update(fn func(transaction) error) error {
	return fn(ms)
}

func (ms *memStore) view(fn func(transaction) error) error {
	return fn(ms)
}

func notifyWebhooksDummy(mode string, id uuid.UUID, urls []string, state gameState) {
	for _, url := range urls {
		log.Printf("(dummy) notifying %v (game result: %v, game id: %v)", url, state.result, id)
	}
}
