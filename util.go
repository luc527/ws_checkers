package main

import (
	"crypto/rand"
	"encoding/hex"
)

func genToken() (string, error) {
	bs := make([]byte, 36)
	if _, err := rand.Read(bs); err != nil {
		return "", err
	}
	return hex.EncodeToString(bs), nil
}
