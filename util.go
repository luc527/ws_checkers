package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"testing"
)

// shortcut for json object
type jo map[string]any

func tmarshal(x any, t *testing.T) []byte {
	if bs, err := json.Marshal(x); err != nil {
		t.Logf("failed to marshal %v, error: %v\n", x, err)
		t.Fail()
		return nil
	} else {
		return bs
	}
}

func tunmarshal(s string, x any, t *testing.T) {
	if err := json.Unmarshal([]byte(s), x); err != nil {
		t.Logf("failed to unmarshal %v, error: %v\n", s, err)
		t.Fail()
	}
}

func generateToken() (string, error) {
	var bs [36]byte
	if _, err := rand.Read(bs[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bs[:]), nil
}
