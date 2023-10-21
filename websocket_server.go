package main

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

func connReader(conn *websocket.Conn, incoming chan<- []byte, ended chan<- struct{}) {
	defer func() {
		conn.Close()
		close(ended)
		close(incoming)
	}()

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		incoming <- msg
	}
}

func connWriter(conn *websocket.Conn, outgoing <-chan []byte, ended <-chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		conn.Close()
		ticker.Stop()
		// Drain channel so the goroutine sending to this channel doesn't block
		// (even if the goroutine checks for <-ended before sending, it's possible
		// for the connection to be closed just after the check). That goroutine
		// will be responsible for closing the channel, otherwise there'll be a
		// goroutine leak.
		for range outgoing {
		}
	}()

	for {
		select {
		case <-ended:
			return
		case msg, ok := <-outgoing:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			conn.WriteMessage(websocket.TextMessage, msg)
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func websocketRawClient(conn *websocket.Conn) *client {
	incoming := make(chan []byte)
	outgoing := make(chan []byte)
	ended := make(chan struct{})

	go connReader(conn, incoming, ended)
	go connWriter(conn, outgoing, ended)

	return &client{incoming, outgoing}
}
