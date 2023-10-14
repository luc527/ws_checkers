package main

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type wsTestClient struct {
	conn *websocket.Conn
	cli  *rawClient
}

func makeTestClient(t *testing.T) *wsTestClient {
	mux := http.NewServeMux()

	clients := make(chan *rawClient)

	upgrader := websocket.Upgrader{}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("failed to upgrade to websocket: %v", err)
			t.Fail()
		}
		clients <- websocketRawClient(conn)
	})

	server := httptest.NewServer(mux)
	url := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Logf("failed to connect to websocket server: %v", err)
		t.Fail()
		return nil
	}

	select {
	case <-time.After(1 * time.Second):
		t.Log("failed to receive in time")
		t.Fail()
		return nil
	case cli := <-clients:
		t.Log("client received successfully")
		return &wsTestClient{conn, cli}
	}
}

func TestServesWebsocket(t *testing.T) {
	makeTestClient(t)
}

func assertClosed(t *testing.T, c *rawClient) {
	_, ok := <-c.incoming
	if ok {
		t.Log("wanted incoming channel to be closed")
		t.Fail()
		return
	}

	select {
	case <-c.ended:
	default:
		t.Log("expected ended channel to be closed")
		t.Fail()
	}
}

func TestRawClient(t *testing.T) {
	wst := makeTestClient(t)
	conn, cli := wst.conn, wst.cli

	messages := []string{
		"{\"name\": \"testman\"}",
		"hello",
		"how are you",
		"goodbye",
		"",
		"this is a slightly longer message",
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		for _, m0 := range messages {
			<-time.After(1*time.Millisecond + time.Duration(rand.Intn(int(20*time.Millisecond))))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(m0)); err != nil {
				t.Logf("failed to write message: %v", err)
				t.Fail()
			}
			m1 := string(<-cli.incoming)
			t.Logf("incoming: want %q got %q", m0, m1)
			if m0 != m1 {
				t.Fail()
			}
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for _, m0 := range messages {
			<-time.After(1*time.Millisecond + time.Duration(rand.Intn(int(20*time.Millisecond))))
			cli.outgoing <- []byte(m0)
			_, bs, err := conn.ReadMessage()
			if err != nil {
				t.Logf("failed to read message: %v", err)
				t.Fail()
			}
			m1 := string(bs)
			t.Logf("outgoing: want %q got %q", m0, m1)
			if m0 != m1 {
				t.Fail()
			}
		}
		wg.Done()
	}()

	wg.Wait()

	conn.Close()

	assertClosed(t, cli)
}

func TestCloseOutgoing(t *testing.T) {
	tt := makeTestClient(t)
	_, cli := tt.conn, tt.cli

	close(cli.outgoing)

	assertClosed(t, cli)
}
