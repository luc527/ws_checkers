package main

import (
	"encoding/json"
	"log"
	"os"
	"slices"

	"github.com/boltdb/bolt"
)

var db *bolt.DB

func init() {
	path := os.Getenv("DB_PATH")
	if path == "" {
		path = "./checkers.db"
	}
	log.Printf("opening database at %v", path)
	db = must(bolt.Open(path, 0600, nil))
}

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}

func loadValue(tx *bolt.Tx, key string, v any) error {
	bucket := tx.Bucket([]byte("checkers"))
	bytes := bucket.Get([]byte(key))
	if bytes == nil {
		return nil
	}
	if err := json.Unmarshal(bytes, v); err != nil {
		return err
	}
	return nil
}

func storeValue(tx *bolt.Tx, key string, v any) error {
	bucket := tx.Bucket([]byte("checkers"))
	bytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := bucket.Put([]byte(key), bytes); err != nil {
		return err
	}
	return nil
}

func addWebhook(db *bolt.DB, url string) ([]string, error) {
	var urls []string
	err := db.Update(func(tx *bolt.Tx) error {
		if err := loadValue(tx, "webhooks", &urls); err != nil {
			return err
		}
		if idx := slices.Index(urls, url); idx != -1 {
			return nil
		}
		urls = append(urls, url)
		if err := storeValue(tx, "webhooks", urls); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return urls, nil
}

func deleteWebhook(db *bolt.DB, url string) ([]string, error) {
	var urls []string
	err := db.Update(func(tx *bolt.Tx) error {
		if err := loadValue(tx, "webhooks", &urls); err != nil {
			return err
		}
		idx := slices.Index(urls, url)
		if idx == -1 {
			return nil
		}
		urls = slices.Delete(urls, idx, idx+1)
		if err := storeValue(tx, "webhooks", urls); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return urls, nil
}

func getWebhooks(db *bolt.DB) ([]string, error) {
	var urls []string
	err := db.Update(func(tx *bolt.Tx) error {
		if err := loadValue(tx, "webhooks", &urls); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return urls, nil
}

func clearWebhooks(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		return storeValue(tx, "webhooks", []string{})
	})
}
