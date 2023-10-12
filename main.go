package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", ":8080", "http service address")

var upgrader = websocket.Upgrader{}

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
			conn.WriteMessage(websocket.TextMessage, []byte(msg))
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("connWriter error (3) ", err)
				return
			}
		}
	}
}

func runServer() {
	flag.Parse()

	// Just for testing
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method now allowed", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrader: %v\n", err)
			return
		}

		incoming := make(chan []byte)
		outgoing := make(chan []byte)
		ended := make(chan struct{})

		go connReader(conn, incoming, ended)
		go connWriter(conn, outgoing, ended)

		raw := rawClient{incoming, outgoing, ended}
		raw.handleFirstMessage()
	})

	httpServer := &http.Server{
		Addr:              *addr,
		ReadHeaderTimeout: 3 * time.Second,
	}
	fmt.Println("Running server on localhost:8080")
	err := httpServer.ListenAndServe()
	if err != nil {
		log.Println("ListenAndServe: ", err)
	}
}

func main() {
	runServer()
}
