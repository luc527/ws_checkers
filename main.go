package main

import (
	"encoding/json"
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

		handleFirstAction(incoming, outgoing, stopSignal)
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

func trySend(outgoing chan<- []byte, v any) {
	bytes, err := json.Marshal(v)
	log.Printf("trying to send %q, err %v\n", string(bytes), err)
	if err == nil {
		outgoing <- bytes
	}
}

func handleFirstAction(incoming <-chan []byte, outgoing chan<- []byte, stopSignal chan struct{}) {
	timer := time.NewTimer(1 * time.Minute)
	defer timer.Stop()
outer:
	for {
		select {
		case <-timer.C:
			break outer
		case bytes, ok := <-incoming:
			if !ok {
				break outer
			}
			envelope := messageEnvelope{}
			if err := json.Unmarshal(bytes, &envelope); err != nil {
				trySend(outgoing, errorMessage("invalid message format: "+err.Error()))
				continue
			}
			switch envelope.T {
			case "new-game":
				trySend(outgoing, errorMessage("unimplemented"))
				break outer
			case "join-game":
				trySend(outgoing, errorMessage("unimplemented"))
				break outer
			case "reconnect-to-game":
				trySend(outgoing, errorMessage("unimplemented"))
				break outer
			default:
				trySend(outgoing, errorMessage("invalid message type at this point (first message)"))
				continue
			}
		}
	}
	close(stopSignal)
}
