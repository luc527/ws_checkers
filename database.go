package main

import (
	"bytes"
	"encoding/json"
	"log"
	"slices"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
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

func gameKey(mode gameMode, id uuid.UUID) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := buf.WriteString("game"); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(mode.String()); err != nil {
		return nil, err
	}
	if _, err := buf.Write(id[:]); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func savePlyHistory(db store, mode gameMode, id uuid.UUID, history []core.Ply) error {
	log.Printf("saving ply history (mode %v, id %v)", mode, id)
	err := db.update(func(tx transaction) error {
		key, err := gameKey(mode, id)
		if err != nil {
			return err
		}
		val, err := json.Marshal(history)
		if err != nil {
			return err
		}
		if err := tx.put(key, val); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Printf("failed to save game history (mode %v, id %v)", mode, id)
	}
	return err
}

func getPlyHistory(db store, mode gameMode, id uuid.UUID) ([]core.Ply, error) {
	var history []core.Ply
	err := db.update(func(tx transaction) error {
		key, err := gameKey(mode, id)
		if err != nil {
			return err
		}
		val := tx.get(key)
		if err := json.Unmarshal(val, &history); err != nil {
			return err
		}
		return nil
	})
	return history, err
}

func getGameIds(db store, mode gameMode) ([]uuid.UUID, error) {
	prefix := []byte("game")
	prefix = append(prefix, []byte(mode.String())...)

	var ids []uuid.UUID

	err := db.view(func(tx transaction) error {
		c := tx.cursor()
		for k, _ := c.seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.next() {
			n := len(k)
			id, err := uuid.FromBytes(k[n-16 : n])
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	})

	return ids, err
}
