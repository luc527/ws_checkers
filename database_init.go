package main

import (
	"bytes"
	"cmp"
	"log"
	"os"
	"slices"

	"github.com/boltdb/bolt"
)

type store interface {
	update(func(transaction) error) error
	view(func(transaction) error) error
}

type transaction interface {
	get([]byte) []byte
	put([]byte, []byte) error
	cursor() cursor
}

type cursor interface {
	seek([]byte) ([]byte, []byte)
	first() ([]byte, []byte)
	next() ([]byte, []byte)
}

// Initialization

var db store

func init() {
	testing := os.Getenv("CHECKERS_TESTING") == "1"
	if testing {
		log.Println("initializing test database")
		db = &memStore{}
	} else {
		path := os.Getenv("DB_PATH")
		if path == "" {
			path = "./data/checkers.db"
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
	}
}

// BoltDB implementation

var _ store = boltStore{}
var _ transaction = boltTransaction{}
var _ cursor = boltCursor{}

type boltStore struct {
	db *bolt.DB
}

type boltTransaction struct {
	tx *bolt.Tx
}

type boltCursor struct {
	c *bolt.Cursor
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

func (bt boltTransaction) cursor() cursor {
	return boltCursor{bt.bucket().Cursor()}
}

func (bc boltCursor) seek(b []byte) ([]byte, []byte) {
	return bc.c.Seek(b)
}

func (bc boltCursor) first() ([]byte, []byte) {
	return bc.c.First()
}

func (bc boltCursor) next() ([]byte, []byte) {
	return bc.c.Next()
}

// In-memory implementation

type memEntry struct {
	k []byte
	v []byte
}

type memStore struct {
	entries []memEntry
}

type memCursor struct {
	i  int
	ms *memStore
}

var _ store = &memStore{}
var _ transaction = &memStore{}
var _ cursor = &memCursor{}

func (ms *memStore) get(key []byte) []byte {
	for _, e := range ms.entries {
		if slices.Equal(e.k, key) {
			return e.v
		}
	}
	return nil
}

func compareBytes(a []byte, b []byte) int {
	for {
		if len(a) == 0 && len(b) == 0 {
			return 0
		}
		if len(a) == 0 {
			return -1
		}
		if len(b) == 0 {
			return 1
		}
		c := cmp.Compare(a[0], b[0])
		if c != 0 {
			return c
		}
		a, b = a[1:], b[1:]
	}
}

func (ms *memStore) put(key []byte, val []byte) error {
	idx := slices.IndexFunc(ms.entries, func(e memEntry) bool {
		return slices.Equal(e.k, key)
	})
	if idx == -1 {
		ms.entries = append(ms.entries, memEntry{key, val})
		slices.SortFunc(ms.entries, func(a memEntry, b memEntry) int {
			return compareBytes(a.k, b.k)
		})
	} else {
		ms.entries[idx].v = val
	}
	return nil
}

func (ms *memStore) cursor() cursor {
	return &memCursor{0, ms}
}

func (mc *memCursor) seek(prefix []byte) ([]byte, []byte) {
	idx := slices.IndexFunc(mc.ms.entries, func(e memEntry) bool {
		return bytes.HasPrefix(e.k, prefix)
	})
	if idx != -1 {
		e := mc.ms.entries[idx]
		mc.i = idx + 1
		return e.k, e.v
	} else {
		return nil, nil
	}
}

func (mc *memCursor) next() ([]byte, []byte) {
	if mc.i < len(mc.ms.entries) {
		e := mc.ms.entries[mc.i]
		mc.i++
		return e.k, e.v
	} else {
		return nil, nil
	}
}

func (mc *memCursor) first() ([]byte, []byte) {
	if len(mc.ms.entries) == 0 {
		return nil, nil
	}
	e := mc.ms.entries[0]
	mc.i = 1
	return e.k, e.v
}

func (ms *memStore) update(fn func(transaction) error) error {
	return fn(ms)
}

func (ms *memStore) view(fn func(transaction) error) error {
	return fn(ms)
}
