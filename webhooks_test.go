package main

import (
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
)

func TestWebhookSend(t *testing.T) {
	id, err := uuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}

	bytesC := make(chan []byte)
	errC := make(chan error)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			errC <- err
			return
		}
		bytesC <- bytes
	}))

	g := core.NewGame()
	for !g.Result().Over() {
		plies := g.Plies()
		g.DoPly(plies[rand.Intn(len(plies))])
	}
	state := gameStateFrom(g, 1)

	notifyWebhooks(humanMode, id, state, []string{server.URL})

	select {
	case err := <-errC:
		t.Fatal(err)
	case bytes := <-bytesC:
		var body webhookRequestBody
		if err := json.Unmarshal(bytes, &body); err != nil {
			t.Fatal(err)
		}
	}
}
