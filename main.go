package main

import (
	"flag"
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

func connReader(conn *websocket.Conn, incoming chan<- []byte, stopSignal <-chan struct{}) {
	defer func() {
		conn.Close()
		close(incoming)
	}()
	log.Println("connReader started")

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		log.Println("connReader pong")
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-stopSignal:
			return
		default:
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("connReader err: ", err)
			break
		}
		log.Println("connReader msg: ", string(msg))
		incoming <- msg
	}
}

func connWriter(conn *websocket.Conn, outgoing <-chan []byte, stopSignal chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		conn.Close()
		ticker.Stop()
	}()

	log.Println("connWriter started")
	for {
		select {
		case <-stopSignal:
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
			log.Println(err)
			return
		}

		incoming := make(chan []byte)
		outgoing := make(chan []byte)
		stopSignal := make(chan struct{})

		go connReader(conn, incoming, stopSignal)
		go connWriter(conn, outgoing, stopSignal)

		// TODO later we'll have to handle the first message separately
		// but for now let's always assume it starts a new game

		rc := rawClient{incoming, outgoing, stopSignal}
		gc := gameClient{&rc, make(chan error), make(chan gameState), make(chan plyRequest)}
		go gc.run()
	})

	httpServer := &http.Server{
		Addr:              *addr,
		ReadHeaderTimeout: 3 * time.Second,
	}
	err := httpServer.ListenAndServe()
	if err != nil {
		log.Println("ListenAndServe: ", err)
	}
}

func main() {
	runServer()
}
