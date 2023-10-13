package main

import (
	"log"
	"net/http"
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

var upgrader = websocket.Upgrader{}

type websocketServer struct {
	mux     *http.ServeMux
	clients chan *rawClient
}

func (wss *websocketServer) serve(addr string) {
	defer close(wss.clients)
	if err := http.ListenAndServe(addr, wss.mux); err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}

func newWebsocketServer() *websocketServer {
	mux := http.NewServeMux()
	clients := make(chan *rawClient)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		clients <- websocketRawClient(conn)
	})
	return &websocketServer{mux, clients}
}

func connReader(conn *websocket.Conn, incoming chan<- []byte, ended chan<- struct{}) {
	defer func() {
		conn.Close()
		close(incoming)
		close(ended)
	}()

	log.Println("connReader started")

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		log.Println("connReader pong")
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("connReader err: ", err)
			break
		}
		log.Println("connReader msg: ", string(msg))
		incoming <- msg
	}
}

func connWriter(conn *websocket.Conn, outgoing <-chan []byte, ended <-chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		conn.Close()
		ticker.Stop()
	}()

	log.Println("connWriter started")
	for {
		select {
		case <-ended:
			return
		case msg, ok := <-outgoing:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel has been closed
				if err := conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Println("connWriter error (2) ", err)
				}
				return
			}
			log.Println("connWriter msg ", string(msg))
			conn.WriteMessage(websocket.TextMessage, msg)
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("connWriter error (3) ", err)
				return
			}
		}
	}
}

func websocketRawClient(conn *websocket.Conn) *rawClient {
	incoming := make(chan []byte)
	outgoing := make(chan []byte)
	ended := make(chan struct{})

	go connReader(conn, incoming, ended)
	go connWriter(conn, outgoing, ended)

	return &rawClient{incoming, outgoing, ended}
}
