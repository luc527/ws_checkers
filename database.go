package main

import (
	"encoding/json"
	"slices"
)

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}

func loadValue(tx transaction, key string, v any) error {
	bytes := tx.get([]byte(key))
	if bytes == nil {
		return nil
	}
	if err := json.Unmarshal(bytes, v); err != nil {
		return err
	}
	return nil
}

func storeValue(tx transaction, key string, v any) error {
	bytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := tx.put([]byte(key), bytes); err != nil {
		return err
	}
	return nil
}

func addWebhook(db store, url string) ([]string, error) {
	var urls []string
	err := db.update(func(tx transaction) error {
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

func deleteWebhook(db store, url string) ([]string, error) {
	var urls []string
	err := db.update(func(tx transaction) error {
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

func getWebhooks(db store) ([]string, error) {
	var urls []string
	err := db.update(func(tx transaction) error {
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

func clearWebhooks(db store) error {
	return db.update(func(tx transaction) error {
		return storeValue(tx, "webhooks", []string{})
	})
}
