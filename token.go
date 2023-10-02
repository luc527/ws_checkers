package main

import (
	"crypto/rand"
	"encoding/hex"
)

type token string

func newToken() token {
	bytes := [48]byte{}
	rand.Reader.Read(bytes[:])
	return token(hex.EncodeToString(bytes[:]))
}
