package main

import (
	"crypto/rand"
	"encoding/hex"
)

func newToken() string {
	bytes := [48]byte{}
	rand.Reader.Read(bytes[:])
	return hex.EncodeToString(bytes[:])
}
