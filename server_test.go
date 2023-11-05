package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/luc527/go_checkers/core"
)

func assertIncoming(t *testing.T, conn *websocket.Conn, cli *client, s string) {
	data := []byte(s)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Logf("error when writing to conn: %v", err)
		t.FailNow()
	}
	got, ok := <-cli.incoming
	if !ok {
		t.Log("failed to receive from incoming: channel closed prematurely")
		t.FailNow()
	}
	if slices.Compare(data, got) != 0 {
		t.Logf("expected %v, got %v", string(data), string(got))
		t.FailNow()
	}
}

func assertOutgoing(t *testing.T, conn *websocket.Conn, cli *client, s string) {
	data := []byte(s)
	cli.outgoing <- data
	_, got, err := conn.ReadMessage()
	if err != nil {
		t.Logf("error when reading from conn: %v", err)
		t.FailNow()
	}
	if slices.Compare(data, got) != 0 {
		t.Logf("expected %v, got %v", string(data), string(got))
		t.FailNow()
	}
}

func assertClosed(t *testing.T, cli *client) {
	var ok bool
	_, ok = <-cli.incoming
	if ok {
		t.Log("expected incoming channel to be closed")
		t.FailNow()
	}
}

func tryId(t *testing.T, id string) uuid.UUID {
	parsedId, err := uuid.Parse(id)
	if err != nil {
		t.Logf("invalid id %v", id)
		t.FailNow()
	}
	return parsedId
}

func tryType(t *testing.T, want string, got string) string {
	if want != got {
		t.Logf("wanted type %q got %q", want, got)
		t.FailNow()
	}
	return got
}

func tryColor(t *testing.T, c string) core.Color {
	if c == "white" {
		return core.WhiteColor
	}
	if c == "black" {
		return core.BlackColor
	}
	t.Logf("invalid color %q", c)
	t.FailNow()
	return 0
}

func getClientAndConn(t *testing.T) (client *client, conn *websocket.Conn) {
	upgrader := websocket.Upgrader{}
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			client = websocketRawClient(conn)
		}
		wg.Done()
	}))
	url := strings.ReplaceAll(server.URL, "http", "ws")
	conn, _, err = websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Logf("failed to dial: %v", err)
		t.FailNow()
	}
	wg.Wait()
	if client == nil {
		t.Log("failed to make client")
		t.FailNow()
	}
	return
}

func TestClient(t *testing.T) {
	cli, conn := getClientAndConn(t)

	assertIncoming(t, conn, cli, "hello world")
	assertIncoming(t, conn, cli, "hey answer me")
	assertOutgoing(t, conn, cli, "i am playing checkers")
	assertOutgoing(t, conn, cli, "with a friend")
	assertIncoming(t, conn, cli, "ok goodbye")
	close(cli.outgoing)
	assertClosed(t, cli)
}

func TestConnCloses(t *testing.T) {
	cli, conn := getClientAndConn(t)

	conn.Close()
	assertClosed(t, cli)
}
